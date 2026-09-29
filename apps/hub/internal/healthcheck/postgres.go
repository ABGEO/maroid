package healthcheck

import (
	"context"
	"fmt"

	"github.com/hellofresh/health-go/v5"
	"github.com/jmoiron/sqlx"
)

// databaseCheck checks that the database answers a ping.
func databaseCheck(db *sqlx.DB) health.Config {
	return limited("database", func(ctx context.Context) error {
		if err := db.PingContext(ctx); err != nil {
			return fmt.Errorf("pinging the database: %w", err)
		}

		return nil
	})
}
