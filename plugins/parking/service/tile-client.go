package service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"resty.dev/v3"

	"github.com/abgeo/maroid/plugins/parking/config"
	"github.com/abgeo/maroid/plugins/parking/errs"
)

const (
	// tileUserAgent names the client, as the usage policy of the default map source demands.
	tileUserAgent = "Maroid/1.0 (parking map)"
	maxTileBytes  = 1 << 20
)

// TileSource reads one map tile as PNG bytes.
type TileSource interface {
	Tile(ctx context.Context, zoom, x, y int) ([]byte, error)
}

// TileClient reads the map tiles from the map source of the configuration.
type TileClient struct {
	client      *resty.Client
	urlTemplate string
}

var _ TileSource = (*TileClient)(nil)

// NewTileClient creates a TileClient. It sends no credential of a person.
func NewTileClient(cfg config.MapConfig) *TileClient {
	client := resty.New().
		SetHeaders(map[string]string{
			"User-Agent": tileUserAgent,
			"Accept":     "image/png",
		}).
		SetResponseBodyLimit(maxTileBytes)

	if cfg.Timeout > 0 {
		client.SetTimeout(cfg.Timeout)
	}

	return &TileClient{client: client, urlTemplate: cfg.TileURL}
}

// Tile reads the tile at the zoom and the tile coordinates.
func (c *TileClient) Tile(ctx context.Context, zoom, column, row int) ([]byte, error) {
	address := strings.NewReplacer(
		"{z}", strconv.Itoa(zoom),
		"{x}", strconv.Itoa(column),
		"{y}", strconv.Itoa(row),
	).Replace(c.urlTemplate)

	resp, err := c.client.R().SetContext(ctx).Get(address)
	if err != nil {
		if requestErr, isRequestErr := errors.AsType[*url.Error](err); isRequestErr {
			// The address can carry a key of the map source.
			err = requestErr.Err
		}

		return nil, fmt.Errorf(
			"%w: reading the tile %d/%d/%d: %w",
			errs.ErrMapSource,
			zoom,
			column,
			row,
			err,
		)
	}

	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf(
			"%w: the tile %d/%d/%d returned [%d]",
			errs.ErrMapSource, zoom, column, row, resp.StatusCode(),
		)
	}

	return resp.Bytes(), nil
}
