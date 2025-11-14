package notifier

// TransportConfig defines the configuration for a single notifier transport.
type TransportConfig struct {
	URL     string `validate:"required,url"`
	Enabled bool   `default:"true"`
}

// ChannelConfig defines a logical notification channel, which may
// use multiple transports to deliver messages.
type ChannelConfig struct {
	Description string
	Transports  []string `validate:"required,min=1"`
	Fallback    []string
}

// Config holds the complete notification system configuration,
// defining all available transports and logical channels.
type Config struct {
	Transports map[string]TransportConfig `validate:"required,min=1,dive"`
	Channels   map[string]ChannelConfig   `validate:"required,min=1,dive"`
}
