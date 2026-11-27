package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/abgeo/maroid/plugins/pensions/errs"
	"github.com/abgeo/maroid/plugins/pensions/model"
)

// dateLength is the length of the calendar date that starts each date of the pension
// service, YYYY-MM-DD.
const dateLength = len(time.DateOnly)

// SignInRequest is the body of the sign in.
//
//nolint:tagliatelle // the pension service names the members.
type SignInRequest struct {
	Username     string `json:"username"`
	PasswordOne  string `json:"passwordOne"`
	PasswordTwo  string `json:"passwordTwo"`
	LanguageCode string `json:"languageCode"`
}

// SignInResponse is the answer of the sign in. The refresh token stays undecoded,
// because the plugin keeps none.
//
//nolint:tagliatelle // the pension service names the members.
type SignInResponse struct {
	AccessToken string `json:"accessToken"`
}

// ParticipantResponse is the holder of the account and the summary of the account. It
// declares no member for the personal number, so the number never reaches a value of Go.
//
//nolint:tagliatelle // the pension service names the members.
type ParticipantResponse struct {
	FirstName           string  `json:"firstName"`
	LastName            string  `json:"lastName"`
	CummSavings         float64 `json:"cummSavings"`
	Gain                float64 `json:"gain"`
	TotalWithheldAmount float64 `json:"totalWithheldAmount"`
	EmpContr            float64 `json:"empContr"`
	OrgContr            float64 `json:"orgContr"`
	GovtContr           float64 `json:"govtContr"`
	TotalContr          float64 `json:"totalContr"`
}

// MapToModel answers the participant. The full name joins the first and the last name
// with one space.
func (d *ParticipantResponse) MapToModel() model.Participant {
	return model.Participant{
		FullName: strings.TrimSpace(
			strings.TrimSpace(d.FirstName) + " " + strings.TrimSpace(d.LastName),
		),
		Summary: model.AccountSummary{
			AssetValue:            d.CummSavings,
			Gain:                  d.Gain,
			WithheldAmount:        d.TotalWithheldAmount,
			PersonContributions:   d.EmpContr,
			EmployerContributions: d.OrgContr,
			StateContributions:    d.GovtContr,
			TotalContributions:    d.TotalContr,
		},
	}
}

// AssetResponse is the share of the account in one portfolio.
//
//nolint:tagliatelle // the pension service names the members.
type AssetResponse struct {
	PortfolioID        int64   `json:"portfolioId"`
	PortfolioNameGe    string  `json:"portfolioNameGe"`
	PortfolioNameEn    string  `json:"portfolioNameEn"`
	AssetManagerNameGe string  `json:"assetManagerNameGe"`
	AssetManagerNameEn string  `json:"assetManagerNameEn"`
	Ticker             string  `json:"ticker"`
	RiskTierNameGe     string  `json:"riskTierNameGe"`
	RiskTierNameEn     string  `json:"riskTierNameEn"`
	TargetShare        float64 `json:"targetShare"`
	ShareReal          float64 `json:"shareReal"`
	Amount             float64 `json:"amount"`
	Units              float64 `json:"units"`
	UnitPrice          float64 `json:"unitPrice"`
}

// MapToModel answers the portfolio share.
func (d *AssetResponse) MapToModel() model.PortfolioShare {
	return model.PortfolioShare{
		PortfolioID:        d.PortfolioID,
		PortfolioNameKa:    d.PortfolioNameGe,
		PortfolioNameEn:    d.PortfolioNameEn,
		AssetManagerNameKa: d.AssetManagerNameGe,
		AssetManagerNameEn: d.AssetManagerNameEn,
		Ticker:             d.Ticker,
		RiskTierNameKa:     d.RiskTierNameGe,
		RiskTierNameEn:     d.RiskTierNameEn,
		TargetShare:        d.TargetShare,
		CurrentShare:       d.ShareReal,
		Amount:             d.Amount,
		Units:              d.Units,
		UnitPrice:          d.UnitPrice,
	}
}

// ChartRequest is the body of the chart of the asset value: two dates, YYYY-MM-DD.
//
//nolint:tagliatelle // the pension service names the members.
type ChartRequest struct {
	DateFrom string `json:"dateFrom"`
	DateTo   string `json:"dateTo"`
}

// ChartResponse is the answer of the chart of the asset value.
type ChartResponse struct {
	Errors json.RawMessage `json:"errors"`
	Result []ChartPoint    `json:"result"`
}

// Failed reports an answer that names errors.
func (d *ChartResponse) Failed() bool {
	trimmed := strings.TrimSpace(string(d.Errors))

	return trimmed != "" && trimmed != "null"
}

// ChartPoint is the asset value of the account on one date.
//
//nolint:tagliatelle // the pension service names the members.
type ChartPoint struct {
	Date        string  `json:"date"`
	CummSavings float64 `json:"cummSavings"`
	CummUnits   float64 `json:"cummUnits"`
	UnitPrice   float64 `json:"unitPrice"`
}

// MapToModel answers the asset value.
func (d *ChartPoint) MapToModel() (model.AssetValue, error) {
	valuedOn, err := ParseDate(d.Date)
	if err != nil {
		return model.AssetValue{}, fmt.Errorf("reading the date of an asset value: %w", err)
	}

	return model.AssetValue{
		ValuedOn:   valuedOn,
		AssetValue: d.CummSavings,
		Units:      d.CummUnits,
		UnitPrice:  d.UnitPrice,
	}, nil
}

// NavRequest is the body of the unit prices of the portfolios.
//
//nolint:tagliatelle // the pension service names the members.
type NavRequest struct {
	PortfolioIDs []int64 `json:"portfolioIds"`
}

// NavPortfolio is the history of the unit price of one portfolio.
//
//nolint:tagliatelle // the pension service names the members.
type NavPortfolio struct {
	PortfolioID   int64      `json:"portfolioId"`
	NetAssetValue []NavPoint `json:"netAssetValue"`
}

// NavPoint is the unit price of one portfolio on one day.
//
//nolint:tagliatelle // the pension service names the members.
type NavPoint struct {
	PortfolioID      int64    `json:"portfolioId"`
	NavDate          string   `json:"navDate"`
	ClosingNav       *float64 `json:"closingNav"`
	ClosingUnitPrice float64  `json:"closingUnitPrice"`
	ClosingUnits     *float64 `json:"closingUnits"`
}

// MapToModel answers the unit price.
func (d *NavPoint) MapToModel() (model.PortfolioUnitPrice, error) {
	pricedOn, err := ParseDate(d.NavDate)
	if err != nil {
		return model.PortfolioUnitPrice{}, fmt.Errorf("reading the date of a unit price: %w", err)
	}

	return model.PortfolioUnitPrice{
		PortfolioID:   d.PortfolioID,
		PricedOn:      pricedOn,
		UnitPrice:     d.ClosingUnitPrice,
		NetAssetValue: d.ClosingNav,
		Units:         d.ClosingUnits,
	}, nil
}

// ContributionsPage is one page of the contributions.
type ContributionsPage struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageCount int            `json:"pageCount"` //nolint:tagliatelle // the service names it.
		Result    []Contribution `json:"result"`
	} `json:"data"`
}

// Contribution is one contribution as the pension service answers it.
//
//nolint:tagliatelle // the pension service names the members.
type Contribution struct {
	BasisID          uuid.UUID `json:"basisId"`
	Date             string    `json:"date"`
	ClosingDate      *string   `json:"closingDate"`
	Year             *int      `json:"year"`
	Month            *int      `json:"month"`
	GrossSalary      float64   `json:"grossSalary"`
	ContrType        string    `json:"contrType"`
	Source           string    `json:"source"`
	Amount           *float64  `json:"amount"`
	Units            *float64  `json:"units"`
	OrganizationCode *string   `json:"organizationCode"`
	OrganizationName *string   `json:"organizationName"`
}

// MapToModel answers the contribution. A contribution names its employer only with both
// a code and a name.
func (d *Contribution) MapToModel() (model.Contribution, error) {
	paidOn, err := ParseDate(d.Date)
	if err != nil {
		return model.Contribution{}, fmt.Errorf("reading the date of a contribution: %w", err)
	}

	contribution := model.Contribution{
		BasisID:          d.BasisID,
		Source:           d.Source,
		ContributionType: d.ContrType,
		PaidOn:           paidOn,
		SalaryYear:       d.Year,
		SalaryMonth:      d.Month,
		GrossSalary:      d.GrossSalary,
		Amount:           d.Amount,
		Units:            d.Units,
	}

	if d.ClosingDate != nil && *d.ClosingDate != "" {
		closedOn, err := ParseDate(*d.ClosingDate)
		if err != nil {
			return model.Contribution{}, fmt.Errorf(
				"reading the closing date of a contribution: %w",
				err,
			)
		}

		contribution.ClosedOn = &closedOn
	}

	code, name := trimmed(d.OrganizationCode), trimmed(d.OrganizationName)
	if code != "" && name != "" {
		contribution.EmployerCode = &code
		contribution.Employer = &model.Employer{Code: code, Name: name}
	}

	return contribution, nil
}

// ParseDate reads the calendar date of the first 10 characters of a date of the pension
// service. The service marks a time of Tbilisi with Z on some members and with no zone on
// others, so a conversion of the zone would move the date across midnight.
func ParseDate(raw string) (time.Time, error) {
	if len(raw) < dateLength {
		return time.Time{}, fmt.Errorf(
			"%w: the date %q is shorter than %s",
			errs.ErrRequestFailed,
			raw,
			time.DateOnly,
		)
	}

	day, err := time.Parse(time.DateOnly, raw[:dateLength])
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing the date %q: %w", raw, err)
	}

	return day, nil
}

func trimmed(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
