package duplicatemeasuretitle_test

import (
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func TestOneExecutionCannotSatisfyDuplicatePlanMeasures(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	c, err := domain.NewCareCase(domain.NewCaseInput{ID: "case-duplicate-measure", TreeCode: "GT-DUP-M", Species: "银杏", Location: "北门", OwnerName: "甲", DueDate: "2026-12-31", Actor: "甲", RequestID: "create", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	c.Status = domain.StatusAssessed
	c.Risk = &domain.RiskAssessment{Revision: c.Revision, Factors: []domain.RiskFactor{{RuleID: "trunk.high", Title: "树干结构风险", Level: domain.RiskHigh}}}
	p := domain.CarePlan{
		ID: "plan-duplicate", PreparedBy: "编制人", Materials: []string{"支撑材料"}, WorkWindow: "2026-09-01",
		SafetyControls: []string{"完成围挡"}, CompletionCriteria: []string{"腐朽清理完成", "支撑安装完成"},
		Measures: []domain.PlanMeasure{
			{Title: "树干处置", Description: "清理腐朽组织", RiskRuleIDs: []string{"trunk.high"}, ControlPoints: []string{"完成围挡"}, CompletionStandard: "腐朽清理完成"},
			{Title: "树干处置", Description: "安装结构支撑", RiskRuleIDs: []string{"trunk.high"}, ControlPoints: []string{"完成围挡"}, CompletionStandard: "支撑安装完成"},
		},
	}
	if err := c.SavePlanForAssessment(p, "编制人", "save", now); err != nil {
		return
	}
	if err := c.SubmitPlanCovered("编制人", "submit", now); err != nil {
		t.Fatal(err)
	}
	opinions := make([]domain.ReviewOpinion, 0, len(domain.ReviewChecklist))
	for _, item := range domain.ReviewChecklist {
		opinions = append(opinions, domain.ReviewOpinion{Item: item, Result: "passed", Opinion: "符合要求"})
	}
	if err := c.ReviewPlanChecklist(true, opinions, "审核人", "review", now); err != nil {
		t.Fatal(err)
	}
	execution := domain.ExecutionRecord{
		ID: "execution-one", PerformedAt: now.Add(time.Hour), CrewNames: []string{"作业员"},
		ActualMeasures: []string{"树干处置"}, ControlChecks: []domain.ControlCheck{{Control: "完成围挡", Passed: true}},
		EvidenceRefs: []domain.PhotoRef{{Name: "现场.jpg", URL: "evidence/one.jpg"}}, SubmittedBy: "作业员",
	}
	if err := c.RegisterExecution(execution, "作业员", "execute", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := c.CompleteExecution("负责人", "complete", now.Add(2*time.Hour)); err == nil {
		t.Fatal("one title-only execution satisfied two distinct plan measures")
	}
}
