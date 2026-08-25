package application

import (
	"context"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

type CaseRepository interface {
	Get(context.Context, string) (*domain.CareCase, error)
	List(context.Context) ([]*domain.CareCase, error)
	LookupRequest(context.Context, string) (*domain.CareCase, bool, error)
	LookupActiveTreeCode(context.Context, string) (*domain.CareCase, bool, error)
	Commit(context.Context, *domain.CareCase, int64, string) (*domain.CareCase, bool, error)
}

type Clock interface{ Now() time.Time }

type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now().UTC() }

type IDGenerator interface{ NewID(prefix string) string }

type CaseSummary struct {
	ID            string           `json:"id"`
	TreeCode      string           `json:"tree_code"`
	Species       string           `json:"species"`
	Location      string           `json:"location"`
	OwnerName     string           `json:"owner_name"`
	DueDate       string           `json:"due_date"`
	Status        domain.Status    `json:"status"`
	StatusLabel   string           `json:"status_label"`
	RiskLevel     domain.RiskLevel `json:"risk_level,omitempty"`
	RiskLabel     string           `json:"risk_label,omitempty"`
	Revision      int64            `json:"revision"`
	UpdatedAt     time.Time        `json:"updated_at"`
	NextAction    string           `json:"next_action"`
	DeadlineLevel string           `json:"deadline_level"`
	DeadlineLabel string           `json:"deadline_label"`
}

type CaseDetail struct {
	Case        *domain.CareCase    `json:"case"`
	StatusLabel string              `json:"status_label"`
	RiskLabel   string              `json:"risk_label,omitempty"`
	NextAction  string              `json:"next_action"`
	Timeline    []domain.AuditEvent `json:"timeline"`
}

type Dashboard struct {
	Cases      []CaseSummary       `json:"cases"`
	Counts     map[string]int      `json:"counts"`
	Statistics DashboardStatistics `json:"statistics"`
	Query      CaseQuery           `json:"query"`
}

type CaseQuery struct {
	Keyword       string `json:"keyword,omitempty"`
	OwnerName     string `json:"owner_name,omitempty"`
	Status        string `json:"status,omitempty"`
	RiskLevel     string `json:"risk_level,omitempty"`
	DeadlineLevel string `json:"deadline_level,omitempty"`
}

type DashboardStatistics struct {
	Statuses       map[string]int `json:"statuses"`
	RiskLevels     map[string]int `json:"risk_levels"`
	DeadlineLevels map[string]int `json:"deadline_levels"`
	NextActions    map[string]int `json:"next_actions"`
}
