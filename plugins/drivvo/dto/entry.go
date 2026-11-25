package dto

import (
	"time"

	"github.com/abgeo/maroid/plugins/drivvo/model"
)

// LogEntryItemResponse is one row of the history or of a list. A member that the kind does
// not hold is absent.
type LogEntryItemResponse struct {
	ID          string         `json:"id"`
	Kind        string         `json:"kind"`
	OccurredAt  time.Time      `json:"occurred_at"`
	Odometer    float64        `json:"odometer"`
	Amount      *MoneyResponse `json:"amount,omitempty"`
	Title       string         `json:"title,omitempty"`
	Labels      []string       `json:"labels"`
	Volume      *float64       `json:"volume,omitempty"`
	Consumption *float64       `json:"consumption,omitempty"`
	IncomeType  string         `json:"income_type,omitempty"`
}

// NewLogEntryItemResponseList converts one page of entries. consumption holds the value of
// each refuel that ends a part with a value.
func NewLogEntryItemResponseList(
	entries []model.EntryRow,
	consumption map[string]float64,
	currency string,
) []LogEntryItemResponse {
	answer := make([]LogEntryItemResponse, 0, len(entries))

	for _, entry := range entries {
		item := LogEntryItemResponse{
			ID:         entry.ID,
			Kind:       string(entry.Kind),
			OccurredAt: entry.OccurredAt.UTC(),
			Odometer:   entry.Odometer,
			Amount:     moneyOf(entry.Amount, currency),
			Title:      text(entry.Title),
			Labels:     labels(entry.Lines),
		}

		switch entry.Kind {
		case model.KindRefuel:
			addRefuel(&item, entry, consumption)
		case model.KindIncome:
			item.IncomeType = text(entry.IncomeType)
		case model.KindMaintenance, model.KindExpense, model.KindOdometer:
		}

		answer = append(answer, item)
	}

	return answer
}

func addRefuel(item *LogEntryItemResponse, entry model.EntryRow, consumption map[string]float64) {
	var volume float64

	for _, line := range entry.Lines {
		if line.Fuel() && line.Volume != nil {
			volume += *line.Volume
		}
	}

	item.Volume = &volume

	if value, found := consumption[entry.ID]; found {
		item.Consumption = &value
	}
}

// LogEntryTotalsResponse sums the entries of one kind in a period.
type LogEntryTotalsResponse struct {
	Kind   string        `json:"kind"`
	Count  int32         `json:"count"`
	Amount MoneyResponse `json:"amount"`
	Volume *float64      `json:"volume,omitempty"`
}

// NewLogEntryTotalsResponse converts the totals. A refuel answers the volume too.
func NewLogEntryTotalsResponse(
	kind model.Kind,
	totals model.EntryTotals,
	currency string,
) LogEntryTotalsResponse {
	answer := LogEntryTotalsResponse{
		Kind:   string(kind),
		Count:  totals.Count,
		Amount: NewMoneyResponse(totals.Amount, currency),
	}

	if kind == model.KindRefuel {
		answer.Volume = &totals.Volume
	}

	return answer
}

// LogEntryResponse is one log entry with each value that the copy holds of it. A member
// that the kind does not hold, or that the service did not send, is absent.
type LogEntryResponse struct {
	ID              string               `json:"id"`
	Kind            string               `json:"kind"`
	OccurredAt      time.Time            `json:"occurred_at"`
	Odometer        float64              `json:"odometer"`
	Note            string               `json:"note,omitempty"`
	Amount          *MoneyResponse       `json:"amount,omitempty"`
	Title           string               `json:"title,omitempty"`
	Station         string               `json:"station,omitempty"`
	Place           string               `json:"place,omitempty"`
	PaymentMethod   string               `json:"payment_method,omitempty"`
	Reason          string               `json:"reason,omitempty"`
	IncomeType      string               `json:"income_type,omitempty"`
	MissedRefuel    *bool                `json:"missed_refuel,omitempty"`
	BatteryStartPct *float64             `json:"battery_start_pct,omitempty"`
	BatteryEndPct   *float64             `json:"battery_end_pct,omitempty"`
	ChargeMinutes   *int32               `json:"charge_minutes,omitempty"`
	Lines           []LineResponse       `json:"lines,omitempty"`
	Instalments     []InstalmentResponse `json:"instalments,omitempty"`
}

// LineResponse is one line of a log entry. The members after amount hold a fuel line alone.
type LineResponse struct {
	Position  int            `json:"position"`
	Item      string         `json:"item,omitempty"`
	Amount    MoneyResponse  `json:"amount"`
	UnitPrice *MoneyResponse `json:"unit_price,omitempty"`
	Volume    *float64       `json:"volume,omitempty"`
	FullTank  *bool          `json:"full_tank,omitempty"`
	Free      *bool          `json:"free,omitempty"`
}

// InstalmentResponse is one instalment of a log entry.
type InstalmentResponse struct {
	Number int16         `json:"number"`
	DueOn  string        `json:"due_on"`
	Amount MoneyResponse `json:"amount"`
}

// NewLogEntryResponse converts one entry with its lines and instalments.
func NewLogEntryResponse(entry model.EntryRow, currency string) LogEntryResponse {
	answer := LogEntryResponse{
		ID:              entry.ID,
		Kind:            string(entry.Kind),
		OccurredAt:      entry.OccurredAt.UTC(),
		Odometer:        entry.Odometer,
		Note:            text(entry.Note),
		Amount:          moneyOf(entry.Amount, currency),
		Title:           text(entry.Title),
		Station:         text(entry.Station),
		Place:           text(entry.Place),
		PaymentMethod:   text(entry.PaymentMethod),
		Reason:          text(entry.Reason),
		IncomeType:      text(entry.IncomeType),
		BatteryStartPct: entry.BatteryStartPct,
		BatteryEndPct:   entry.BatteryEndPct,
		ChargeMinutes:   entry.ChargeMinutes,
	}

	if entry.Kind == model.KindRefuel {
		answer.MissedRefuel = &entry.MissedRefuel
	}

	for _, line := range entry.Lines {
		answer.Lines = append(answer.Lines, LineResponse{
			Position:  line.Position,
			Item:      text(line.Item),
			Amount:    NewMoneyResponse(line.Amount, currency),
			UnitPrice: moneyOf(line.UnitPrice, currency),
			Volume:    line.Volume,
			FullTank:  line.FullTank,
			Free:      line.Free,
		})
	}

	for _, instalment := range entry.Instalments {
		answer.Instalments = append(answer.Instalments, InstalmentResponse{
			Number: instalment.Number,
			DueOn:  model.FormatDay(instalment.DueOn),
			Amount: NewMoneyResponse(instalment.Amount, currency),
		})
	}

	return answer
}

func labels(lines []model.LineRow) []string {
	names := make([]string, 0, len(lines))

	for _, line := range lines {
		if line.Item != nil {
			names = append(names, *line.Item)
		}
	}

	return names
}

func text(value *string) string {
	if value == nil {
		return ""
	}

	return *value
}
