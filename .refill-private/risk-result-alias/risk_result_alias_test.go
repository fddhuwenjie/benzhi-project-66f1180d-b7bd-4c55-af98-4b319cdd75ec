package riskresultalias_test

import (
	"testing"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
)

func TestRiskPreviewResultDoesNotShareCacheBuffers(t *testing.T) {
	engine := risk.NewEngine()
	first := engine.Evaluate(domain.ConditionSurvey{
		Crown: domain.Observation{Condition: domain.ConditionPoor, Notes: "首次任务树冠衰弱"},
	})
	if len(first.Factors) == 0 {
		t.Fatal("首次评估未产生风险因子")
	}
	wantRule := first.Factors[0].RuleID

	engine.Evaluate(domain.ConditionSurvey{
		Crown: domain.Observation{Condition: domain.ConditionCritical, Notes: "另一任务树冠有断枝"},
	})
	if first.Factors[0].RuleID != wantRule {
		t.Fatalf("第一次评估结果被后续缓存复用覆盖：got=%s want=%s", first.Factors[0].RuleID, wantRule)
	}
}
