package store

import "benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"

func cloneCase(c *domain.CareCase) (*domain.CareCase, error) {
	result := *c
	return &result, nil
}
