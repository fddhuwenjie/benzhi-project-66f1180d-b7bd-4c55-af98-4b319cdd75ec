package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func TestCommitPersistsRestoresAndDeduplicates(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 0, 0, 0, 0, time.UTC)
	c, err := domain.NewCareCase(domain.NewCaseInput{ID: "case-store", TreeCode: "GT-1", Species: "银杏", Location: "公园", OwnerName: "甲", DueDate: "2026-12-31", Actor: "甲", RequestID: "create", Now: now})
	if err != nil {
		t.Fatal(err)
	}
	saved, duplicate, err := repo.Commit(context.Background(), c, 0, "create")
	if err != nil || duplicate {
		t.Fatalf("commit duplicate=%v err=%v", duplicate, err)
	}
	again, duplicate, err := repo.Commit(context.Background(), c, 0, "create")
	if err != nil || !duplicate || again.ID != saved.ID {
		t.Fatalf("duplicate result=%+v duplicate=%v err=%v", again, duplicate, err)
	}
	reopened, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := reopened.Get(context.Background(), c.ID)
	if err != nil || loaded.Revision != 1 {
		t.Fatalf("loaded=%+v err=%v", loaded, err)
	}
	if _, ok, err := reopened.LookupRequest(context.Background(), "create"); err != nil || !ok {
		t.Fatalf("restored idempotency ok=%v err=%v", ok, err)
	}
}

func TestCommitDetectsRevisionConflict(t *testing.T) {
	repo, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	c, _ := domain.NewCareCase(domain.NewCaseInput{ID: "case-conflict", TreeCode: "GT-2", Species: "香樟", Location: "街角", OwnerName: "乙", DueDate: "2026-12-31", Actor: "乙", RequestID: "one", Now: now})
	if _, _, err := repo.Commit(context.Background(), c, 0, "one"); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Commit(context.Background(), c, 99, "two"); err == nil {
		t.Fatal("expected conflict")
	} else {
		var conflict *domain.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("got %T", err)
		}
	}
}

func TestOpenRepairsMalformedAuditFromSnapshot(t *testing.T) {
	dir := t.TempDir()
	repo, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	c, _ := domain.NewCareCase(domain.NewCaseInput{ID: "case-repair", TreeCode: "GT-3", Species: "榕树", Location: "广场", OwnerName: "丙", DueDate: "2026-12-31", Actor: "丙", RequestID: "repair", Now: time.Now().UTC()})
	if _, _, err := repo.Commit(context.Background(), c, 0, "repair"); err != nil {
		t.Fatal(err)
	}
	f, err := os.OpenFile(repo.auditPath(c.ID), os.O_APPEND|os.O_WRONLY, 0o640)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("{broken\n"); err != nil {
		t.Fatal(err)
	}
	_ = f.Close()
	if _, err := Open(dir); err != nil {
		t.Fatalf("repair failed: %v", err)
	}
	events, err := readAudit(repo.auditPath(c.ID))
	if err != nil || len(events) != 1 {
		t.Fatalf("events=%d err=%v", len(events), err)
	}
}
