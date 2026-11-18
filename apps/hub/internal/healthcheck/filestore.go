package healthcheck

import (
	"context"
	"fmt"

	"github.com/hellofresh/health-go/v5"
)

// Pinger answers whether the file store serves the bucket of the hub.
type Pinger interface {
	Ping(ctx context.Context) error
}

// fileStoreCheck checks that the file store answers for the bucket of the hub. A
// signed address never reaches the file store, so only this check sees it down.
func fileStoreCheck(store Pinger) health.Config {
	return limited("file-store", func(ctx context.Context) error {
		if err := store.Ping(ctx); err != nil {
			return fmt.Errorf("reaching the file store: %w", err)
		}

		return nil
	})
}
