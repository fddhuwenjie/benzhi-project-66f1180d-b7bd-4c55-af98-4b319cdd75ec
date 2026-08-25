package application

import (
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

type CommandMeta struct {
	RequestID        string `json:"request_id"`
	ExpectedRevision int64  `json:"revision"`
	Actor            string `json:"actor_name"`
}

type CreateCaseCommand struct {
	RequestID string `json:"request_id"`
	Actor     string `json:"actor_name"`
	TreeCode  string `json:"tree_code"`
	Species   string `json:"species"`
	Location  string `json:"location"`
	OwnerName string `json:"owner_name"`
	DueDate   string `json:"due_date"`
}

type ReviseCaseCommand struct {
	CommandMeta
	CaseID    string `json:"-"`
	Species   string `json:"species"`
	Location  string `json:"location"`
	OwnerName string `json:"owner_name"`
	DueDate   string `json:"due_date"`
}

type SubmitSurveyCommand struct {
	CommandMeta
	CaseID      string                        `json:"-"`
	Crown       domain.Observation            `json:"crown_condition"`
	Trunk       domain.Observation            `json:"trunk_condition"`
	RootZone    domain.Observation            `json:"root_zone_condition"`
	Environment domain.EnvironmentObservation `json:"environment"`
	ObservedAt  time.Time                     `json:"observed_at"`
	Observer    string                        `json:"observer_name"`
	PhotoRefs   []domain.PhotoRef             `json:"photo_refs"`
}

type AssessRiskCommand struct {
	CommandMeta
	CaseID       string           `json:"-"`
	ManualLevel  domain.RiskLevel `json:"manual_level,omitempty"`
	ManualReason string           `json:"manual_reason,omitempty"`
	Assessor     string           `json:"assessor_name"`
}

type SavePlanCommand struct {
	CommandMeta
	CaseID               string                       `json:"-"`
	Measures             []domain.PlanMeasure         `json:"measures"`
	Materials            []string                     `json:"materials"`
	WorkWindow           string                       `json:"work_window"`
	SafetyControls       []string                     `json:"safety_controls"`
	CompletionCriteria   []string                     `json:"completion_criteria"`
	PreparedBy           string                       `json:"prepared_by"`
	Exemptions           []domain.RiskExemption       `json:"exemptions,omitempty"`
	RemediationResponses []domain.RemediationResponse `json:"remediation_responses,omitempty"`
}

type SubmitPlanCommand struct {
	CommandMeta
	CaseID string `json:"-"`
}

type ReviewPlanCommand struct {
	CommandMeta
	CaseID   string                 `json:"-"`
	Approved bool                   `json:"approved"`
	Reviewer string                 `json:"reviewer"`
	Opinions []domain.ReviewOpinion `json:"opinions"`
}

type RecordExecutionCommand struct {
	CommandMeta
	CaseID         string                    `json:"-"`
	PerformedAt    time.Time                 `json:"performed_at"`
	CrewNames      []string                  `json:"crew_names"`
	ActualMeasures []string                  `json:"actual_measures"`
	ControlChecks  []domain.ControlCheck     `json:"control_checks"`
	EvidenceRefs   []domain.PhotoRef         `json:"evidence_refs"`
	SubmittedBy    string                    `json:"submitted_by"`
	Remediations   []domain.IssueRemediation `json:"remediations,omitempty"`
}

type CompleteExecutionCommand struct {
	CommandMeta
	CaseID string `json:"-"`
}

type AcceptCommand struct {
	CommandMeta
	CaseID           string                 `json:"-"`
	Passed           bool                   `json:"passed"`
	Inspector        string                 `json:"inspector"`
	InspectedAt      time.Time              `json:"inspected_at"`
	CriterionResults []string               `json:"criterion_results"`
	Nonconformities  []domain.Nonconformity `json:"nonconformities"`
	Notes            string                 `json:"notes"`
}
