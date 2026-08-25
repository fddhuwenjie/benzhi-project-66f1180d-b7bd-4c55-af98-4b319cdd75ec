package store

import (
	"encoding/json"
	"fmt"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func cloneCase(c *domain.CareCase) (*domain.CareCase, error) {
	data, err := json.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("复制任务: %w", err)
	}
	var result domain.CareCase
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("复制任务: %w", err)
	}
	return &result, nil
}
