package risk

import (
	"fmt"
	"strings"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

type conditionRule struct {
	component, field  string
	level             domain.ConditionLevel
	score             int
	title, suggestion string
}

// Engine is a stateless risk evaluator; each Evaluate call returns a Result
// whose slices are independent and remain stable after subsequent calls.
type Engine struct{}

var conditionRules = []conditionRule{
	{"树冠", "crown_condition.condition", domain.ConditionAttention, 10, "树冠出现轻度异常", "安排枯枝清理并持续监测树势"},
	{"树冠", "crown_condition.condition", domain.ConditionPoor, 22, "树冠衰弱或存在明显枯损", "实施专业修剪与树势复壮"},
	{"树冠", "crown_condition.condition", domain.ConditionCritical, 38, "树冠存在大枝断裂等紧迫风险", "立即设置警戒并处置危险枝"},
	{"树干", "trunk_condition.condition", domain.ConditionAttention, 12, "树干发现轻微损伤", "清理创面并定期复查"},
	{"树干", "trunk_condition.condition", domain.ConditionPoor, 26, "树干存在腐朽、空洞或裂缝", "开展腐朽检测并制定支撑加固措施"},
	{"树干", "trunk_condition.condition", domain.ConditionCritical, 42, "树干结构稳定性受到严重影响", "立即隔离现场并组织结构安全专项评估"},
	{"根区", "root_zone_condition.condition", domain.ConditionAttention, 10, "根区出现轻度裸露或踩踏", "改善覆盖并限制进入根区"},
	{"根区", "root_zone_condition.condition", domain.ConditionPoor, 24, "根区压实或根系受损", "实施土壤改良与根系保护"},
	{"根区", "root_zone_condition.condition", domain.ConditionCritical, 40, "根系严重受损或锚固能力不足", "设置临时支撑并开展根系专项处置"},
}

func (e *Engine) Evaluate(s domain.ConditionSurvey) Result {
	factors := make([]domain.RiskFactor, 0)
	suggestions := make([]string, 0)
	result := Result{Factors: factors, Suggestions: suggestions}
	levels := map[string]domain.ConditionLevel{"树冠": s.Crown.Condition, "树干": s.Trunk.Condition, "根区": s.RootZone.Condition}
	notes := map[string]string{"树冠": s.Crown.Notes, "树干": s.Trunk.Notes, "根区": s.RootZone.Notes}
	for _, rule := range conditionRules {
		if levels[rule.component] != rule.level {
			continue
		}
		factor := domain.RiskFactor{
			RuleID: fmt.Sprintf("condition.%s.%s", rule.component, rule.level), Title: rule.title,
			EvidenceField: rule.field, Evidence: notes[rule.component], Score: rule.score, Suggestion: rule.suggestion,
			Group: groupName(rule.component), Level: factorLevel(rule.score),
		}
		result.Factors = append(result.Factors, factor)
		result.Score += rule.score
		result.Suggestions = append(result.Suggestions, rule.suggestion)
	}
	environmentRules := []struct {
		hit                          bool
		id, title, field, suggestion string
		score                        int
	}{
		{s.Environment.SoilCompaction, "environment.soil_compaction", "根区土壤板结", "environment.soil_compaction", "疏松土壤并改善透气透水条件", 12},
		{s.Environment.NearbyConstruction, "environment.nearby_construction", "邻近施工作业影响", "environment.nearby_construction", "划定根系保护区并落实工程监护", 18},
		{s.Environment.HeavyTraffic, "environment.heavy_traffic", "周边人车流量较大", "environment.heavy_traffic", "设置防护围栏和安全警示", 8},
		{s.Environment.DrainageProblem, "environment.drainage_problem", "根区排水异常", "environment.drainage_problem", "疏通排水并监测根区含水量", 14},
	}
	for _, rule := range environmentRules {
		if !rule.hit {
			continue
		}
		result.Factors = append(result.Factors, domain.RiskFactor{RuleID: rule.id, Title: rule.title, EvidenceField: rule.field, Evidence: s.Environment.Notes, Score: rule.score, Suggestion: rule.suggestion, Group: "environment", Level: factorLevel(rule.score)})
		result.Score += rule.score
		result.Suggestions = append(result.Suggestions, rule.suggestion)
	}
	result.Level = levelForScore(result.Score)
	if len(result.Factors) == 0 {
		result.Factors = append(result.Factors, domain.RiskFactor{RuleID: "baseline.good", Title: "未发现显著结构风险", EvidenceField: "survey", Evidence: "各结构观测为良好", Score: 0, Suggestion: "按常规周期巡查养护", Group: "baseline", Level: domain.RiskLow})
		result.Suggestions = append(result.Suggestions, "按常规周期巡查养护")
	}
	result.Groups = groupFactors(result.Factors)
	return result
}

func groupName(component string) string {
	switch component {
	case "树冠":
		return "crown"
	case "树干":
		return "trunk"
	case "根区":
		return "root_zone"
	default:
		return "environment"
	}
}

func factorLevel(score int) domain.RiskLevel {
	switch {
	case score >= 35:
		return domain.RiskCritical
	case score >= 20:
		return domain.RiskHigh
	case score >= 10:
		return domain.RiskMedium
	default:
		return domain.RiskLow
	}
}

func groupFactors(factors []domain.RiskFactor) []domain.RiskGroupResult {
	order := []string{"crown", "trunk", "root_zone", "environment", "baseline"}
	byGroup := map[string][]domain.RiskFactor{}
	for _, factor := range factors {
		byGroup[factor.Group] = append(byGroup[factor.Group], factor)
	}
	result := make([]domain.RiskGroupResult, 0, len(byGroup))
	for _, name := range order {
		if items := byGroup[name]; len(items) > 0 {
			group := domain.RiskGroupResult{Group: name, Factors: items}
			for _, item := range items {
				group.Score += item.Score
			}
			result = append(result, group)
		}
	}
	return result
}

func levelForScore(score int) domain.RiskLevel {
	switch {
	case score >= 60:
		return domain.RiskCritical
	case score >= 35:
		return domain.RiskHigh
	case score >= 15:
		return domain.RiskMedium
	default:
		return domain.RiskLow
	}
}

func BuildAssessment(result Result, manual domain.RiskLevel, reason, assessor string) (domain.RiskAssessment, error) {
	if strings.TrimSpace(assessor) == "" {
		return domain.RiskAssessment{}, &domain.ValidationError{Field: "assessor_name", Message: "评估人不能为空"}
	}
	final := result.Level
	if manual != "" {
		if !manual.Valid() {
			return domain.RiskAssessment{}, &domain.ValidationError{Field: "manual_level", Message: "人工风险等级无效"}
		}
		if manual != result.Level && strings.TrimSpace(reason) == "" {
			return domain.RiskAssessment{}, &domain.ValidationError{Field: "manual_reason", Message: "人工调整风险等级必须说明原因"}
		}
		ranks := map[domain.RiskLevel]int{domain.RiskLow: 1, domain.RiskMedium: 2, domain.RiskHigh: 3, domain.RiskCritical: 4}
		if ranks[manual] < ranks[result.Level]-1 {
			return domain.RiskAssessment{}, &domain.ValidationError{Field: "manual_level", Message: "人工降低风险不得跨越两个等级"}
		}
		if result.Level == domain.RiskCritical && manual != domain.RiskCritical && manual != domain.RiskHigh {
			return domain.RiskAssessment{}, &domain.ValidationError{Field: "manual_level", Message: "自动重大风险只能维持重大风险或降低为高风险"}
		}
		final = manual
	}
	return domain.RiskAssessment{AutomaticScore: result.Score, AutomaticLevel: result.Level, FinalLevel: final, Factors: append([]domain.RiskFactor(nil), result.Factors...), Groups: append([]domain.RiskGroupResult(nil), result.Groups...), ManualLevel: manual, ManualReason: strings.TrimSpace(reason), Assessor: strings.TrimSpace(assessor)}, nil
}
