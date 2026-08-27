package repository_nested_state_alias

import (
	"context"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestRepositoryGetDoesNotExposeNestedState(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	caseItem, err := domain.NewCareCase(domain.NewCaseInput{
		ID: "case-alias", TreeCode: "GT-ALIAS", Species: "银杏", Location: "广场",
		OwnerName: "负责人", DueDate: "2026-12-31", Actor: "负责人", RequestID: "create-alias", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	caseItem.Survey = &domain.ConditionSurvey{
		ID: "survey-alias", CaseID: caseItem.ID, Observer: "现场员", ObservedAt: now,
		PhotoRefs: []domain.PhotoRef{{Name: "树冠.jpg", URL: "photos/original.jpg"}},
	}
	if _, _, err := repo.Commit(context.Background(), caseItem, 0, "create-alias"); err != nil {
		t.Fatal(err)
	}

	loaded, err := repo.Get(context.Background(), caseItem.ID)
	if err != nil {
		t.Fatal(err)
	}
	loaded.Survey.PhotoRefs[0].URL = "photos/changed-without-commit.jpg"

	again, err := repo.Get(context.Background(), caseItem.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got := again.Survey.PhotoRefs[0].URL; got != "photos/original.jpg" {
		t.Fatalf("未提交的返回值修改污染了仓储状态: got %q", got)
	}
}
