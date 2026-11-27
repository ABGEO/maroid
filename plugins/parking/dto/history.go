package dto

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/abgeo/maroid/plugins/parking/errs"
	"github.com/abgeo/maroid/plugins/parking/model"
)

const (
	statusFinished = "FINISHED"

	serviceZone = "Asia/Tbilisi"

	// PARKHIST-DD-006: The sessions carry the local time of Tbilisi with no zone. The
	// transactions carry UTC with an offset.
	layoutSessionTime     = "2006-01-02T15:04:05.999999"
	layoutTransactionTime = "2006-01-02T15:04:05.000-0700"
)

// serviceLocation loads the zone once for every row of every collection.
//
//nolint:gochecknoglobals // a sync.OnceValues of a package holds the loaded zone.
var serviceLocation = sync.OnceValues(func() (*time.Location, error) {
	return time.LoadLocation(serviceZone)
})

// PageMeta is the position of one page in a list of the parking service. Number counts
// from 0, and Total counts the items of the whole list.
type PageMeta struct {
	Total  int `json:"total"`
	Number int `json:"number"`
	Size   int `json:"size"`
}

// Last reports whether no item of the list follows this page.
func (m PageMeta) Last() bool {
	return (m.Number+1)*m.Size >= m.Total
}

// Page is one page of a list of the parking service.
type Page[T any] struct {
	Items []T
	Meta  PageMeta
}

// UnmarshalJSON decodes the envelope of the parking service: result.data and result.page.
func (p *Page[T]) UnmarshalJSON(body []byte) error {
	var envelope struct {
		Result struct {
			Data []T      `json:"data"`
			Page PageMeta `json:"page"`
		} `json:"result"`
	}

	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("decoding the page: %w", err)
	}

	p.Items = envelope.Result.Data
	p.Meta = envelope.Result.Page

	return nil
}

// HistoryPlace is the parking lot of a past session.
//
//nolint:tagliatelle
type HistoryPlace struct {
	UniqueNumber string `json:"uniqueNumber"`
	Address      string `json:"address"`
}

// HistorySession is one row of the past sessions of the parking service.
//
// PARKHIST-INV-002: The type declares no member for the wallet number, so it never
// leaves the decoder.
//
//nolint:tagliatelle
type HistorySession struct {
	ID           int64        `json:"id"`
	Status       string       `json:"status"`
	Type         string       `json:"type"`
	GovNumber    string       `json:"govNumber"`
	ParkingPlace HistoryPlace `json:"parkingPlace"`
	StartDate    string       `json:"startDate"`
	EndDate      *string      `json:"endDate"`
	Amount       float64      `json:"amount"`
}

// Finished reports whether the parking service reports the end of the session.
//
// PARKHIST-DD-003: An active session has no end and a growing amount.
func (s HistorySession) Finished() bool {
	return s.Status == statusFinished && s.EndDate != nil
}

// ToModel answers the session, with its times read in Asia/Tbilisi. A session that is
// not finished, or holds a negative amount, answers errs.ErrUnexpectedAnswer.
//
// PARKHIST-FR-012, PARKHIST-FR-013: The parking service gives the local time of Tbilisi.
func (s HistorySession) ToModel() (model.Session, error) {
	if !s.Finished() {
		return model.Session{}, fmt.Errorf(
			"%w: session %d is not finished",
			errs.ErrUnexpectedAnswer,
			s.ID,
		)
	}

	if s.Amount < 0 {
		return model.Session{}, fmt.Errorf(
			"%w: session %d has a negative amount",
			errs.ErrUnexpectedAnswer,
			s.ID,
		)
	}

	location, err := serviceLocation()
	if err != nil {
		return model.Session{}, fmt.Errorf("loading the time zone %s: %w", serviceZone, err)
	}

	started, err := time.ParseInLocation(layoutSessionTime, s.StartDate, location)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: the start of session %d: %w",
			errs.ErrUnexpectedAnswer,
			s.ID,
			err,
		)
	}

	ended, err := time.ParseInLocation(layoutSessionTime, *s.EndDate, location)
	if err != nil {
		return model.Session{}, fmt.Errorf(
			"%w: the end of session %d: %w",
			errs.ErrUnexpectedAnswer,
			s.ID,
			err,
		)
	}

	return model.Session{
		ExternalID:  s.ID,
		LotNumber:   s.ParkingPlace.UniqueNumber,
		Address:     s.ParkingPlace.Address,
		Plate:       s.GovNumber,
		ParkingType: s.Type,
		StartedAt:   started.UTC(),
		EndedAt:     ended.UTC(),
		Amount:      s.Amount,
	}, nil
}

// PaymentEntry is one row of the ledger of the balance at the parking service.
//
// PARKHIST-INV-002: The type declares no member for the wallet number or the personal
// identifier, so neither leaves the decoder.
//
//nolint:tagliatelle
type PaymentEntry struct {
	ID             int64   `json:"id"`
	OperationType  string  `json:"operationType"`
	Amount         float64 `json:"amount"`
	CurrentBalance float64 `json:"currentBalance"`
	ParkingID      *int64  `json:"parkingId"`
	Description    string  `json:"additionalInformationForMobileEng"`
	OperationDate  string  `json:"operationDate"`
}

// ToModel answers the transaction. A negative amount or a moment in an unknown layout
// answers errs.ErrUnexpectedAnswer.
//
// PARKHIST-FR-014, PARKHIST-FR-016: The transaction keeps the session that it pays.
func (e PaymentEntry) ToModel() (model.Transaction, error) {
	if e.Amount < 0 {
		return model.Transaction{}, fmt.Errorf(
			"%w: transaction %d has a negative amount", errs.ErrUnexpectedAnswer, e.ID,
		)
	}

	occurred, err := time.Parse(layoutTransactionTime, e.OperationDate)
	if err != nil {
		return model.Transaction{}, fmt.Errorf(
			"%w: the moment of transaction %d: %w", errs.ErrUnexpectedAnswer, e.ID, err,
		)
	}

	return model.Transaction{
		ExternalID:        e.ID,
		Kind:              kindOf(e.OperationType),
		ServiceKind:       e.OperationType,
		Amount:            e.Amount,
		BalanceAfter:      e.CurrentBalance,
		OccurredAt:        occurred.UTC(),
		SessionExternalID: e.ParkingID,
		Description:       strings.TrimSpace(e.Description),
	}, nil
}

// kindOf maps the kind that the parking service names.
//
// PARKHIST-FR-015, PARKHIST-DD-007: A kind that Maroid does not know is another kind.
func kindOf(operationType string) model.TransactionKind {
	switch operationType {
	case "FILL_BALANCE":
		return model.TransactionKindTopUp
	case "ZONE_PARKING_PAYMENT":
		return model.TransactionKindParkingPayment
	case "CARD_PAYMENT":
		return model.TransactionKindPermitPurchase
	default:
		return model.TransactionKindOther
	}
}
