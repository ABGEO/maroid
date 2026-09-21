package registry

// APIRoute is one route that a plugin serves.
type APIRoute struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

// CLICommand is one command that a plugin adds to the command tree.
type CLICommand struct {
	Command string `json:"command"`
}

// CronJob is one job that a plugin runs on a schedule.
type CronJob struct {
	ID       string `json:"id"`
	Schedule string `json:"schedule"`
}

// MQTTSubscriber is one topic that a plugin subscribes to.
type MQTTSubscriber struct {
	ID    string `json:"id"`
	Topic string `json:"topic"`
}

// TelegramCommand is one bot command that a plugin answers.
type TelegramCommand struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// TelegramConversation is one conversation that a plugin drives.
type TelegramConversation struct {
	ID    string `json:"id"`
	Entry string `json:"entry"`
}

// MCPToolItem is one tool that a plugin exposes over the Model Context Protocol.
type MCPToolItem struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
