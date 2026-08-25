package httpui

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func testServer(t *testing.T) *httptest.Server {
	t.Helper()
	repo, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	service := application.NewService(repo, risk.NewEngine(), nil, nil)
	return httptest.NewServer(NewHandler(service, slog.New(slog.NewTextHandler(io.Discard, nil))))
}

func TestWorkbenchAndCreateCase(t *testing.T) {
	server := testServer(t)
	defer server.Close()
	response, err := http.Get(server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	page, _ := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if response.StatusCode != http.StatusOK || !bytes.Contains(page, []byte("<body>")) {
		t.Fatalf("status=%d", response.StatusCode)
	}
	payload := `{"request_id":"http-create","actor_name":"负责人","tree_code":"GT-H1","species":"银杏","location":"公园","owner_name":"负责人","due_date":"2026-12-31"}`
	response, err = http.Post(server.URL+"/api/cases", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(response.Body)
		t.Fatalf("status=%d body=%s", response.StatusCode, body)
	}
	var created map[string]any
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil || created["status"] != "draft" {
		t.Fatalf("created=%v err=%v", created, err)
	}
}

func TestJSONAndMethodBoundaries(t *testing.T) {
	server := testServer(t)
	defer server.Close()
	response, err := http.Post(server.URL+"/api/cases", "text/plain", strings.NewReader("{}"))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("content type status=%d", response.StatusCode)
	}
	response, err = http.Post(server.URL+"/api/cases", "application/json", strings.NewReader("{} trailing"))
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusBadRequest {
		t.Fatalf("trailing JSON status=%d", response.StatusCode)
	}
	request, _ := http.NewRequest(http.MethodDelete, server.URL+"/api/cases", nil)
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	_ = response.Body.Close()
	if response.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("method status=%d", response.StatusCode)
	}
}
