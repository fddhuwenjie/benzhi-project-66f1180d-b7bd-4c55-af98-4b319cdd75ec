package risk

import (
	"reflect"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func TestEvaluateIsStableAndExplainable(t *testing.T) {
	survey := domain.ConditionSurvey{Crown: domain.Observation{Condition: domain.ConditionCritical, Notes: "大枝断裂"}, Trunk: domain.Observation{Condition: domain.ConditionPoor, Notes: "树干空洞"}, RootZone: domain.Observation{Condition: domain.ConditionAttention, Notes: "根区踩踏"}, Environment: domain.EnvironmentObservation{Notes: "邻近施工", NearbyConstruction: true, HeavyTraffic: true}, ObservedAt: time.Now(), Observer: "甲"}
	engine := NewEngine()
	first, second := engine.Evaluate(survey), engine.Evaluate(survey)
	if !reflect.DeepEqual(first, second) {
		t.Fatal("same survey produced different results")
	}
	if first.Level != domain.RiskCritical || first.Score < 60 {
		t.Fatalf("score=%d level=%s", first.Score, first.Level)
	}
	for _, factor := range first.Factors {
		if factor.RuleID == "" || factor.EvidenceField == "" || factor.Suggestion == "" {
			t.Fatalf("incomplete factor: %+v", factor)
		}
	}
}

func TestManualOverrideRequiresReason(t *testing.T) {
	result := Result{Score: 5, Level: domain.RiskLow}
	if _, err := BuildAssessment(result, domain.RiskHigh, "", "评估员"); err == nil {
		t.Fatal("expected missing reason error")
	}
	assessment, err := BuildAssessment(result, domain.RiskHigh, "发现未结构化记录的倾斜", "评估员")
	if err != nil {
		t.Fatal(err)
	}
	if assessment.AutomaticLevel != domain.RiskLow || assessment.FinalLevel != domain.RiskHigh {
		t.Fatalf("override difference not retained: %+v", assessment)
	}
}
