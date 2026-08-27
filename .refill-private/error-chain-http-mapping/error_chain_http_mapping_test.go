package error_chain_http_mapping

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/httpui"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func TestDomainValidationErrorSurvivesHTTPMapping(t *testing.T) {
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repo, risk.NewEngine(), nil, nil)
	server := httptest.NewServer(httpui.NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))))
	defer server.Close()

	create := `{"request_id":"error-chain-create","actor_name":"负责人","tree_code":"GT-ERR-1","species":"银杏","location":"公园","owner_name":"负责人","due_date":"2026-12-31"}`
	response, err := http.Post(server.URL+"/api/cases", "application/json", strings.NewReader(create))
	if err != nil {
		t.Fatal(err)
	}
	var created struct {
		ID string `json:"id"`
	}
	if response.StatusCode != http.StatusCreated || json.NewDecoder(response.Body).Decode(&created) != nil {
		response.Body.Close()
		t.Fatalf("create status=%d", response.StatusCode)
	}
	response.Body.Close()

	// 领域层会以 *domain.ValidationError 拒绝这份不完整现状；HTTP 适配器应保留该类型并返回客户端错误。
	invalidSurvey := `{"request_id":"error-chain-survey","revision":1,"actor_name":"调查员","crown_condition":{"condition":"","notes":""}}`
	response, err = http.Post(server.URL+"/api/cases/"+created.ID+"/survey", "application/json", strings.NewReader(invalidSurvey))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("validation error was not mapped through the service error chain: status=%d body=%s", response.StatusCode, body)
	}
}
