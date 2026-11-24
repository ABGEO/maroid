package dto

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/abgeo/maroid/plugins/magti/errs"
	"github.com/abgeo/maroid/plugins/magti/model"
)

// The members of the portal that the readers of this file decode. A member that is not
// named here never reaches a value of Go, the personal data of the holder among them.
const (
	groupBalance     = "BALANCE"
	groupInternet    = "INTERNET"
	groupMinutes     = "MINUTES"
	groupSMS         = "SMS"
	groupISPBalance  = "ISP_BALANCE"
	groupISPInternet = "ISP_INTERNET"

	webNameAttribute = "WEB_NAME"

	// portalMomentLayout is the form of a moment in an answer of the portal.
	portalMomentLayout = "02/01/2006 15:04:05"

	// earliestUsageYear bounds a usage row. The portal answers a row of the epoch for a
	// day that it does not know.
	earliestUsageYear = 2000

	secondsOfMinute = 60
	secondsOfHour   = 60 * secondsOfMinute
)

// LogInAnswer is the answer of the sign in.
type LogInAnswer struct {
	Message string `json:"message"`
}

// The messages of LogInAnswer that the script of the portal reads.
const (
	LogInSuccess = "success"
	//nolint:gosec // G101: the name of an answer, not a credential.
	LogInInvalidCredentials = "invalid-credentials"
	LogInCodeRequired       = "two-factor-code-required"
	LogInLocked             = "locked"
)

// SubscriberLink is one subscriber that the menu of a page links to.
type SubscriberLink struct {
	Number string
	Label  string
}

// SubscriberPage is what a page of the portal tells about the subscribers: the kind of
// the selected subscriber, and the links to the others. The menu holds no link to the
// selected subscriber.
type SubscriberPage struct {
	Kind  model.SubscriberKind
	Links []SubscriberLink
}

// Number is a number of the portal that arrives as a JSON number or as a string.
type Number float64

// UnmarshalJSON reads a number, a string that holds one, or null as zero.
func (n *Number) UnmarshalJSON(raw []byte) error {
	text := strings.Trim(string(bytes.TrimSpace(raw)), `"`)
	if text == "" || text == "null" {
		*n = 0

		return nil
	}

	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return fmt.Errorf("%w: reading the number %q: %w", errs.ErrRequestFailed, text, err)
	}

	*n = Number(value)

	return nil
}

// String answers the number as the portal takes it in a query, with no trailing zero.
func (n Number) String() string {
	return strconv.FormatFloat(float64(n), 'f', -1, 64)
}

// Flag is a truth value of the portal that arrives as a boolean, a number, or a string.
type Flag bool

// UnmarshalJSON reads true, a number other than zero, or a string other than "", "0",
// and "false" as true.
func (f *Flag) UnmarshalJSON(raw []byte) error {
	text := strings.Trim(string(bytes.TrimSpace(raw)), `"`)
	*f = Flag(text != "" && text != "0" && text != "false" && text != "null")

	return nil
}

// AccountBalance is one balance of a subscriber: the money, or one allowance.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type AccountBalance struct {
	Balance                 string         `json:"balance"`
	GroupKey                string         `json:"groupKey"`
	AccountGroupConfigItems []accountGroup `json:"accountGroupConfigItems"`
}

//nolint:tagliatelle // the portal names its members in camelCase.
type accountGroup struct {
	AccountGroupKey string `json:"accountGroupKey"`
}

// Key answers the group that the balance belongs to.
func (b AccountBalance) Key() string {
	if len(b.AccountGroupConfigItems) > 0 && b.AccountGroupConfigItems[0].AccountGroupKey != "" {
		return b.AccountGroupConfigItems[0].AccountGroupKey
	}

	return b.GroupKey
}

// ActiveCocktail is the active plan of a mobile subscriber.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type ActiveCocktail struct {
	CocktailID Number `json:"cocktailId"`
	UntilDate  string `json:"untilDate"`
	ValidityID Number `json:"validityId"`
}

// Accounts is the answer of accounts.php for a mobile subscriber.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type Accounts struct {
	Status         string           `json:"status"`
	Balances       []AccountBalance `json:"balances"`
	ActiveCocktail *ActiveCocktail  `json:"activeCocktail"`
}

// HasPlan reports whether the subscriber holds an active plan.
func (a Accounts) HasPlan() bool {
	return a.ActiveCocktail != nil && a.ActiveCocktail.CocktailID != 0
}

// CocktailPackage is the answer of cocktail-package.php: the name and the price of a plan.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type CocktailPackage struct {
	Name               string              `json:"name"`
	Price              Number              `json:"price"`
	CocktailAttributes []cocktailAttribute `json:"cocktailAttributes"`
}

//nolint:tagliatelle // the portal names its members in camelCase.
type cocktailAttribute struct {
	NameKeyDescriptions  map[string]string `json:"nameKeyDescriptions"`
	ValueKeyDescriptions map[string]string `json:"valueKeyDescriptions"`
}

// DisplayName answers the name that the page of the portal shows: the WEB_NAME attribute
// when the plan holds one, in English when the portal gives it.
func (p CocktailPackage) DisplayName() string {
	for _, attribute := range p.CocktailAttributes {
		if !namesWebName(attribute.NameKeyDescriptions) {
			continue
		}

		for _, language := range []string{"en", "ka"} {
			if value := attribute.ValueKeyDescriptions[language]; value != "" {
				return value
			}
		}
	}

	return p.Name
}

func namesWebName(names map[string]string) bool {
	for _, name := range names {
		if name == webNameAttribute {
			return true
		}
	}

	return false
}

// IspAccounts is the answer of accounts-isp.php for a home line.
type IspAccounts struct {
	PackageSum Number           `json:"package_sum"`
	Balances   []AccountBalance `json:"balances"`
}

// AccountData is the answer of account-data.php: the status and the days to the renewal.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type AccountData struct {
	StatusName    string `json:"statusName"`
	DaysRemaining Number `json:"daysRemaining"`
}

// Statistics is the answer of step 3 of statistics.php.
type Statistics struct {
	Key     string          `json:"key"`
	History json.RawMessage `json:"history"`
}

// UsageRow is one day of the data, the minutes, or the SMS.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type UsageRow struct {
	Day         int64  `json:"day"`
	ChartData   Number `json:"chartData"`
	ChartDataIn Number `json:"chartDataIn"`
	Sum         Flag   `json:"sum"`
}

// MoneyRow is one top up or one charge.
//
//nolint:tagliatelle // the portal names its members in camelCase.
type MoneyRow struct {
	TransactionDate string `json:"transactionDate"`
	EntryAmount     Number `json:"entryAmount"`
	ActionType      string `json:"actionType"`
	Sum             Flag   `json:"sum"`
}

// MobileState reads the state of a mobile subscriber. plan is nil when the subscriber
// holds no active plan. The days left of the plan stay zero, because they depend on the
// moment of the read.
func MobileState(number string, accounts Accounts, plan *CocktailPackage) (model.State, error) {
	state := model.State{
		SubscriberNumber: number,
		Kind:             model.SubscriberKindMobile,
		Status:           accounts.Status,
		Allowances:       new(model.Allowances),
	}

	if !holdsBalance(accounts.Balances, groupBalance) {
		return model.State{}, fmt.Errorf("%w: the answer holds no balance", errs.ErrRequestFailed)
	}

	for _, balance := range accounts.Balances {
		if err := readMobileBalance(&state, balance); err != nil {
			return model.State{}, err
		}
	}

	if plan == nil || !accounts.HasPlan() {
		return state, nil
	}

	renewsAt, err := time.ParseInLocation(
		portalMomentLayout,
		accounts.ActiveCocktail.UntilDate,
		model.Tbilisi(),
	)
	if err != nil {
		return model.State{}, fmt.Errorf("%w: reading the renewal: %w", errs.ErrRequestFailed, err)
	}

	state.Plan = &model.Plan{
		Name:     plan.DisplayName(),
		Price:    model.TetriOf(float64(plan.Price)),
		RenewsAt: &renewsAt,
	}

	return state, nil
}

func readMobileBalance(state *model.State, balance AccountBalance) error {
	var err error

	switch balance.Key() {
	case groupBalance:
		var lari float64

		lari, err = parseDecimal(balance.Balance)
		state.Balance = model.TetriOf(lari)
	case groupInternet:
		state.Allowances.DataMB, err = parseDecimal(balance.Balance)
	case groupMinutes:
		state.Allowances.CallSeconds, err = parseDuration(balance.Balance)
	case groupSMS:
		var count float64

		count, err = parseDecimal(balance.Balance)
		state.Allowances.SMS = int(math.Round(count))
	}

	if err != nil {
		return fmt.Errorf("reading the balance %s: %w", balance.Key(), err)
	}

	return nil
}

func holdsBalance(balances []AccountBalance, key string) bool {
	for _, balance := range balances {
		if balance.Key() == key {
			return true
		}
	}

	return false
}

// HomeLineState reads the state of a home line.
func HomeLineState(number string, accounts IspAccounts, data AccountData) (model.State, error) {
	state := model.State{
		SubscriberNumber: number,
		Kind:             model.SubscriberKindHomeLine,
		Status:           data.StatusName,
		Plan: &model.Plan{
			Price:    model.TetriOf(float64(accounts.PackageSum)),
			DaysLeft: max(int(data.DaysRemaining), 0),
		},
	}

	if !holdsBalance(accounts.Balances, groupISPBalance) {
		return model.State{}, fmt.Errorf("%w: the answer holds no balance", errs.ErrRequestFailed)
	}

	for _, balance := range accounts.Balances {
		switch balance.Key() {
		case groupISPBalance:
			value, err := parseDecimal(balance.Balance)
			if err != nil {
				return model.State{}, fmt.Errorf("reading the balance of a home line: %w", err)
			}

			state.Balance = model.TetriOf(value)
		case groupISPInternet:
			state.Speed = strings.TrimSpace(balance.Balance)
		}
	}

	return state, nil
}

// UsageSeries names the series of statistics.php that a usage row belongs to.
type UsageSeries string

// The usage series, by the key that statistics.php takes.
const (
	SeriesData    UsageSeries = "MB"
	SeriesMinutes UsageSeries = "MIN"
	SeriesSMS     UsageSeries = "SMS"
)

// MergeUsage folds the rows of one usage series into the traffic of each day. The row
// that sums the period, and a row of a day that the portal does not know, stay out.
func MergeUsage(days map[model.Day]*model.TrafficDay, series UsageSeries, rows []UsageRow) {
	for _, row := range rows {
		if row.Sum || row.Day <= 0 {
			continue
		}

		moment := time.UnixMilli(row.Day)
		if moment.Year() < earliestUsageYear {
			continue
		}

		day := model.DayOf(moment)

		traffic, found := days[day]
		if !found {
			traffic = &model.TrafficDay{Day: day.Date()}
			days[day] = traffic
		}

		out, in := float64(row.ChartData), float64(row.ChartDataIn)

		switch series {
		case SeriesData:
			traffic.DataMB = new(out)
		case SeriesMinutes:
			traffic.CallOutSeconds = new(int(math.Round(out * secondsOfMinute)))
			traffic.CallInSeconds = new(int(math.Round(in * secondsOfMinute)))
		case SeriesSMS:
			traffic.SMSSent = new(int(math.Round(out)))
			traffic.SMSReceived = new(int(math.Round(in)))
		}
	}
}

// MoneyEntries reads the money rows of one direction. The row that sums the period stays out.
func MoneyEntries(rows []MoneyRow, direction model.Direction) ([]model.MoneyEntry, error) {
	entries := make([]model.MoneyEntry, 0, len(rows))

	for _, row := range rows {
		if row.Sum || row.TransactionDate == "" {
			continue
		}

		occurredAt, err := time.ParseInLocation(
			portalMomentLayout,
			row.TransactionDate,
			model.Tbilisi(),
		)
		if err != nil {
			return nil, fmt.Errorf(
				"%w: reading the moment of a money entry: %w",
				errs.ErrRequestFailed,
				err,
			)
		}

		entries = append(entries, model.MoneyEntry{
			OccurredAt: occurredAt.UTC(),
			Direction:  direction,
			Kind:       strings.TrimSpace(row.ActionType),
			Amount:     model.TetriOf(math.Abs(float64(row.EntryAmount))),
		})
	}

	return entries, nil
}

func parseDecimal(raw string) (float64, error) {
	value, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil {
		return 0, fmt.Errorf("%w: reading %q: %w", errs.ErrRequestFailed, raw, err)
	}

	return value, nil
}

// parseDuration reads the minutes left, which the portal writes as mm:ss or hh:mm:ss.
func parseDuration(raw string) (int, error) {
	parts := strings.Split(strings.TrimSpace(raw), ":")
	units := []int{secondsOfMinute, 1}

	hourUnits := []int{secondsOfHour, secondsOfMinute, 1}
	if len(parts) < len(units) || len(parts) > len(hourUnits) {
		return 0, fmt.Errorf("%w: reading the minutes %q", errs.ErrRequestFailed, raw)
	}

	if len(parts) == len(hourUnits) {
		units = hourUnits
	}

	seconds := 0

	for index, part := range parts {
		value, err := strconv.Atoi(part)
		if err != nil {
			return 0, fmt.Errorf("%w: reading the minutes %q: %w", errs.ErrRequestFailed, raw, err)
		}

		seconds += value * units[index]
	}

	return seconds, nil
}
