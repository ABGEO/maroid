package model

// Snapshot is everything that one collection of one month writes.
type Snapshot struct {
	Contributions []Contribution
	AssetValues   []AssetValue
	UnitPrices    []PortfolioUnitPrice
	Summary       AccountSummary
	Shares        []PortfolioShare
}

// Collection is what one collection of one month read.
type Collection struct {
	Contributions int
	AssetValues   int
	UnitPrices    int
	Shares        int
}
