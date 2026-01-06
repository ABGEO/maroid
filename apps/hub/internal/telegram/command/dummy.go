package command

import (
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
)

// DummyCommand provides a base implementation of the Command interface.
type DummyCommand struct{}

// Scope dummy implementation returns ScopeDefault.
func (c *DummyCommand) Scope() telego.BotCommandScope { //nolint:ireturn
	return tu.ScopeDefault()
}

// Validate dummy implementation always returns nil.
func (c *DummyCommand) Validate(_ telego.Update) error {
	return nil
}
