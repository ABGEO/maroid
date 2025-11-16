package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"time"

	"github.com/abgeo/maroid/plugins/telasi/model"
)

// BillingItemsRequest represents the request payload for fetching billing items.
type BillingItemsRequest struct {
	AccountID     string `json:"accountId,omitempty"`
	AccountNumber string `json:"accountNumber"`
	DateFrom      string `json:"dateFrom,omitempty"`
	DateTo        string `json:"dateTo,omitempty"`
	Page          int    `json:"page,omitempty"`
}

// BaseResponse represents the common fields returned in most API responses.
type BaseResponse struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// AuthResponse represents the response returned after authentication.
type AuthResponse struct {
	BaseResponse

	Token                    string `json:"accessToken"`
	TokenType                string `json:"tokenType"`
	ExpiresIn                int    `json:"expiresIn"`
	DoNotValidateBillingLink int    `json:"doNotValidateBillingLink"`
}

// CustomerResponse represents the response for a customer details request.
//
//nolint:tagliatelle
type CustomerResponse struct {
	Status                     string `json:"cut_status"`
	StatusKey                  string `json:"statuskey"`
	Primary                    bool   `json:"primary"`
	AllowedToSubmitApplication bool   `json:"allowedToSubmitApplication"`
	MainAccount                string `json:"mainaccount"`
	Key                        string `json:"custkey"`
	UpdateDate                 string `json:"update_time"`
	Name                       string `json:"custname"`
	Phone                      string `json:"fax"`
	Email                      string `json:"email"`
	AccountNumber              string `json:"accnumb"`
	Region                     string `json:"regionname"`
	Street                     string `json:"streetname"`
	House                      string `json:"house"`
	Building                   string `json:"building"`
	Porch                      string `json:"porch"`
	Flate                      string `json:"flate"`
	WaterBalance               string `json:"water_balance"`
	CleaningBalance            string `json:"trash_bal"`
	TelasiBalance              string `json:"telasi_bal"`
	MeterNumber                string `json:"mtnumb"`
	MeterName                  string `json:"mtname_geo"`
	IsSmart                    string `json:"is_smart"`
	OnTime                     string `json:"on_time"`
	TypeDescription            string `json:"type_descr"`
	Payday                     string `json:"lastday"`
	BillingNumber              string `json:"billnumber"`
	BillingDocumentURL         string `json:"last_bill_link"`
}

// ListResponse represents a paginated list response from the API.
type ListResponse[T any] struct {
	BaseResponse

	List struct {
		Page    int `json:"page"`
		Total   int `json:"total"`
		PerPage int `json:"perPage"`
		Items   []T `json:"items"`
	} `json:"list"`
}

// BillingItem represents a single billing item returned from the API.
//
//nolint:tagliatelle
type BillingItem struct {
	RowNumber     string `json:"rn"`
	ItemDate      string `json:"itemdate"`
	EnterDate     string `json:"enterdate"`
	Amount        string `json:"amount"`
	Consumption   string `json:"kwt"`
	Reading       string `json:"reading"`
	AccountNumber string `json:"accnumb"`
	AccountID     string `json:"accid_geo"`
	MeterNumber   string `json:"mtnumb"`
	Operation     string `json:"billopername_geo"`
}

// Hash computes a unique SHA-256 hash for the BillingItem instance.
func (d *BillingItem) Hash() string {
	data := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%s|%s|%s|%s",
		d.ItemDate,
		d.EnterDate,
		d.Amount,
		d.Consumption,
		d.Reading,
		d.AccountNumber,
		d.AccountID,
		d.MeterNumber,
		d.Operation,
	)
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

// MapToModel converts the external BillingItem representation into a
// model.BillingItem entity suitable for persistence.
func (d *BillingItem) MapToModel() model.BillingItem {
	instance := model.BillingItem{
		Hash:        d.Hash(),
		Operation:   d.Operation,
		Reading:     parseFloatOrZero(d.Reading),
		Consumption: parseFloatOrZero(d.Consumption),
		Amount:      parseFloatOrZero(d.Amount),
	}

	instance.Date, _ = time.Parse(`2006-01-02 15:04:05-07:00`, d.EnterDate+"+04:00")

	return instance
}

func parseFloatOrZero(value string) float64 {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}

	return parsed
}
