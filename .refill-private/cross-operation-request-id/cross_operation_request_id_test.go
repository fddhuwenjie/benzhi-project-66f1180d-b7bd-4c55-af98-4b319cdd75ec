package crossoperationrequestid_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type fixedIDs struct{}

func (fixedIDs) NewID(prefix string) string { return prefix + "-request-reuse" }

func TestRequestIDCannotCrossOperationBoundary(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repo, risk.NewEngine(), fixedClock{time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)}, fixedIDs{})
	ctx := context.Background()
	created, err := service.CreateCase(ctx, application.CreateCaseCommand{RequestID: "shared-request", Actor: "甲", TreeCode: "GT-REUSE", Species: "银杏", Location: "北门", OwnerName: "甲", DueDate: "2026-12-31"})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ReviseCase(ctx, application.ReviseCaseCommand{
		CommandMeta: application.CommandMeta{RequestID: "shared-request", ExpectedRevision: created.Revision, Actor: "甲"},
		CaseID:      created.ID, Species: created.Species, Location: "南门", OwnerName: created.OwnerName, DueDate: created.DueDate,
	})
	var validation *domain.ValidationError
	if !errors.As(err, &validation) || validation.Field != "request_id" {
		t.Fatalf("cross-operation request_id reuse returned %v, want request_id validation error", err)
	}
	detail, getErr := service.GetCase(ctx, created.ID)
	if getErr != nil {
		t.Fatal(getErr)
	}
	if detail.Case.Location != "北门" {
		t.Fatalf("cross-operation cache hit mutated location to %q", detail.Case.Location)
	}
}
