// Package telegram implements a notifier transport for sending messages
// via the Telegram Bot API using the telego library.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mymmrac/telego"
	ta "github.com/mymmrac/telego/telegoapi"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/notifier/registry"
	"github.com/abgeo/maroid/libs/notifierapi"
)

const (
	queryParamTopic = "x-topic"
	queryParamDebug = "x-debug"

	retryMaxAttempts  = 4
	retryExponentBase = 2
	retryStartDelay   = time.Millisecond * 10
	retryMaxDelay     = time.Second
)

var (
	// ErrMissingCredentials is returned when the URL is missing required token or chat ID.
	ErrMissingCredentials = errors.New("telegram: missing token or chat_id in URL")
	// ErrInvalidChatID is returned when the chat ID cannot be parsed as a valid integer.
	ErrInvalidChatID = errors.New("telegram: invalid chat ID format")
	// ErrInvalidTopicID is returned when the topic ID cannot be parsed as a valid integer.
	ErrInvalidTopicID = errors.New("telegram: invalid topic ID format")
)

// Config holds the configuration for the Telegram notifier.
type Config struct {
	Debug       bool
	Token       string
	ChatID      telego.ChatID
	ChatTopicID int
}

// Notifier implements the notifierapi.Transport interface for Telegram.
type Notifier struct {
	config *Config
	client *telego.Bot
}

var _ notifierapi.Transport = (*Notifier)(nil)

// New creates a new Telegram notifier from a URL configuration.
// URL format: telegram://TOKEN@CHAT_ID?x-topic=TOPIC_ID&x-debug=true
func New(rawURL *url.URL) (notifierapi.Transport, error) {
	cfg, err := parseConfiguration(rawURL)
	if err != nil {
		return nil, err
	}

	bot, err := createBot(cfg)
	if err != nil {
		return nil, err
	}

	return &Notifier{
		client: bot,
		config: cfg,
	}, nil
}

// Register registers the Telegram notifier with the given registry.
func Register(reg registry.Registry) error {
	if err := reg.Register("telegram", New); err != nil {
		return fmt.Errorf("failed to register telegram notifier: %w", err)
	}

	return nil
}

// Send sends a message through Telegram based on attachment count and type.
func (n *Notifier) Send(ctx context.Context, msg notifierapi.Message) error {
	switch len(msg.Attachments) {
	case 0:
		return n.sendTextMessage(ctx, msg)
	case 1:
		return n.sendSingleAttachment(ctx, msg)
	default:
		return n.sendMultipleAttachments(ctx, msg)
	}
}

func parseConfiguration(rawURL *url.URL) (*Config, error) {
	token := rawURL.User.String()
	chatIDRaw := rawURL.Host

	if token == "" || chatIDRaw == "" {
		return nil, ErrMissingCredentials
	}

	chatID, err := parseChatID(chatIDRaw)
	if err != nil {
		return nil, err
	}

	topicID, err := parseTopicID(rawURL.Query().Get(queryParamTopic))
	if err != nil {
		return nil, err
	}

	debug := rawURL.Query().Get(queryParamDebug) == "true"

	return &Config{
		Debug:       debug,
		Token:       token,
		ChatID:      tu.ID(chatID),
		ChatTopicID: topicID,
	}, nil
}

func createBot(cfg *Config) (*telego.Bot, error) {
	options := []telego.BotOption{
		telego.WithAPICaller(&ta.RetryCaller{
			Caller:       ta.DefaultFastHTTPCaller,
			MaxAttempts:  retryMaxAttempts,
			ExponentBase: retryExponentBase,
			StartDelay:   retryStartDelay,
			MaxDelay:     retryMaxDelay,
		}),
	}

	if cfg.Debug {
		options = append(options, telego.WithDefaultDebugLogger())
	}

	bot, err := telego.NewBot(cfg.Token, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize telegram bot: %w", err)
	}

	return bot, nil
}

func parseChatID(raw string) (int64, error) {
	chatID, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidChatID, err)
	}

	return chatID, nil
}

func parseTopicID(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}

	topicID, err := strconv.Atoi(raw)
	if err != nil {
		return 0, fmt.Errorf("%w: %w", ErrInvalidTopicID, err)
	}

	return topicID, nil
}

func (n *Notifier) sendTextMessage(ctx context.Context, msg notifierapi.Message) error {
	message := tu.Message(n.config.ChatID, formatMessageText(msg)).WithParseMode(telego.ModeHTML)

	if n.config.ChatTopicID != 0 {
		message.WithMessageThreadID(n.config.ChatTopicID)
	}

	_, err := n.client.SendMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (n *Notifier) sendSingleAttachment(ctx context.Context, msg notifierapi.Message) error {
	attachment := normalizeAttachment(msg.Attachments[0])
	file := tu.FileFromBytes(attachment.Content, attachment.Filename)
	caption := formatMessageText(msg)

	if isImage(attachment) {
		return n.sendPhoto(ctx, file, caption)
	}

	return n.sendDocument(ctx, file, caption)
}

func (n *Notifier) sendPhoto(ctx context.Context, file telego.InputFile, caption string) error {
	photo := tu.Photo(n.config.ChatID, file)

	if caption != "" {
		photo.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}

	if n.config.ChatTopicID != 0 {
		photo.WithMessageThreadID(n.config.ChatTopicID)
	}

	_, err := n.client.SendPhoto(ctx, photo)
	if err != nil {
		return fmt.Errorf("failed to send photo: %w", err)
	}

	return nil
}

func (n *Notifier) sendDocument(ctx context.Context, file telego.InputFile, caption string) error {
	document := tu.Document(n.config.ChatID, file)

	if caption != "" {
		document.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}

	if n.config.ChatTopicID != 0 {
		document.WithMessageThreadID(n.config.ChatTopicID)
	}

	_, err := n.client.SendDocument(ctx, document)
	if err != nil {
		return fmt.Errorf("failed to send document: %w", err)
	}

	return nil
}

func (n *Notifier) sendMediaGroup(
	ctx context.Context,
	media []telego.InputMedia,
	caption string,
) error {
	if len(media) == 0 {
		return nil
	}

	mediaGroup := tu.MediaGroup(n.config.ChatID, media...)

	if caption != "" {
		setMediaCaption(mediaGroup.Media[0], caption)
	}

	if n.config.ChatTopicID != 0 {
		mediaGroup.WithMessageThreadID(n.config.ChatTopicID)
	}

	_, err := n.client.SendMediaGroup(ctx, mediaGroup)
	if err != nil {
		return fmt.Errorf("failed to send media group: %w", err)
	}

	return nil
}

func (n *Notifier) sendMultipleAttachments(ctx context.Context, msg notifierapi.Message) error {
	grouped := n.groupAttachmentsByType(msg.Attachments)
	text := formatMessageText(msg)

	// Send text separately if we have mixed attachment types
	if text != "" && len(slices.Collect(maps.Keys(grouped))) > 1 {
		if err := n.sendTextMessage(ctx, msg); err != nil {
			return err
		}

		text = ""
	}

	for _, mediaList := range grouped {
		if err := n.sendMediaGroup(ctx, mediaList, text); err != nil {
			return err
		}
	}

	return nil
}

func (n *Notifier) groupAttachmentsByType(
	attachments []notifierapi.Attachment,
) map[string][]telego.InputMedia {
	grouped := make(map[string][]telego.InputMedia)

	for _, attachment := range attachments {
		attachment = normalizeAttachment(attachment)
		file := tu.FileFromBytes(attachment.Content, attachment.Filename)

		if isImage(attachment) {
			grouped["photos"] = append(grouped["photos"], tu.MediaPhoto(file))
		} else {
			grouped["documents"] = append(grouped["documents"], tu.MediaDocument(file))
		}
	}

	return grouped
}

func setMediaCaption(media telego.InputMedia, caption string) {
	switch m := media.(type) {
	case *telego.InputMediaPhoto:
		m.WithCaption(caption).WithParseMode(telego.ModeHTML)
	case *telego.InputMediaDocument:
		m.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}
}

func normalizeAttachment(attachments notifierapi.Attachment) notifierapi.Attachment {
	if attachments.MIMEType == "" {
		attachments.MIMEType = http.DetectContentType(attachments.Content)
	}

	if attachments.Filename == "" {
		attachments.Filename = "attachment"
	}

	return attachments
}

func isImage(att notifierapi.Attachment) bool {
	return strings.HasPrefix(att.MIMEType, "image/")
}

func formatMessageText(msg notifierapi.Message) string {
	var parts []string

	if msg.Title != "" {
		parts = append(parts, "<b>"+msg.Title+"</b>")
	}

	if msg.Body != "" {
		parts = append(parts, msg.Body)
	}

	return strings.Join(parts, "\n\n")
}
