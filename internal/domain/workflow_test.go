package domain

import (
	"errors"
	"testing"
	"time"
)

func TestValidateSurveyAtRequiresTimelyPartEvidence(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	survey := ConditionSurvey{
		ID: "survey-evidence", Crown: Observation{Condition: ConditionGood, Notes: "正常"},
		Trunk:    Observation{Condition: ConditionCritical, Notes: "主干开裂"},
		RootZone: Observation{Condition: ConditionGood, Notes: "正常"}, Environment: EnvironmentObservation{Notes: "环境稳定"},
		ObservedAt: now.Add(-time.Hour), Observer: "调查员",
		PhotoRefs: []PhotoRef{{Name: "树冠.jpg", URL: "photos/crown.jpg", Part: PhotoPartCrown, TakenAt: now.Add(-time.Hour).Format(time.RFC3339)}},
	}
	err := ValidateSurveyAt(survey, now)
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Field != "photo_refs" {
		t.Fatalf("want missing trunk evidence, got %v", err)
	}
	survey.PhotoRefs = append(survey.PhotoRefs, PhotoRef{Name: "树干.jpg", URL: "photos/trunk.jpg", Part: PhotoPartTrunk, TakenAt: now.Add(-30 * time.Minute).Format(time.RFC3339), Caption: "裂缝近景"})
	if err := ValidateSurveyAt(survey, now); err != nil {
		t.Fatal(err)
	}
	survey.ObservedAt = now.Add(time.Minute)
	if err := ValidateSurveyAt(survey, now); err == nil {
		t.Fatal("expected future observation rejection")
	}
}

func TestRiskOverrideAndSurveyExpiryBoundaries(t *testing.T) {
	risk := RiskAssessment{AutomaticLevel: RiskCritical, FinalLevel: RiskMedium, AutomaticScore: 70, ManualLevel: RiskMedium, ManualReason: "现场复核"}
	if err := validateRiskOverride(risk); err == nil {
		t.Fatal("expected critical to medium rejection")
	}
	risk.FinalLevel, risk.ManualLevel = RiskHigh, RiskHigh
	if err := validateRiskOverride(risk); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	if surveyExpired(now.AddDate(0, 0, -30), now) {
		t.Fatal("exactly thirty calendar days should remain valid")
	}
	if !surveyExpired(now.AddDate(0, 0, -31), now) {
		t.Fatal("thirty-one calendar days should be expired")
	}
}

func TestCoverageReviewAndBatchedCompletion(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	c := newTestCase(t)
	c.Status = StatusAssessed
	c.Risk = &RiskAssessment{Revision: c.Revision, Factors: []RiskFactor{{RuleID: "trunk.high", Title: "树干高风险", Level: RiskHigh}}}
	plan := CarePlan{ID: "plan-covered", PreparedBy: "编制人", Materials: []string{"支撑材料"}, WorkWindow: "2026-09-01", SafetyControls: []string{"围挡", "监护"}, CompletionCriteria: []string{"加固完成", "现场恢复"}, Measures: []PlanMeasure{{Title: "树干加固", Description: "安装支撑", RiskRuleIDs: []string{"trunk.high"}, ControlPoints: []string{"围挡"}, CompletionStandard: "加固完成"}, {Title: "现场恢复", Description: "移除临设", RiskRuleIDs: []string{"trunk.high"}, ControlPoints: []string{"监护"}, CompletionStandard: "现场恢复"}}}
	if err := c.SavePlanForAssessment(plan, "编制人", "plan-save", now); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlanCovered("编制人", "plan-submit", now); err != nil {
		t.Fatal(err)
	}
	opinions := make([]ReviewOpinion, 0, len(ReviewChecklist))
	for _, item := range ReviewChecklist {
		opinions = append(opinions, ReviewOpinion{Item: item, Result: "passed", Opinion: "符合要求"})
	}
	if err := c.ReviewPlanChecklist(true, opinions, "审核人", "review", now); err != nil {
		t.Fatal(err)
	}
	first := ExecutionRecord{ID: "batch-1", PerformedAt: now.Add(time.Hour), CrewNames: []string{"甲"}, ActualMeasures: []string{"树干加固"}, ControlChecks: []ControlCheck{{Control: "围挡", Passed: true}}, EvidenceRefs: []PhotoRef{{Name: "一.jpg", URL: "evidence/1.jpg"}}, SubmittedBy: "甲"}
	if err := c.RegisterExecution(first, "甲", "batch-one", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteExecution("负责人", "complete-early", now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected missing second measure and control")
	}
	second := ExecutionRecord{ID: "batch-2", PerformedAt: now.Add(2 * time.Hour), CrewNames: []string{"乙"}, ActualMeasures: []string{"现场恢复"}, ControlChecks: []ControlCheck{{Control: "监护", Passed: true}}, EvidenceRefs: []PhotoRef{{Name: "二.jpg", URL: "evidence/2.jpg"}}, SubmittedBy: "乙"}
	if err := c.RegisterExecution(second, "乙", "batch-two", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteExecution("负责人", "complete", now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusPendingAcceptance || len(c.Executions) != 2 || c.Executions[1].BatchNumber != 2 {
		t.Fatalf("unexpected completion state: status=%s batches=%d", c.Status, len(c.Executions))
	}
}

func TestAcceptanceIssueIsResolvedOnlyByCompletion(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	reviewed := now.Add(-time.Hour)
	c := &CareCase{ID: "case-remediation", Status: StatusPendingAcceptance, Revision: 1, Plans: []CarePlan{{Version: 1, ReviewStatus: ReviewApproved, ReviewedAt: &reviewed, Measures: []PlanMeasure{{Title: "恢复围挡"}}, SafetyControls: []string{"现场隔离"}, CompletionCriteria: []string{"围挡恢复"}}}}
	a := AcceptanceRecord{Passed: false, Inspector: "验收员", InspectedAt: now, CriterionResults: []string{"未通过"}, Items: []Nonconformity{{CompletionCriterion: "围挡恢复", Description: "围挡缺口未封闭"}}}
	if err := c.AcceptStructured(a, "验收员", "accept-reject", now); err != nil {
		t.Fatal(err)
	}
	if c.Acceptances[0].Items[0].ID != "A1-NC1" || c.Acceptances[0].Items[0].Status != "pending" {
		t.Fatalf("unexpected issue: %+v", c.Acceptances[0].Items[0])
	}
	batch := ExecutionRecord{ID: "fix-1", PerformedAt: now.Add(time.Hour), CrewNames: []string{"甲"}, ActualMeasures: []string{"恢复围挡"}, ControlChecks: []ControlCheck{{Control: "现场隔离", Passed: true}}, EvidenceRefs: []PhotoRef{{Name: "整改.jpg", URL: "evidence/fix.jpg"}}, SubmittedBy: "甲", Remediations: []IssueRemediation{{NonconformityID: "A1-NC1", Description: "补齐并固定围挡"}}}
	if err := c.RegisterExecution(batch, "甲", "fix", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.Acceptances[0].Items[0].Status != "pending" {
		t.Fatal("registering a batch must not resolve an issue")
	}
	if err := c.CompleteExecution("负责人", "fix-complete", now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.Acceptances[0].Items[0].Status != "resolved" || c.Status != StatusPendingAcceptance {
		t.Fatalf("issue was not resolved by completion: %+v", c.Acceptances[0].Items[0])
	}
}
