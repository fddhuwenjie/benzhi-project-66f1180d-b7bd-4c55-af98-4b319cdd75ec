package store_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestFailedCommitDoesNotPoisonRequestIndex(t *testing.T) {
	dir := t.TempDir()
	repo, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)
	c, err := domain.NewCareCase(domain.NewCaseInput{
		ID: "case-retry", TreeCode: "GT-RETRY", Species: "银杏", Location: "公园",
		OwnerName: "负责人", DueDate: "2026-12-31", Actor: "负责人", RequestID: "retry-me", Now: now,
	})
	if err != nil {
		t.Fatal(err)
	}

	snapshotDir := filepath.Join(dir, "snapshots")
	if err := os.Remove(snapshotDir); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(snapshotDir, []byte("snapshot storage unavailable"), 0o640); err != nil {
		t.Fatal(err)
	}
	if _, _, err := repo.Commit(context.Background(), c, 0, "retry-me"); err == nil {
		t.Fatal("expected snapshot write failure")
	}
	if err := os.Remove(snapshotDir); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(snapshotDir, 0o750); err != nil {
		t.Fatal(err)
	}

	saved, duplicate, err := repo.Commit(context.Background(), c, 0, "retry-me")
	if err != nil || duplicate || saved == nil || saved.ID != c.ID {
		t.Fatalf("retry saved=%+v duplicate=%v err=%v", saved, duplicate, err)
	}
}
