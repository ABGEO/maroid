package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/handler"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
)

// WorkspaceService initializes and returns the service of the workspaces. It answers
// the manager, which a caller takes as workspace.Service or as workspace.Catalog.
func (c *Container) WorkspaceService() (*workspace.Manager, error) {
	c.workspaceService.mu.Lock()
	defer c.workspaceService.mu.Unlock()

	var err error

	c.workspaceService.once.Do(func() {
		var dbInstance *sqlx.DB

		if dbInstance, err = c.Database(); err != nil {
			return
		}

		c.workspaceService.instance = workspace.NewManager(dbInstance)
	})

	if err != nil {
		c.workspaceService.once = sync.Once{}

		return nil, fmt.Errorf("initializing workspace service: %w", err)
	}

	return c.workspaceService.instance, nil
}

// ChatSelection initializes and returns the selection of the workspace of each chat.
func (c *Container) ChatSelection() (*workspace.ChatSelection, error) {
	c.chatSelection.mu.Lock()
	defer c.chatSelection.mu.Unlock()

	var err error

	c.chatSelection.once.Do(func() {
		var dbInstance *sqlx.DB

		if dbInstance, err = c.Database(); err != nil {
			return
		}

		c.chatSelection.instance = workspace.NewChatSelection(dbInstance)
	})

	if err != nil {
		c.chatSelection.once = sync.Once{}

		return nil, fmt.Errorf("initializing chat selection: %w", err)
	}

	return c.chatSelection.instance, nil
}

// EnablementService initializes and returns the service of the enablements of a plugin.
func (c *Container) EnablementService() (*workspace.Enablements, error) {
	c.enablementService.mu.Lock()
	defer c.enablementService.mu.Unlock()

	var err error

	c.enablementService.once.Do(func() {
		var dbInstance *sqlx.DB

		if dbInstance, err = c.Database(); err != nil {
			return
		}

		c.enablementService.instance = workspace.NewEnablements(dbInstance, c.PluginRegistry())
	})

	if err != nil {
		c.enablementService.once = sync.Once{}

		return nil, fmt.Errorf("initializing enablement service: %w", err)
	}

	return c.enablementService.instance, nil
}

// workspaceAccess resolves the three checks that a route of a plugin passes.
func (c *Container) workspaceAccess() (handler.WorkspaceAccess, error) {
	dbInstance, err := c.Database()
	if err != nil {
		return handler.WorkspaceAccess{}, err
	}

	enablements, err := c.EnablementService()
	if err != nil {
		return handler.WorkspaceAccess{}, err
	}

	authorizer, err := c.Authorizer()
	if err != nil {
		return handler.WorkspaceAccess{}, err
	}

	return handler.WorkspaceAccess{
		DB:          dbInstance,
		Enablements: enablements,
		Authorizer:  authorizer,
	}, nil
}
