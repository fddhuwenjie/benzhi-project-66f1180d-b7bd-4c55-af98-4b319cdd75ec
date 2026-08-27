package dashboard_stats_alias

import (
	"context"
	"testing"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestDashboardResponseDoesNotShareStatisticsState(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repo, risk.NewEngine(), nil, nil)
	ctx := context.Background()
	if _, err := service.CreateCase(ctx, application.CreateCaseCommand{
		RequestID: "dashboard-alias-create", Actor: "负责人", TreeCode: "GT-ALIAS-1",
		Species: "银杏", Location: "公园", OwnerName: "负责人", DueDate: "2026-12-31",
	}); err != nil {
		t.Fatal(err)
	}
	all, err := service.Dashboard(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if all.Counts["all"] != 1 || all.Statistics.Statuses["draft"] != 1 {
		t.Fatalf("initial dashboard=%+v", all)
	}
	if _, err := service.QueryDashboard(ctx, application.CaseQuery{Status: "closed"}); err != nil {
		t.Fatal(err)
	}
	if all.Counts["all"] != 1 || all.Statistics.Statuses["draft"] != 1 {
		t.Fatalf("prior dashboard mutated after later query: %+v", all)
	}
}
