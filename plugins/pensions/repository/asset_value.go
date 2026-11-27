package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// AssetValueRepository defines the data access contract for the asset values of the
// acting user.
type AssetValueRepository interface {
	Upsert(ctx context.Context, value *model.AssetValue) error
	Since(ctx context.Context, seek page.Seek, from *time.Time) ([]model.AssetValue, error)
}

// AssetValue reads and writes the asset values of the acting user within one
// transaction of the database.
type AssetValue struct {
	tx *sqlx.Tx
}

var _ AssetValueRepository = (*AssetValue)(nil)

// NewAssetValue creates a new AssetValue over the transaction of the database.
func NewAssetValue(tx *sqlx.Tx) *AssetValue {
	return &AssetValue{tx: tx}
}

// Upsert stores the asset value of its date, and replaces a held value of that date.
func (r *AssetValue) Upsert(ctx context.Context, value *model.AssetValue) error {
	const query = `
		INSERT INTO asset_values (valued_on, asset_value, units, unit_price)
		VALUES (CAST(:valued_on AS date), :asset_value, :units, :unit_price)
		ON CONFLICT ON CONSTRAINT asset_values_user_valued_on_key DO UPDATE
		SET asset_value  = EXCLUDED.asset_value,
		    units        = EXCLUDED.units,
		    unit_price   = EXCLUDED.unit_price,
		    collected_at = now()
		WHERE (asset_values.asset_value, asset_values.units, asset_values.unit_price)
		      IS DISTINCT FROM
		      (EXCLUDED.asset_value, EXCLUDED.units, EXCLUDED.unit_price);`

	if _, err := r.tx.NamedExecContext(ctx, query, value); err != nil {
		return fmt.Errorf("upserting the asset value: %w", err)
	}

	return nil
}

// Since answers one keyset read of the asset values of the acting user from the date, the
// latest first, with the identifier as the tie. A nil from reads every asset value.
func (r *AssetValue) Since(
	ctx context.Context,
	seek page.Seek,
	from *time.Time,
) ([]model.AssetValue, error) {
	condition, sorting := keyset(seek, "asset_values", "valued_on")
	query := `
		SELECT t.id, t.valued_on, t.asset_value, t.units, t.unit_price
		FROM asset_values t
		WHERE (CAST(:from AS date) IS NULL OR t.valued_on >= CAST(:from AS date))
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	values, err := selectNamed[model.AssetValue](ctx, r.tx, query, seekArguments(seek, from))
	if err != nil {
		return nil, fmt.Errorf("listing the asset values: %w", err)
	}

	return inOrder(seek, values), nil
}
