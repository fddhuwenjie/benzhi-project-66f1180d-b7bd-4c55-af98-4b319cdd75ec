package failed_commit_case_state

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestFailedCommitDoesNotPublishInMemoryCase(t *testing.T) {
	ctx := context.Background()
	root := t.TempDir()
	repo, err := store.Open(root)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	base, err := domain.NewCareCase(domain.NewCaseInput{
		ID: "case-state", TreeCode: "GT-STATE", Species: "银杏", Location: "北门",
		OwnerName: "负责人", DueDate: "2026-12-31", Actor: "负责人", RequestID: "create-state", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Commit(ctx, base, 0, "create-state"); err != nil {
		t.Fatal(err)
	}
	current, err := repo.Get(ctx, base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := current.ReviseProfile(domain.CaseProfileInput{Species: "银杏", Location: "南门", OwnerName: "负责人", DueDate: "2026-12-31"}, "负责人", "revise-state", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	snapshot := filepath.Join(root, "snapshots", base.ID+".json")
	if err := os.Remove(snapshot); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshot, 0o750); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Commit(ctx, current, 1, "revise-state"); err == nil {
		t.Fatal("expected snapshot failure")
	}
	after, err := repo.Get(ctx, base.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Revision != 1 || after.Location != "北门" {
		t.Fatalf("failed commit leaked into in-memory index: revision=%d location=%q", after.Revision, after.Location)
	}
}
