// Package telegram implements a notifier transport for sending messages
// via the Telegram Bot API using the telego library.
package telegram

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/notifier/registry"
	"github.com/abgeo/maroid/libs/notifierapi"
)

const (
	queryParamTopic = "x-topic"
	queryParamDebug = "x-debug"
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
	var options []telego.BotOption
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
	_, err := n.client.SendMessage(ctx, &telego.SendMessageParams{
		ChatID:          n.config.ChatID,
		Text:            formatMessageText(msg),
		MessageThreadID: n.config.ChatTopicID,
		ParseMode:       telego.ModeHTML,
	})
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	return nil
}

func (n *Notifier) sendSingleAttachment(ctx context.Context, msg notifierapi.Message) error {
	attachment := n.normalizeAttachment(msg.Attachments[0])
	caption := formatMessageText(msg)

	if isImage(attachment) {
		return n.sendPhoto(ctx, attachment, caption)
	}

	return n.sendDocument(ctx, attachment, caption)
}

func (n *Notifier) sendPhoto(
	ctx context.Context,
	attachment notifierapi.Attachment,
	caption string,
) error {
	_, err := n.client.SendPhoto(ctx, &telego.SendPhotoParams{
		ChatID:          n.config.ChatID,
		Photo:           tu.FileFromBytes(attachment.Content, attachment.Filename),
		Caption:         caption,
		ParseMode:       telego.ModeHTML,
		MessageThreadID: n.config.ChatTopicID,
	})
	if err != nil {
		return fmt.Errorf("failed to send photo: %w", err)
	}

	return nil
}

func (n *Notifier) sendDocument(
	ctx context.Context,
	attachment notifierapi.Attachment,
	caption string,
) error {
	_, err := n.client.SendDocument(ctx, &telego.SendDocumentParams{
		ChatID:          n.config.ChatID,
		Document:        tu.FileFromBytes(attachment.Content, attachment.Filename),
		Caption:         caption,
		ParseMode:       telego.ModeHTML,
		MessageThreadID: n.config.ChatTopicID,
	})
	if err != nil {
		return fmt.Errorf("failed to send document: %w", err)
	}

	return nil
}

func (n *Notifier) sendMultipleAttachments(ctx context.Context, msg notifierapi.Message) error {
	photos, documents := n.groupAttachmentsByType(msg.Attachments)

	hasPhotos := len(photos) > 0
	hasDocuments := len(documents) > 0

	// Single media type: send with caption
	if hasPhotos && !hasDocuments {
		return n.sendMediaGroupWithCaption(ctx, photos, formatMessageText(msg), "photo")
	}

	if hasDocuments && !hasPhotos {
		return n.sendMediaGroupWithCaption(ctx, documents, formatMessageText(msg), "document")
	}

	// Multiple media types: send text first, then groups separately
	if err := n.sendTextMessage(ctx, msg); err != nil {
		return err
	}

	if err := n.sendMediaGroup(ctx, photos, "photo"); err != nil {
		return err
	}

	if err := n.sendMediaGroup(ctx, documents, "document"); err != nil {
		return err
	}

	return nil
}

func (n *Notifier) groupAttachmentsByType(
	attachments []notifierapi.Attachment,
) ([]telego.InputMedia, []telego.InputMedia) {
	var (
		photos    []telego.InputMedia
		documents []telego.InputMedia
	)

	for _, attachment := range attachments {
		attachment = n.normalizeAttachment(attachment)

		if isImage(attachment) {
			photos = append(photos, n.createPhotoMedia(attachment))
		} else {
			documents = append(documents, n.createDocumentMedia(attachment))
		}
	}

	return photos, documents
}

func (n *Notifier) sendMediaGroup(
	ctx context.Context,
	media []telego.InputMedia,
	mediaType string,
) error {
	if len(media) == 0 {
		return nil
	}

	_, err := n.client.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
		ChatID:          n.config.ChatID,
		Media:           media,
		MessageThreadID: n.config.ChatTopicID,
	})
	if err != nil {
		return fmt.Errorf("failed to send %s media group: %w", mediaType, err)
	}

	return nil
}

func (n *Notifier) sendMediaGroupWithCaption(
	ctx context.Context,
	media []telego.InputMedia,
	caption, mediaType string,
) error {
	if len(media) == 0 {
		return nil
	}

	// Add caption to the first media item
	switch firstMedia := media[0].(type) {
	case *telego.InputMediaPhoto:
		firstMedia.Caption = caption
		firstMedia.ParseMode = telego.ModeHTML
	case *telego.InputMediaDocument:
		firstMedia.Caption = caption
		firstMedia.ParseMode = telego.ModeHTML
	}

	_, err := n.client.SendMediaGroup(ctx, &telego.SendMediaGroupParams{
		ChatID:          n.config.ChatID,
		Media:           media,
		MessageThreadID: n.config.ChatTopicID,
	})
	if err != nil {
		return fmt.Errorf("failed to send %s media group: %w", mediaType, err)
	}

	return nil
}

func (n *Notifier) normalizeAttachment(att notifierapi.Attachment) notifierapi.Attachment {
	if att.MIMEType == "" {
		att.MIMEType = http.DetectContentType(att.Content)
	}

	if att.Filename == "" {
		att.Filename = "attachment"
	}

	return att
}

func isImage(attachment notifierapi.Attachment) bool {
	return strings.HasPrefix(attachment.MIMEType, "image/")
}

func (n *Notifier) createPhotoMedia(attachment notifierapi.Attachment) *telego.InputMediaPhoto {
	return &telego.InputMediaPhoto{
		Type:  telego.MediaTypePhoto,
		Media: tu.FileFromBytes(attachment.Content, attachment.Filename),
	}
}

func (n *Notifier) createDocumentMedia(
	attachment notifierapi.Attachment,
) *telego.InputMediaDocument {
	return &telego.InputMediaDocument{
		Type:  telego.MediaTypeDocument,
		Media: tu.FileFromBytes(attachment.Content, attachment.Filename),
	}
}

func formatMessageText(msg notifierapi.Message) string {
	text := ""

	if msg.Title != "" {
		text += "<b>" + msg.Title + "</b>"
	}

	if msg.Title != "" && msg.Body != "" {
		text += "\n"
	}

	if msg.Body != "" {
		text += msg.Body
	}

	return text
}
