package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"github.com/abgeo/maroid/libs/rest/page"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// ContributionRepository defines the data access contract for the contributions of the
// acting user.
type ContributionRepository interface {
	Upsert(ctx context.Context, contribution *model.Contribution) error
	Page(
		ctx context.Context,
		seek page.Seek,
		filter model.ContributionFilter,
	) ([]model.Contribution, error)
	Total(ctx context.Context, filter model.ContributionFilter) (model.ContributionTotal, error)
	MonthSums(ctx context.Context, from *time.Time) ([]model.MonthSum, error)
}

// Contribution reads and writes the contributions of the acting user within one
// transaction of the database.
type Contribution struct {
	tx *sqlx.Tx
}

var _ ContributionRepository = (*Contribution)(nil)

// NewContribution creates a new Contribution over the transaction of the database.
func NewContribution(tx *sqlx.Tx) *Contribution {
	return &Contribution{tx: tx}
}

// Upsert stores the contribution. A held contribution of the same basis and source takes
// the values of the new one when a value changed.
func (r *Contribution) Upsert(ctx context.Context, contribution *model.Contribution) error {
	const query = `
		INSERT INTO contributions (
			basis_id, source, contribution_type, paid_on, closed_on, salary_year,
			salary_month, gross_salary, amount, units, employer_code
		)
		VALUES (
			:basis_id, :source, :contribution_type, CAST(:paid_on AS date),
			CAST(:closed_on AS date), :salary_year, :salary_month, :gross_salary, :amount,
			:units, :employer_code
		)
		ON CONFLICT ON CONSTRAINT contributions_identity_key DO UPDATE
		SET contribution_type = EXCLUDED.contribution_type,
		    paid_on           = EXCLUDED.paid_on,
		    closed_on         = EXCLUDED.closed_on,
		    salary_year       = EXCLUDED.salary_year,
		    salary_month      = EXCLUDED.salary_month,
		    gross_salary      = EXCLUDED.gross_salary,
		    amount            = EXCLUDED.amount,
		    units             = EXCLUDED.units,
		    employer_code     = EXCLUDED.employer_code,
		    collected_at      = now()
		WHERE (contributions.contribution_type, contributions.paid_on, contributions.closed_on,
		       contributions.salary_year, contributions.salary_month,
		       contributions.gross_salary, contributions.amount, contributions.units,
		       contributions.employer_code)
		      IS DISTINCT FROM
		      (EXCLUDED.contribution_type, EXCLUDED.paid_on, EXCLUDED.closed_on,
		       EXCLUDED.salary_year, EXCLUDED.salary_month, EXCLUDED.gross_salary,
		       EXCLUDED.amount, EXCLUDED.units, EXCLUDED.employer_code);`

	if _, err := r.tx.NamedExecContext(ctx, query, contribution); err != nil {
		return fmt.Errorf("upserting the contribution: %w", err)
	}

	return nil
}

// matchesFilter keeps the contributions of the filter. The alias is t.
const matchesFilter = `
	(CAST(:employer AS text) IS NULL OR t.employer_code = CAST(:employer AS text))
	AND (CAST(:from AS date) IS NULL OR t.paid_on >= CAST(:from AS date))
	AND (CAST(:to AS date) IS NULL OR t.paid_on <= CAST(:to AS date))`

// contributionRow is a contribution with the name of its employer.
type contributionRow struct {
	model.Contribution

	EmployerName *string `db:"employer_name"`
}

// Page answers one keyset read of the contributions of the filter, the latest date
// first, with the identifier as the tie.
func (r *Contribution) Page(
	ctx context.Context,
	seek page.Seek,
	filter model.ContributionFilter,
) ([]model.Contribution, error) {
	condition, sorting := keyset(seek, "contributions", "paid_on")
	query := `
		SELECT t.id, t.basis_id, t.source, t.contribution_type, t.paid_on, t.closed_on,
		       t.salary_year, t.salary_month, t.gross_salary, t.amount, t.units,
		       t.employer_code, e.name AS employer_name
		FROM contributions t
		LEFT JOIN employers e ON e.code = t.employer_code
		WHERE ` + matchesFilter + `
		  AND ` + condition + `
		ORDER BY ` + sorting + `
		LIMIT :limit;`

	arguments := filterArguments(filter)
	arguments["boundary"] = boundaryOf(seek)
	arguments["limit"] = seek.Limit

	rows, err := selectNamed[contributionRow](ctx, r.tx, query, arguments)
	if err != nil {
		return nil, fmt.Errorf("listing the contributions: %w", err)
	}

	contributions := make([]model.Contribution, 0, len(rows))

	for _, row := range rows {
		contribution := row.Contribution
		if row.EmployerCode != nil && row.EmployerName != nil {
			contribution.Employer = &model.Employer{
				Code: *row.EmployerCode,
				Name: *row.EmployerName,
			}
		}

		contributions = append(contributions, contribution)
	}

	return inOrder(seek, contributions), nil
}

// Total answers the sum of the amounts and the count of every contribution of the filter.
func (r *Contribution) Total(
	ctx context.Context,
	filter model.ContributionFilter,
) (model.ContributionTotal, error) {
	query := `
		SELECT coalesce(sum(t.amount), 0) AS amount, count(*) AS count
		FROM contributions t
		WHERE ` + matchesFilter + `;`

	totals, err := selectNamed[model.ContributionTotal](ctx, r.tx, query, filterArguments(filter))
	if err != nil {
		return model.ContributionTotal{}, fmt.Errorf("summing the contributions: %w", err)
	}

	return totals[0], nil
}

// MonthSums answers the sum of the contributions of each month that holds one, from the
// month of from, the earliest month first. A nil from reads from the first contribution.
func (r *Contribution) MonthSums(ctx context.Context, from *time.Time) ([]model.MonthSum, error) {
	const query = `
		SELECT CAST(date_trunc('month', t.paid_on) AS date) AS month,
		       coalesce(sum(t.amount), 0) AS amount
		FROM contributions t
		WHERE CAST(:from AS date) IS NULL OR t.paid_on >= CAST(:from AS date)
		GROUP BY 1
		ORDER BY 1;`

	sums, err := selectNamed[model.MonthSum](
		ctx,
		r.tx,
		query,
		map[string]any{paramFrom: dateOf(from)},
	)
	if err != nil {
		return nil, fmt.Errorf("summing the contributions of each month: %w", err)
	}

	return sums, nil
}

func filterArguments(filter model.ContributionFilter) map[string]any {
	var employer any
	if filter.EmployerCode != nil {
		employer = *filter.EmployerCode
	}

	return map[string]any{
		"employer": employer,
		paramFrom:  dateOf(filter.From),
		"to":       dateOf(filter.To),
	}
}
