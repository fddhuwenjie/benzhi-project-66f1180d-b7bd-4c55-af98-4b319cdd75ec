package canceledwritecontext_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestCanceledRepositoryOperationsStopBeforeAccess(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	c, err := domain.NewCareCase(domain.NewCaseInput{
		ID: "case-canceled", TreeCode: "GT-CANCEL", Species: "银杏", Location: "北门",
		OwnerName: "负责人", DueDate: "2026-12-31", Actor: "负责人", RequestID: "cancel-create",
		Now: time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err = repo.Commit(ctx, c, 0, "cancel-create")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled commit returned %v, want context.Canceled", err)
	}
	if _, getErr := repo.Get(context.Background(), c.ID); getErr == nil {
		t.Fatal("canceled commit persisted a case")
	}
	if _, _, err := repo.Commit(context.Background(), c, 0, "cancel-create"); err != nil {
		t.Fatal(err)
	}

	if _, err := repo.Get(ctx, c.ID); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled get returned %v, want context.Canceled", err)
	}
	if _, err := repo.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled list returned %v, want context.Canceled", err)
	}
	if _, _, err := repo.LookupRequest(ctx, "cancel-create"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled request lookup returned %v, want context.Canceled", err)
	}
	if _, _, err := repo.LookupActiveTreeCode(ctx, "GT-CANCEL"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled tree lookup returned %v, want context.Canceled", err)
	}
}
