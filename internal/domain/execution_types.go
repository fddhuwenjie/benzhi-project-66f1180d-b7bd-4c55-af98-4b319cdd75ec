package domain

import "time"

type ControlCheck struct {
	Control string `json:"control"`
	Passed  bool   `json:"passed"`
	Notes   string `json:"notes,omitempty"`
}

type ExecutionRecord struct {
	ID               string             `json:"id"`
	CaseID           string             `json:"case_id"`
	PlanVersion      int                `json:"plan_version"`
	PerformedAt      time.Time          `json:"performed_at"`
	CrewNames        []string           `json:"crew_names"`
	ActualMeasures   []string           `json:"actual_measures"`
	ControlChecks    []ControlCheck     `json:"control_checks"`
	EvidenceRefs     []PhotoRef         `json:"evidence_refs"`
	AcceptanceResult string             `json:"acceptance_result,omitempty"`
	Nonconformities  []string           `json:"nonconformities,omitempty"`
	SubmittedBy      string             `json:"submitted_by"`
	BatchNumber      int                `json:"batch_number"`
	Remediations     []IssueRemediation `json:"remediations,omitempty"`
}

type IssueRemediation struct {
	NonconformityID string `json:"nonconformity_id"`
	Description     string `json:"description"`
}

type Nonconformity struct {
	ID                  string `json:"id"`
	CompletionCriterion string `json:"completion_criterion"`
	Description         string `json:"description"`
	Status              string `json:"status"`
	ResolvedByBatches   []int  `json:"resolved_by_batches,omitempty"`
}

type AcceptanceRecord struct {
	Sequence         int             `json:"sequence"`
	Passed           bool            `json:"passed"`
	Inspector        string          `json:"inspector"`
	InspectedAt      time.Time       `json:"inspected_at"`
	CriterionResults []string        `json:"criterion_results"`
	Nonconformities  []string        `json:"nonconformities,omitempty"`
	Notes            string          `json:"notes,omitempty"`
	Items            []Nonconformity `json:"items,omitempty"`
}

type AuditEvent struct {
	ID         string    `json:"id"`
	CaseID     string    `json:"case_id"`
	EventType  string    `json:"event_type"`
	ActorName  string    `json:"actor_name"`
	OccurredAt time.Time `json:"occurred_at"`
	FromStatus Status    `json:"from_status"`
	ToStatus   Status    `json:"to_status"`
	Summary    string    `json:"summary"`
	RequestID  string    `json:"request_id"`
	Revision   int64     `json:"revision"`
}
