package application_test

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

type sequenceIDs struct{ next int }

func (s *sequenceIDs) NewID(prefix string) string {
	s.next++
	return prefix + "-test-" + string(rune('a'+s.next))
}

func TestServiceIdempotencyAndOptimisticConflict(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	clock := fixedClock{now: time.Date(2026, 8, 25, 8, 0, 0, 0, time.UTC)}
	service := application.NewService(repo, risk.NewEngine(), clock, &sequenceIDs{})
	ctx := context.Background()
	c, err := service.CreateCase(ctx, application.CreateCaseCommand{RequestID: "create", Actor: "负责人", TreeCode: "GT-9", Species: "银杏", Location: "公园", OwnerName: "负责人", DueDate: "2026-12-31"})
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.SubmitSurveyCommand{CommandMeta: application.CommandMeta{RequestID: "survey", ExpectedRevision: c.Revision, Actor: "调查员"}, CaseID: c.ID, Crown: domain.Observation{Condition: domain.ConditionGood, Notes: "树冠正常"}, Trunk: domain.Observation{Condition: domain.ConditionGood, Notes: "树干正常"}, RootZone: domain.Observation{Condition: domain.ConditionAttention, Notes: "轻微踩踏"}, Environment: domain.EnvironmentObservation{Notes: "环境稳定"}, ObservedAt: clock.now, Observer: "调查员"}
	first, err := service.SubmitSurvey(ctx, cmd)
	if err != nil {
		t.Fatal(err)
	}
	duplicate, err := service.SubmitSurvey(ctx, cmd)
	if err != nil || duplicate.Revision != first.Revision {
		t.Fatalf("duplicate revision=%d want=%d err=%v", duplicate.Revision, first.Revision, err)
	}
	cmd.RequestID = "stale-new-request"
	if _, err := service.SubmitSurvey(ctx, cmd); err == nil {
		t.Fatal("expected optimistic conflict")
	} else {
		var conflict *domain.ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("got %T", err)
		}
	}
}

func TestDashboardBuildsTodoAndLabels(t *testing.T) {
	repo, _ := store.Open(t.TempDir())
	service := application.NewService(repo, risk.NewEngine(), fixedClock{time.Now().UTC()}, &sequenceIDs{})
	created, err := service.CreateCase(context.Background(), application.CreateCaseCommand{RequestID: "dashboard-create", Actor: "甲", TreeCode: "GT-10", Species: "香樟", Location: "广场", OwnerName: "甲", DueDate: "2026-12-31"})
	if err != nil {
		t.Fatal(err)
	}
	dashboard, err := service.Dashboard(context.Background())
	if err != nil || len(dashboard.Cases) != 1 {
		t.Fatalf("dashboard=%+v err=%v", dashboard, err)
	}
	if dashboard.Cases[0].ID != created.ID || dashboard.Cases[0].StatusLabel != "草稿" || dashboard.Cases[0].NextAction == "" {
		t.Fatalf("summary=%+v", dashboard.Cases[0])
	}
}
