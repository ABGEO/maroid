package depresolver

import (
	"fmt"
	"sync"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/workspace"
)

// WorkspaceMemberRepository initializes and returns the membership repository.
func (c *Container) WorkspaceMemberRepository() (repository.WorkspaceMemberRepository, error) {
	c.workspaceMemberRepository.mu.Lock()
	defer c.workspaceMemberRepository.mu.Unlock()

	var err error

	c.workspaceMemberRepository.once.Do(func() {
		var dbInstance *sqlx.DB

		dbInstance, err = c.Database()
		if err != nil {
			return
		}

		c.workspaceMemberRepository.instance = repository.NewWorkspaceMember(dbInstance)
	})

	if err != nil {
		c.workspaceMemberRepository.once = sync.Once{}

		return nil, fmt.Errorf("initializing workspace member repository: %w", err)
	}

	return c.workspaceMemberRepository.instance, nil
}

// WorkspaceService initializes and returns the service of the workspaces.
func (c *Container) WorkspaceService() (workspace.Service, error) {
	c.workspaceService.mu.Lock()
	defer c.workspaceService.mu.Unlock()

	var err error

	c.workspaceService.once.Do(func() {
		var (
			dbInstance *sqlx.DB
			members    repository.WorkspaceMemberRepository
			users      repository.UserRepository
		)

		if dbInstance, err = c.Database(); err != nil {
			return
		}

		if members, err = c.WorkspaceMemberRepository(); err != nil {
			return
		}

		if users, err = c.UserRepository(); err != nil {
			return
		}

		c.workspaceService.instance = workspace.NewManager(
			dbInstance,
			repository.NewWorkspace(dbInstance),
			members,
			users,
		)
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
		var (
			dbInstance *sqlx.DB
			members    repository.WorkspaceMemberRepository
		)

		if dbInstance, err = c.Database(); err != nil {
			return
		}

		if members, err = c.WorkspaceMemberRepository(); err != nil {
			return
		}

		c.chatSelection.instance = workspace.NewChatSelection(
			dbInstance,
			repository.NewWorkspace(dbInstance),
			members,
		)
	})

	if err != nil {
		c.chatSelection.once = sync.Once{}

		return nil, fmt.Errorf("initializing chat selection: %w", err)
	}

	return c.chatSelection.instance, nil
}
