package dto

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// AuthResponse represents the response from the authentication API.
type AuthResponse struct {
	IsValid          bool      `json:"isValid,omitempty"`
	IsEmployee       bool      `json:"isEmployee,omitempty"`
	IsFirstLogin     bool      `json:"isFirstLogined,omitempty"`
	HasManyCompanies bool      `json:"hasManyCompanies,omitempty"`
	Message          string    `json:"message,omitempty"`
	AccessToken      string    `json:"accessToken,omitempty"`
	RefreshToken     string    `json:"refreshToken,omitempty"`
	UserID           uuid.UUID `json:"userId,omitempty"`
}

// ParticipantInfoResponse represents the participant information response from the API.
type ParticipantInfoResponse struct {
	UserID                   uuid.UUID `binding:"required" json:"applicationUserId"`
	PersonalID               string    `binding:"required" json:"personalId"`
	FirstName                string    `binding:"required" json:"firstName"`
	LastName                 string    `binding:"required" json:"lastName"`
	TotalUnits               float64   `binding:"required" json:"totalUnits"`
	EmployeeContribution     float64   `binding:"required" json:"empContr"`
	OrganisationContribution float64   `binding:"required" json:"orgContr"`
	GovernmentContribution   float64   `binding:"required" json:"govtContr"`
	TotalContributions       float64   `binding:"required" json:"totalContr"`
	TotalSavings             float64   `binding:"required" json:"cummSavings"`
}

// PaginatedResponse represents a generic paginated response structure.
type PaginatedResponse[T any] struct {
	Status  string `json:"status"`
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		PageNumber int `json:"pageNumber"`
		PageSize   int `json:"pageSize"`
		RowCount   int `json:"rowCount"`
		PageCount  int `json:"pageCount"`
		Result     []T `json:"result"`
	} `json:"data"`
}

// ContributionsRequest represents the request parameters for fetching contributions.
type ContributionsRequest struct {
	Page         int        `json:"pageNumber"`
	PageSize     int        `json:"pageSize"`
	Organization string     `json:"orgCode,omitempty"`
	StartDate    *time.Time `json:"startDate,omitempty"`
	EndDate      *time.Time `json:"endDate,omitempty"`
}

// ToQueryParams converts the ContributionsRequest into a map of query parameters.
func (d *ContributionsRequest) ToQueryParams() map[string]string {
	params := map[string]string{
		"pageNumber": strconv.Itoa(d.Page),
		"pageSize":   strconv.Itoa(d.PageSize),
	}

	if d.StartDate != nil {
		params["startDate"] = d.StartDate.Format("2006-01-02")
	}

	if d.EndDate != nil {
		params["endDate"] = d.EndDate.Format("2006-01-02")
	}

	if d.Organization != "" {
		params["orgCode"] = d.Organization
	}

	return params
}

// Contribution represents a pension contribution record from the external API.
type Contribution struct {
	BasisID          uuid.UUID `json:"basisId"`
	Date             string    `json:"date"`
	ClosingDate      *string   `json:"closingDate,omitempty"`
	OrganizationCode *string   `json:"organizationCode,omitempty"`
	OrganizationName *string   `json:"organizationName,omitempty"`
	Year             *int      `json:"year,omitempty"`
	Month            *int      `json:"month,omitempty"`
	GrossSalary      float64   `json:"grossSalary"`
	Type             string    `json:"contrType"`
	Source           string    `json:"source"`
	Amount           *float64  `json:"amount,omitempty"`
	Units            *float64  `json:"units,omitempty"`
}

// Hash computes a unique SHA-256 hash for the Contribution instance.
func (d *Contribution) Hash() string {
	ps := func(s *string) string {
		if s == nil {
			return ""
		}

		return *s
	}

	pf := func(f *float64) float64 {
		if f == nil {
			return 0
		}

		return *f
	}

	data := fmt.Sprintf(
		"%s|%s|%s|%s|%s|%d|%d|%f|%s|%s|%f|%f",
		d.BasisID,
		d.Date,
		ps(d.ClosingDate),
		ps(d.OrganizationCode),
		ps(d.OrganizationName),
		d.Year,
		d.Month,
		d.GrossSalary,
		d.Type,
		d.Source,
		pf(d.Amount),
		pf(d.Units),
	)
	hash := sha256.Sum256([]byte(data))

	return hex.EncodeToString(hash[:])
}

// MapToModel converts the external Contribution representation into a
// model.Contribution entity suitable for persistence.
func (d *Contribution) MapToModel() model.Contribution {
	instance := model.Contribution{
		Hash:        d.Hash(),
		BasisID:     d.BasisID,
		Year:        d.Year,
		Month:       d.Month,
		GrossSalary: d.GrossSalary,
		Type:        d.Type,
		Source:      d.Source,
		Amount:      d.Amount,
		Units:       d.Units,
	}

	if d.Date != "" {
		fixed := strings.TrimSuffix(d.Date, "Z") + "+04:00"
		instance.Date, _ = time.Parse(time.RFC3339Nano, fixed)
	}

	if d.ClosingDate != nil {
		fixed := strings.TrimSuffix(*d.ClosingDate, "Z") + "+04:00"
		parsed, _ := time.Parse(time.RFC3339Nano, fixed)
		instance.ClosingDate = &parsed
	}

	if d.OrganizationCode != nil && d.OrganizationName != nil {
		instance.OrganizationCode = d.OrganizationCode
		instance.Organization = &model.Organization{
			Code: *d.OrganizationCode,
			Name: *d.OrganizationName,
		}
	}

	return instance
}
