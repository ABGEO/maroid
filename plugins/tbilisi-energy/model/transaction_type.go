package model

// TransactionType represents a single transaction type record.
type TransactionType struct {
	ID   int    `db:"id"`
	Name string `db:"name"`
}
