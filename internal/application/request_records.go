package application

import (
	"encoding/json"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

// operation identifiers used for idempotency verification.
const (
	opCreateCase         = "create_case"
	opReviseCase         = "revise_case"
	opSubmitSurvey       = "submit_survey"
	opAssessRisk         = "assess_risk"
	opSavePlan           = "save_plan"
	opSubmitPlan         = "submit_plan"
	opReviewPlan         = "review_plan"
	opRecordExecution    = "record_execution"
	opCompleteExecution  = "complete_execution"
	opAccept             = "accept"
)

// verifyReplay checks that a prior request with the same request_id is a
// genuine replay: the same operation and an equivalent payload. When the
// operation or payload differs it returns a deterministic validation error
// scoped to request_id.
func verifyReplay(prior *domain.CareCase, requestID, operation, fingerprint string) error {
	record, ok := lookupRecord(prior, requestID)
	if !ok {
		// The request_id is tracked (it appears in audit events) but the
		// fingerprint ledger is unavailable (e.g. upgraded snapshot). Fall
		// back to operation-only verification using audit event types.
		if matchesOperationEvent(prior, requestID, operation) {
			return nil
		}
		return &domain.ValidationError{Field: "request_id", Message: "request_id 已用于其他操作"}
	}
	if record.Operation != operation {
		return &domain.ValidationError{Field: "request_id", Message: "request_id 已用于其他操作"}
	}
	if record.Fingerprint != fingerprint {
		return &domain.ValidationError{Field: "request_id", Message: "request_id 对应的请求负载不一致"}
	}
	return nil
}

func lookupRecord(c *domain.CareCase, requestID string) (domain.RequestRecord, bool) {
	if c == nil || c.RequestRecords == nil {
		return domain.RequestRecord{}, false
	}
	rec, ok := c.RequestRecords[requestID]
	return rec, ok
}

// matchesOperationEvent maps a service operation to the audit event types
// it can produce and reports whether the request_id appears on one of them.
func matchesOperationEvent(c *domain.CareCase, requestID, operation string) bool {
	expected, ok := operationEventTypes[operation]
	if !ok {
		return false
	}
	for _, event := range c.AuditEvents {
		if event.RequestID == requestID && expected[event.EventType] {
			return true
		}
	}
	return false
}

var operationEventTypes = map[string]map[string]bool{
	opCreateCase:        {"case.created": true},
	opReviseCase:        {"case.profile_revised": true},
	opSubmitSurvey:      {"survey.submitted": true},
	opAssessRisk:        {"risk.assessed": true, "risk.reassessed": true},
	opSavePlan:          {"plan.saved": true},
	opSubmitPlan:        {"plan.submitted": true},
	opReviewPlan:        {"plan.approved": true, "plan.rejected": true},
	opRecordExecution:   {"execution.batch_recorded": true, "execution.submitted": true},
	opCompleteExecution: {"execution.completed": true},
	opAccept:            {"acceptance.passed": true, "acceptance.rejected": true},
}

// recordRequest stamps the request ledger on the case so future replays can
// be validated. It is a no-op when the request is already recorded.
func recordRequest(c *domain.CareCase, requestID, operation, fingerprint string) {
	if c.RequestRecords == nil {
		c.RequestRecords = map[string]domain.RequestRecord{}
	}
	if _, exists := c.RequestRecords[requestID]; exists {
		return
	}
	c.RequestRecords[requestID] = domain.RequestRecord{Operation: operation, Fingerprint: fingerprint}
}

// canonicalJSON marshals value to a deterministic JSON string. Field order
// follows struct definitions, so fingerprints are stable across calls with
// equal inputs.
func canonicalJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

func fingerprintCreate(cmd CreateCaseCommand) string {
	return canonicalJSON(struct {
		TreeCode  string `json:"tree_code"`
		Species   string `json:"species"`
		Location  string `json:"location"`
		OwnerName string `json:"owner_name"`
		DueDate   string `json:"due_date"`
		Actor     string `json:"actor_name"`
	}{cmd.TreeCode, cmd.Species, cmd.Location, cmd.OwnerName, cmd.DueDate, cmd.Actor})
}

func fingerprintRevise(cmd ReviseCaseCommand) string {
	return canonicalJSON(struct {
		Species   string `json:"species"`
		Location  string `json:"location"`
		OwnerName string `json:"owner_name"`
		DueDate   string `json:"due_date"`
		Actor     string `json:"actor_name"`
	}{cmd.Species, cmd.Location, cmd.OwnerName, cmd.DueDate, cmd.Actor})
}

func fingerprintSurvey(cmd SubmitSurveyCommand) string {
	return canonicalJSON(struct {
		Crown       domain.Observation            `json:"crown_condition"`
		Trunk       domain.Observation            `json:"trunk_condition"`
		RootZone    domain.Observation            `json:"root_zone_condition"`
		Environment domain.EnvironmentObservation `json:"environment"`
		ObservedAt  time.Time                      `json:"observed_at"`
		Observer    string                         `json:"observer_name"`
		PhotoRefs   []domain.PhotoRef              `json:"photo_refs"`
		Actor       string                         `json:"actor_name"`
	}{cmd.Crown, cmd.Trunk, cmd.RootZone, cmd.Environment, cmd.ObservedAt, cmd.Observer, cmd.PhotoRefs, cmd.Actor})
}

func fingerprintAssess(cmd AssessRiskCommand) string {
	return canonicalJSON(struct {
		ManualLevel  domain.RiskLevel `json:"manual_level,omitempty"`
		ManualReason string           `json:"manual_reason,omitempty"`
		Assessor     string           `json:"assessor_name"`
		Actor        string           `json:"actor_name"`
	}{cmd.ManualLevel, cmd.ManualReason, cmd.Assessor, cmd.Actor})
}

func fingerprintSavePlan(cmd SavePlanCommand) string {
	return canonicalJSON(struct {
		Measures             []domain.PlanMeasure         `json:"measures"`
		Materials            []string                     `json:"materials"`
		WorkWindow           string                       `json:"work_window"`
		SafetyControls       []string                     `json:"safety_controls"`
		CompletionCriteria   []string                     `json:"completion_criteria"`
		PreparedBy           string                       `json:"prepared_by"`
		Exemptions           []domain.RiskExemption       `json:"exemptions,omitempty"`
		RemediationResponses []domain.RemediationResponse `json:"remediation_responses,omitempty"`
		Actor                string                       `json:"actor_name"`
	}{cmd.Measures, cmd.Materials, cmd.WorkWindow, cmd.SafetyControls, cmd.CompletionCriteria, cmd.PreparedBy, cmd.Exemptions, cmd.RemediationResponses, cmd.Actor})
}

func fingerprintSubmitPlan(cmd SubmitPlanCommand) string {
	return canonicalJSON(struct {
		Actor string `json:"actor_name"`
	}{cmd.Actor})
}

func fingerprintReviewPlan(cmd ReviewPlanCommand) string {
	return canonicalJSON(struct {
		Approved bool                   `json:"approved"`
		Reviewer string                 `json:"reviewer"`
		Opinions []domain.ReviewOpinion `json:"opinions"`
		Actor    string                 `json:"actor_name"`
	}{cmd.Approved, cmd.Reviewer, cmd.Opinions, cmd.Actor})
}

func fingerprintRecordExecution(cmd RecordExecutionCommand) string {
	return canonicalJSON(struct {
		PerformedAt    time.Time                  `json:"performed_at"`
		CrewNames      []string                   `json:"crew_names"`
		ActualMeasures []string                   `json:"actual_measures"`
		ControlChecks  []domain.ControlCheck      `json:"control_checks"`
		EvidenceRefs   []domain.PhotoRef          `json:"evidence_refs"`
		SubmittedBy    string                     `json:"submitted_by"`
		Remediations   []domain.IssueRemediation `json:"remediations,omitempty"`
		Actor          string                     `json:"actor_name"`
	}{cmd.PerformedAt, cmd.CrewNames, cmd.ActualMeasures, cmd.ControlChecks, cmd.EvidenceRefs, cmd.SubmittedBy, cmd.Remediations, cmd.Actor})
}

func fingerprintCompleteExecution(cmd CompleteExecutionCommand) string {
	return canonicalJSON(struct {
		Actor string `json:"actor_name"`
	}{cmd.Actor})
}

func fingerprintAccept(cmd AcceptCommand) string {
	return canonicalJSON(struct {
		Passed           bool                     `json:"passed"`
		Inspector        string                   `json:"inspector"`
		InspectedAt      time.Time                `json:"inspected_at"`
		CriterionResults []string                 `json:"criterion_results"`
		Nonconformities  []domain.Nonconformity   `json:"nonconformities"`
		Notes            string                   `json:"notes"`
		Actor            string                   `json:"actor_name"`
	}{cmd.Passed, cmd.Inspector, cmd.InspectedAt, cmd.CriterionResults, cmd.Nonconformities, cmd.Notes, cmd.Actor})
}
