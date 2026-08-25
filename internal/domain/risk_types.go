package domain

import "time"

type RiskLevel string

const (
	RiskLow      RiskLevel = "low"
	RiskMedium   RiskLevel = "medium"
	RiskHigh     RiskLevel = "high"
	RiskCritical RiskLevel = "critical"
)

func (r RiskLevel) Valid() bool {
	return r == RiskLow || r == RiskMedium || r == RiskHigh || r == RiskCritical
}

func (r RiskLevel) Label() string {
	switch r {
	case RiskLow:
		return "低风险"
	case RiskMedium:
		return "中风险"
	case RiskHigh:
		return "高风险"
	case RiskCritical:
		return "重大风险"
	default:
		return string(r)
	}
}

type RiskFactor struct {
	RuleID        string    `json:"rule_id"`
	Title         string    `json:"title"`
	EvidenceField string    `json:"evidence_field"`
	Evidence      string    `json:"evidence"`
	Score         int       `json:"score"`
	Suggestion    string    `json:"suggestion"`
	Group         string    `json:"group"`
	Level         RiskLevel `json:"level"`
}

type RiskGroupResult struct {
	Group   string       `json:"group"`
	Score   int          `json:"score"`
	Factors []RiskFactor `json:"factors"`
}

type RiskAssessment struct {
	ID                string            `json:"id"`
	Sequence          int               `json:"sequence"`
	SurveyID          string            `json:"survey_id"`
	Revision          int64             `json:"revision"`
	AutomaticScore    int               `json:"automatic_score"`
	AutomaticLevel    RiskLevel         `json:"automatic_level"`
	FinalLevel        RiskLevel         `json:"final_level"`
	Factors           []RiskFactor      `json:"factors"`
	ManualLevel       RiskLevel         `json:"manual_level,omitempty"`
	ManualReason      string            `json:"manual_reason,omitempty"`
	Assessor          string            `json:"assessor_name"`
	AssessedAt        time.Time         `json:"assessed_at"`
	Groups            []RiskGroupResult `json:"groups"`
	DifferenceSummary string            `json:"difference_summary,omitempty"`
}
