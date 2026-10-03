package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/repository"
	"github.com/abgeo/maroid/apps/hub/internal/user"
)

// UserService initializes and returns the service of the user records.
func (c *Container) UserService() (user.Service, error) {
	c.userService.mu.Lock()
	defer c.userService.mu.Unlock()

	var err error

	c.userService.once.Do(func() {
		c.userService.instance, err = c.buildUserService()
	})

	if err != nil {
		c.userService.once = sync.Once{}

		return nil, fmt.Errorf("initializing user service: %w", err)
	}

	return c.userService.instance, nil
}

func (c *Container) buildUserService() (*user.Manager, error) {
	dbInstance, err := c.Database()
	if err != nil {
		return nil, err
	}

	users, err := c.UserRepository()
	if err != nil {
		return nil, err
	}

	authService, err := c.AuthService()
	if err != nil {
		return nil, err
	}

	return user.NewManager(
		dbInstance,
		user.Repositories{
			Users:   users,
			Allowed: repository.NewAllowedPlugin(dbInstance),
		},
		authService,
		c.PluginRegistry(),
		c.Config().Auth.InvitationTTL,
	), nil
}
