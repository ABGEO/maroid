// Package loader provides functionality to dynamically load plugins.
package loader

import (
	"fmt"
	"plugin"

	"github.com/abgeo/maroid/apps/hub/internal/domain/errs"
	"github.com/abgeo/maroid/libs/pluginapi"
)

// ConstructorSymbol is the name of the exported constructor symbol
// expected in each plugin.
const ConstructorSymbol = "New"

// LoadPlugin opens, initializes, and validates a plugin from the given path.
// It uses the provided host and configuration map for plugin initialization.
func LoadPlugin(path string, host pluginapi.Host, cfg map[string]any) (pluginapi.Plugin, error) {
	constructor, err := openConstructor(path)
	if err != nil {
		return nil, err
	}

	plg, err := constructor(host, cfg)
	if err != nil {
		return nil, err
	}

	if err = validatePlugin(plg); err != nil {
		return nil, err
	}

	return plg, nil
}

func openConstructor(path string) (pluginapi.Constructor, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open plugin %q: %w", path, err)
	}

	symbol, err := p.Lookup(ConstructorSymbol)
	if err != nil {
		return nil, fmt.Errorf(
			"cannot find constructor in plugin %q: %w",
			path,
			err,
		)
	}

	constructor, ok := symbol.(*pluginapi.Constructor)
	if !ok {
		return nil, fmt.Errorf(
			"%w: %q (%s)",
			errs.ErrUnexpectedPluginSymbolType,
			ConstructorSymbol,
			path,
		)
	}

	return *constructor, nil
}

func validatePlugin(plg pluginapi.Plugin) error {
	meta := plg.Meta()

	if meta.ID == nil {
		return errs.ErrInvalidPluginID
	}

	if meta.APIVersion != pluginapi.APIVersion {
		return fmt.Errorf("%w: plugin %q built for API %s, expected %s",
			errs.ErrIncompatiblePluginAPIVersion, meta.ID, meta.APIVersion, pluginapi.APIVersion)
	}

	return nil
}
