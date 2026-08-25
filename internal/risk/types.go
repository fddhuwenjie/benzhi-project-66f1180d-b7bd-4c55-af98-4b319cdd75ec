package risk

import "benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"

type Result struct {
	Score       int                      `json:"score"`
	Level       domain.RiskLevel         `json:"level"`
	Factors     []domain.RiskFactor      `json:"factors"`
	Suggestions []string                 `json:"suggestions"`
	Groups      []domain.RiskGroupResult `json:"groups"`
}

func NewEngine() *Engine { return &Engine{} }
