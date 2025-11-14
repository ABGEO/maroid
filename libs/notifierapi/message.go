package notifierapi

// Attachment represents a file or binary data to be sent along with a message.
type Attachment struct {
	Filename string
	Content  []byte
	MIMEType string
}

// Message defines the structure of a notification message.
type Message struct {
	Title       string
	Body        string
	Attachments []Attachment
}
