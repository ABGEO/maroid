package dto

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/abgeo/maroid/plugins/pensions/model"
)

// monthLayout is the form of a month in an answer, 2026-10.
const monthLayout = "2006-01"

// ContributionResponse is one contribution of the contributions page. A member that the
// pension service did not give is absent.
type ContributionResponse struct {
	ID          uuid.UUID         `json:"id"`
	PaidOn      string            `json:"paid_on"`
	Employer    *EmployerResponse `json:"employer,omitempty"`
	Source      string            `json:"source"`
	Type        string            `json:"type"`
	SalaryMonth string            `json:"salary_month,omitempty"`
	GrossSalary MoneyResponse     `json:"gross_salary"`
	Amount      *MoneyResponse    `json:"amount,omitempty"`
}

// EmployerResponse is an employer that a contribution names.
type EmployerResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// ContributionTotalResponse is the sum and the count of the contributions of a filter.
type ContributionTotalResponse struct {
	Amount MoneyResponse `json:"amount"`
	Count  int           `json:"count"`
}

// ContributionMonthResponse is the sum of the contributions of one month.
type ContributionMonthResponse struct {
	Month  string        `json:"month"`
	Amount MoneyResponse `json:"amount"`
}

// NewContributionResponseList converts the contributions to their answers.
func NewContributionResponseList(contributions []model.Contribution) []ContributionResponse {
	responses := make([]ContributionResponse, 0, len(contributions))

	for i := range contributions {
		contribution := &contributions[i]
		response := ContributionResponse{
			ID:          contribution.ID,
			PaidOn:      contribution.PaidOn.Format(time.DateOnly),
			Source:      contribution.Source,
			Type:        contribution.ContributionType,
			GrossSalary: NewMoneyResponse(contribution.GrossSalary),
		}

		if contribution.Employer != nil {
			response.Employer = &EmployerResponse{
				Code: contribution.Employer.Code,
				Name: contribution.Employer.Name,
			}
		}

		if contribution.SalaryYear != nil && contribution.SalaryMonth != nil {
			response.SalaryMonth = fmt.Sprintf(
				"%04d-%02d",
				*contribution.SalaryYear,
				*contribution.SalaryMonth,
			)
		}

		if contribution.Amount != nil {
			amount := NewMoneyResponse(*contribution.Amount)
			response.Amount = &amount
		}

		responses = append(responses, response)
	}

	return responses
}

// NewContributionTotalResponse converts the total to its answer.
func NewContributionTotalResponse(total model.ContributionTotal) ContributionTotalResponse {
	return ContributionTotalResponse{Amount: NewMoneyResponse(total.Amount), Count: total.Count}
}

// NewContributionMonthResponseList converts the sums of the months to their answers.
func NewContributionMonthResponseList(sums []model.MonthSum) []ContributionMonthResponse {
	responses := make([]ContributionMonthResponse, 0, len(sums))
	for _, sum := range sums {
		responses = append(responses, ContributionMonthResponse{
			Month:  sum.Month.Format(monthLayout),
			Amount: NewMoneyResponse(sum.Amount),
		})
	}

	return responses
}

// NewEmployerResponseList converts the employers to their answers.
func NewEmployerResponseList(employers []model.Employer) []EmployerResponse {
	responses := make([]EmployerResponse, 0, len(employers))
	for _, employer := range employers {
		responses = append(responses, EmployerResponse(employer))
	}

	return responses
}
