package partialcompletionstatepoison

import (
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func TestFailedCompletionDoesNotResolvePendingNonconformity(t *testing.T) {
	now := time.Date(2026, 8, 25, 12, 0, 0, 0, time.UTC)
	reviewed := now.Add(-time.Hour)
	caseFile := &domain.CareCase{
		ID: "case-partial-completion", Status: domain.StatusPendingAcceptance, Revision: 1,
		Plans: []domain.CarePlan{{Version: 1, ReviewStatus: domain.ReviewApproved, ReviewedAt: &reviewed,
			Measures:       []domain.PlanMeasure{{Title: "树干加固"}, {Title: "现场恢复"}},
			SafetyControls: []string{"围挡", "监护"}, CompletionCriteria: []string{"加固完成"}}},
	}
	// 先走一次不通过验收，生成可供后续实施批次引用的待整改项。
	if err := caseFile.AcceptStructured(domain.AcceptanceRecord{
		Inspector: "验收员", InspectedAt: now, Passed: false,
		CriterionResults: []string{"未通过"},
		Items:            []domain.Nonconformity{{CompletionCriterion: "加固完成", Description: "围挡缺口未封闭"}},
	}, "验收员", "accept-reject", now); err != nil {
		t.Fatal(err)
	}
	batch := domain.ExecutionRecord{
		ID: "batch-partial", PerformedAt: now.Add(time.Hour), CrewNames: []string{"甲"},
		ActualMeasures: []string{"树干加固"}, ControlChecks: []domain.ControlCheck{{Control: "围挡", Passed: true}},
		EvidenceRefs: []domain.PhotoRef{{Name: "整改.jpg", URL: "evidence/partial.jpg"}}, SubmittedBy: "甲",
		Remediations: []domain.IssueRemediation{{NonconformityID: "A1-NC1", Description: "补齐并固定围挡"}},
	}
	if err := caseFile.RegisterExecution(batch, "甲", "batch-partial", now.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := caseFile.CompleteExecution("负责人", "complete-partial", now.Add(2*time.Hour)); err == nil {
		t.Fatal("expected missing measure and control error")
	}
	if got := caseFile.Acceptances[0].Items[0].Status; got != "pending" {
		t.Fatalf("failed completion changed pending nonconformity status to %q", got)
	}
}
