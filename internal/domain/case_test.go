package domain

import (
	"errors"
	"testing"
	"time"
)

var testNow = time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)

func newTestCase(t *testing.T) *CareCase {
	t.Helper()
	c, err := NewCareCase(NewCaseInput{ID: "case-test", TreeCode: "GT-001", Species: "银杏", Location: "中心公园北门", OwnerName: "张工", DueDate: "2026-12-31", Actor: "张工", RequestID: "r1", Now: testNow})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func validSurvey() ConditionSurvey {
	return ConditionSurvey{ID: "survey-1", Crown: Observation{Condition: ConditionPoor, Notes: "枯枝较多"}, Trunk: Observation{Condition: ConditionAttention, Notes: "轻微损伤"}, RootZone: Observation{Condition: ConditionGood, Notes: "根区稳定"}, Environment: EnvironmentObservation{Notes: "周边开阔"}, ObservedAt: testNow, Observer: "调查员"}
}

func validPlan() CarePlan {
	return CarePlan{ID: "plan-1", PreparedBy: "编制人", Measures: []PlanMeasure{{Title: "修剪", Description: "清除枯枝"}}, Materials: []string{"保护剂"}, WorkWindow: "2026-09-01", SafetyControls: []string{"完成围挡"}, CompletionCriteria: []string{"枯枝清除完成"}}
}

func TestCareCaseLifecycleAndAudit(t *testing.T) {
	c := newTestCase(t)
	if err := c.SubmitSurvey(validSurvey(), "调查员", "r2", testNow.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	risk := RiskAssessment{AutomaticScore: 40, AutomaticLevel: RiskHigh, FinalLevel: RiskHigh, Assessor: "评估员", AssessedAt: testNow}
	if err := c.ApplyRisk(risk, "评估员", "r3", testNow.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.SavePlan(validPlan(), "编制人", "r4", testNow.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan("编制人", "r5", testNow.Add(4*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.ReviewPlan(true, []ReviewOpinion{{Item: "安全", Opinion: "同意"}}, "审核人", "r6", testNow.Add(5*time.Hour)); err != nil {
		t.Fatal(err)
	}
	execution := ExecutionRecord{ID: "exec-1", PerformedAt: testNow, CrewNames: []string{"甲"}, ActualMeasures: []string{"完成修剪"}, ControlChecks: []ControlCheck{{Control: "完成围挡", Passed: true}}, EvidenceRefs: []PhotoRef{{Name: "完工.jpg", URL: "evidence/done.jpg"}}, SubmittedBy: "现场人员"}
	if err := c.RecordExecution(execution, "现场人员", "r7", testNow.Add(6*time.Hour)); err != nil {
		t.Fatal(err)
	}
	acceptance := AcceptanceRecord{Passed: true, Inspector: "张工", InspectedAt: testNow, CriterionResults: []string{"现场核验通过"}}
	if err := c.Accept(acceptance, "张工", "r8", testNow.Add(7*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusClosed {
		t.Fatalf("want closed, got %s", c.Status)
	}
	if c.Revision != 8 || len(c.AuditEvents) != 8 {
		t.Fatalf("revision=%d events=%d", c.Revision, len(c.AuditEvents))
	}
	for i, event := range c.AuditEvents {
		if event.Revision != int64(i+1) {
			t.Fatalf("event %d revision=%d", i, event.Revision)
		}
	}
}

func TestCareCaseRejectsInvalidTransitionsAndIncompleteControls(t *testing.T) {
	c := newTestCase(t)
	err := c.SubmitPlan("编制人", "bad", testNow)
	var stateErr *StateError
	if !errors.As(err, &stateErr) {
		t.Fatalf("want StateError, got %v", err)
	}
	if err := c.SubmitSurvey(validSurvey(), "调查员", "r2", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyRisk(RiskAssessment{AutomaticLevel: RiskLow, FinalLevel: RiskLow, AssessedAt: testNow}, "评估员", "r3", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.SavePlan(validPlan(), "编制人", "r4", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan("编制人", "r5", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.ReviewPlan(true, []ReviewOpinion{{Item: "安全", Opinion: "同意"}}, "审核人", "r6", testNow); err != nil {
		t.Fatal(err)
	}
	err = c.RecordExecution(ExecutionRecord{ID: "exec", PerformedAt: testNow, CrewNames: []string{"甲"}, ActualMeasures: []string{"修剪"}, EvidenceRefs: []PhotoRef{{Name: "x", URL: "x"}}, SubmittedBy: "甲"}, "甲", "r7", testNow)
	var validation *ValidationError
	if !errors.As(err, &validation) || validation.Field != "control_checks" {
		t.Fatalf("want control validation, got %v", err)
	}
}

func TestReviewRejectionCreatesNewPlanVersion(t *testing.T) {
	c := newTestCase(t)
	if err := c.SubmitSurvey(validSurvey(), "调查员", "r2", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.ApplyRisk(RiskAssessment{AutomaticLevel: RiskLow, FinalLevel: RiskLow, AssessedAt: testNow}, "评估员", "r3", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.SavePlan(validPlan(), "编制人", "r4", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitPlan("编制人", "r5", testNow); err != nil {
		t.Fatal(err)
	}
	if err := c.ReviewPlan(false, []ReviewOpinion{{Item: "措施", Opinion: "补充支撑"}}, "审核人", "r6", testNow); err != nil {
		t.Fatal(err)
	}
	p2 := validPlan()
	p2.ID = "plan-2"
	if err := c.SavePlan(p2, "编制人", "r7", testNow); err != nil {
		t.Fatal(err)
	}
	if len(c.Plans) != 2 || c.Plans[1].Version != 2 || c.Plans[0].ReviewStatus != ReviewRejected {
		t.Fatalf("unexpected plans: %+v", c.Plans)
	}
}
