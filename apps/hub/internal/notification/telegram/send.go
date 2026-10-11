package telegram

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"

	"github.com/abgeo/maroid/libs/pluginapi"
)

// delivery sends one message through one bot to one target.
type delivery struct {
	bot    *telego.Bot
	target target
}

// send picks the Bot API method from the count and the type of the attachments.
func (d *delivery) send(ctx context.Context, msg pluginapi.Message) error {
	switch len(msg.Attachments) {
	case 0:
		return d.sendTextMessage(ctx, msg)
	case 1:
		return d.sendSingleAttachment(ctx, msg)
	default:
		return d.sendMultipleAttachments(ctx, msg)
	}
}

func (d *delivery) sendTextMessage(ctx context.Context, msg pluginapi.Message) error {
	message := tu.Message(d.target.chat, formatMessageText(msg)).WithParseMode(telego.ModeHTML)

	if d.target.thread != 0 {
		message.WithMessageThreadID(d.target.thread)
	}

	_, err := d.bot.SendMessage(ctx, message)
	if err != nil {
		return fmt.Errorf("sending message: %w", err)
	}

	return nil
}

func (d *delivery) sendSingleAttachment(ctx context.Context, msg pluginapi.Message) error {
	attachment := normalizeAttachment(msg.Attachments[0])
	file := tu.FileFromBytes(attachment.Content, attachment.Filename)
	caption := formatMessageText(msg)

	if isImage(attachment) {
		return d.sendPhoto(ctx, file, caption)
	}

	return d.sendDocument(ctx, file, caption)
}

func (d *delivery) sendPhoto(ctx context.Context, file telego.InputFile, caption string) error {
	photo := tu.Photo(d.target.chat, file)

	if caption != "" {
		photo.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}

	if d.target.thread != 0 {
		photo.WithMessageThreadID(d.target.thread)
	}

	_, err := d.bot.SendPhoto(ctx, photo)
	if err != nil {
		return fmt.Errorf("sending photo: %w", err)
	}

	return nil
}

func (d *delivery) sendDocument(ctx context.Context, file telego.InputFile, caption string) error {
	document := tu.Document(d.target.chat, file)

	if caption != "" {
		document.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}

	if d.target.thread != 0 {
		document.WithMessageThreadID(d.target.thread)
	}

	_, err := d.bot.SendDocument(ctx, document)
	if err != nil {
		return fmt.Errorf("sending document: %w", err)
	}

	return nil
}

func (d *delivery) sendMediaGroup(
	ctx context.Context,
	media []telego.InputMedia,
	caption string,
) error {
	if len(media) == 0 {
		return nil
	}

	mediaGroup := tu.MediaGroup(d.target.chat, media...)

	if caption != "" {
		setMediaCaption(mediaGroup.Media[0], caption)
	}

	if d.target.thread != 0 {
		mediaGroup.WithMessageThreadID(d.target.thread)
	}

	_, err := d.bot.SendMediaGroup(ctx, mediaGroup)
	if err != nil {
		return fmt.Errorf("sending media group: %w", err)
	}

	return nil
}

func (d *delivery) sendMultipleAttachments(ctx context.Context, msg pluginapi.Message) error {
	groups := groupByMedia(msg.Attachments)
	text := formatMessageText(msg)

	// Telegram puts a caption on a media group of one type only, so a message that mixes
	// photos and documents sends its text on its own first.
	if text != "" && len(groups) > 1 {
		if err := d.sendTextMessage(ctx, msg); err != nil {
			return err
		}

		text = ""
	}

	for _, media := range groups {
		if err := d.sendMediaGroup(ctx, media, text); err != nil {
			return err
		}
	}

	return nil
}

// groupByMedia answers the photos, then the documents, and leaves out an empty group.
func groupByMedia(attachments []pluginapi.Attachment) [][]telego.InputMedia {
	var photos, documents []telego.InputMedia

	for _, attachment := range attachments {
		attachment = normalizeAttachment(attachment)
		file := tu.FileFromBytes(attachment.Content, attachment.Filename)

		if isImage(attachment) {
			photos = append(photos, tu.MediaPhoto(file))
		} else {
			documents = append(documents, tu.MediaDocument(file))
		}
	}

	groups := make([][]telego.InputMedia, 0, 2) //nolint:mnd // one group for each media type

	for _, group := range [][]telego.InputMedia{photos, documents} {
		if len(group) > 0 {
			groups = append(groups, group)
		}
	}

	return groups
}

func setMediaCaption(media telego.InputMedia, caption string) {
	switch m := media.(type) {
	case *telego.InputMediaPhoto:
		m.WithCaption(caption).WithParseMode(telego.ModeHTML)
	case *telego.InputMediaDocument:
		m.WithCaption(caption).WithParseMode(telego.ModeHTML)
	}
}

func normalizeAttachment(attachment pluginapi.Attachment) pluginapi.Attachment {
	if attachment.MIMEType == "" {
		attachment.MIMEType = http.DetectContentType(attachment.Content)
	}

	if attachment.Filename == "" {
		attachment.Filename = "attachment"
	}

	return attachment
}

func isImage(attachment pluginapi.Attachment) bool {
	return strings.HasPrefix(attachment.MIMEType, "image/")
}

func formatMessageText(msg pluginapi.Message) string {
	var parts []string

	if msg.Title != "" {
		parts = append(parts, "<b>"+msg.Title+"</b>")
	}

	if msg.Body != "" {
		parts = append(parts, msg.Body)
	}

	return strings.Join(parts, "\n\n")
}
