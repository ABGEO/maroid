package depresolver

import (
	"fmt"
	"sync"

	"github.com/abgeo/maroid/apps/hub/internal/auth"
)

// OIDCService initializes and returns the OIDC service instance.
func (c *Container) OIDCService() (*auth.OIDCService, error) {
	c.oidcService.mu.Lock()
	defer c.oidcService.mu.Unlock()

	var err error

	c.oidcService.once.Do(func() {
		c.oidcService.instance, err = auth.NewOIDCService(c.Config())
	})

	if err != nil {
		c.oidcService.once = sync.Once{}

		return nil, fmt.Errorf("initializing OIDC Service: %w", err)
	}

	return c.oidcService.instance, nil
}

// OIDCFlow initializes and returns the OIDC flow service instance.
func (c *Container) OIDCFlow() (*auth.OIDCFlow, error) {
	c.oidcFlow.mu.Lock()
	defer c.oidcFlow.mu.Unlock()

	var err error

	c.oidcFlow.once.Do(func() {
		var oidcSvc *auth.OIDCService

		oidcSvc, err = c.OIDCService()
		if err != nil {
			return
		}

		c.oidcFlow.instance = auth.NewOIDCFlow(oidcSvc)
	})

	if err != nil {
		c.oidcFlow.once = sync.Once{}

		return nil, fmt.Errorf("initializing OIDC flow: %w", err)
	}

	return c.oidcFlow.instance, nil
}

// JWTService initializes and returns the JWT service instance.
func (c *Container) JWTService() (*auth.JWTService, error) {
	c.jwtService.mu.Lock()
	defer c.jwtService.mu.Unlock()

	var err error

	c.jwtService.once.Do(func() {
		c.jwtService.instance, err = auth.NewJWTService(c.Config())
	})

	if err != nil {
		c.jwtService.once = sync.Once{}

		return nil, fmt.Errorf("initializing JWT Service: %w", err)
	}

	return c.jwtService.instance, nil
}
