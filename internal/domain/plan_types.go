package domain

import "time"

type PlanMeasure struct {
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	RiskRuleID         string   `json:"risk_rule_id,omitempty"`
	RiskRuleIDs        []string `json:"risk_rule_ids,omitempty"`
	ControlPoints      []string `json:"control_points,omitempty"`
	CompletionStandard string   `json:"completion_standard,omitempty"`
}

type RiskExemption struct {
	RiskRuleID string `json:"risk_rule_id"`
	Reason     string `json:"reason"`
}

type RiskCoverage struct {
	RiskRuleID          string    `json:"risk_rule_id"`
	RiskTitle           string    `json:"risk_title"`
	RiskLevel           RiskLevel `json:"risk_level"`
	MeasureTitles       []string  `json:"measure_titles,omitempty"`
	ControlPoints       []string  `json:"control_points,omitempty"`
	CompletionStandards []string  `json:"completion_standards,omitempty"`
	ExemptionReason     string    `json:"exemption_reason,omitempty"`
	Covered             bool      `json:"covered"`
}

type ReviewStatus string

const (
	ReviewDraft    ReviewStatus = "draft"
	ReviewPending  ReviewStatus = "pending"
	ReviewRejected ReviewStatus = "rejected"
	ReviewApproved ReviewStatus = "approved"
)

type ReviewOpinion struct {
	ID      string `json:"id,omitempty"`
	Item    string `json:"item"`
	Result  string `json:"result,omitempty"`
	Opinion string `json:"opinion"`
}

type RemediationResponse struct {
	OpinionID string `json:"opinion_id"`
	Response  string `json:"response"`
}

type CarePlan struct {
	ID                   string                `json:"id"`
	CaseID               string                `json:"case_id"`
	Version              int                   `json:"version"`
	Measures             []PlanMeasure         `json:"measures"`
	Materials            []string              `json:"materials"`
	WorkWindow           string                `json:"work_window"`
	SafetyControls       []string              `json:"safety_controls"`
	CompletionCriteria   []string              `json:"completion_criteria"`
	ReviewStatus         ReviewStatus          `json:"review_status"`
	ReviewNotes          []ReviewOpinion       `json:"review_notes,omitempty"`
	PreparedBy           string                `json:"prepared_by"`
	CreatedAt            time.Time             `json:"created_at"`
	ReviewedBy           string                `json:"reviewed_by,omitempty"`
	ReviewedAt           *time.Time            `json:"reviewed_at,omitempty"`
	AssessmentRevision   int64                 `json:"assessment_revision"`
	RiskRuleIDs          []string              `json:"risk_rule_ids"`
	Coverage             []RiskCoverage        `json:"coverage"`
	Exemptions           []RiskExemption       `json:"exemptions,omitempty"`
	AssessmentMismatch   bool                  `json:"assessment_mismatch"`
	RemediationResponses []RemediationResponse `json:"remediation_responses,omitempty"`
}
