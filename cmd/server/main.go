package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/httpui"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/store"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cfg, err := parseConfig(args, os.Getenv("PORT"))
	if err != nil {
		return err
	}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if cfg.selfcheck {
		return runSelfcheck(cfg, logger)
	}
	repo, err := store.Open(cfg.dataDir)
	if err != nil {
		return fmt.Errorf("打开数据存储: %w", err)
	}
	handler := buildHandler(repo, logger)
	server := newHTTPServer(cfg.addr, handler)
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	logger.Info("古树养护闭环核验台已启动", "addr", listener.Addr().String(), "data", cfg.dataDir)
	errCh := make(chan error, 1)
	go func() { errCh <- server.Serve(listener) }()
	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case serveErr := <-errCh:
		if !errors.Is(serveErr, http.ErrServerClosed) {
			return serveErr
		}
		return nil
	case <-signalCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		logger.Info("正在关闭 HTTP 服务")
		return server.Shutdown(shutdownCtx)
	}
}

func buildHandler(repo application.CaseRepository, logger *slog.Logger) http.Handler {
	service := application.NewService(repo, risk.NewEngine(), application.SystemClock{}, application.RandomIDs{})
	return httpui.NewHandler(service, logger)
}

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr: addr, Handler: handler, ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second,
		WriteTimeout: 20 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 32 << 10,
	}
}

func runSelfcheck(cfg config, logger *slog.Logger) error {
	tempDir, err := os.MkdirTemp("", "benzhi-tree-selfcheck-*")
	if err != nil {
		return fmt.Errorf("创建自检目录: %w", err)
	}
	defer os.RemoveAll(tempDir)
	repo, err := store.Open(tempDir)
	if err != nil {
		return err
	}
	server := newHTTPServer(cfg.addr, buildHandler(repo, logger))
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("自检监听 %s: %w", cfg.addr, err)
	}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()
	baseURL := "http://" + listener.Addr().String()
	checkCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 4 * time.Second}
	if err := exerciseClosedLoop(checkCtx, client, baseURL); err != nil {
		_ = server.Close()
		return err
	}
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("关闭自检服务: %w", err)
	}
	select {
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("自检 HTTP 服务: %w", err)
		}
	case <-time.After(time.Second):
		return errors.New("自检 HTTP 服务未按时退出")
	}
	fmt.Println("selfcheck passed: health, workbench and full care case lifecycle")
	return nil
}

func exerciseClosedLoop(ctx context.Context, client *http.Client, baseURL string) error {
	var health map[string]string
	if err := getJSON(ctx, client, baseURL+"/healthz", &health); err != nil {
		return fmt.Errorf("健康检查: %w", err)
	}
	if health["status"] != "ok" {
		return errors.New("健康检查未返回 ok")
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/", nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("访问工作台: %w", err)
	}
	page, readErr := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusOK || !bytes.Contains(page, []byte("古树养护闭环核验台")) {
		return errors.New("工作台页面不可用")
	}
	var current caseResponse
	if err := postJSON(ctx, client, baseURL+"/api/cases", map[string]any{"request_id": "self-create", "actor_name": "自检负责人", "tree_code": "SELF-001", "species": "银杏", "location": "自检点位", "owner_name": "自检负责人", "due_date": "2026-12-31"}, &current, http.StatusCreated); err != nil {
		return fmt.Errorf("创建任务: %w", err)
	}
	id := current.ID
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/survey", map[string]any{"request_id": "self-survey", "revision": current.Revision, "actor_name": "自检调查员", "crown_condition": map[string]any{"condition": "poor", "notes": "树冠枯枝较多"}, "trunk_condition": map[string]any{"condition": "attention", "notes": "树干轻微损伤"}, "root_zone_condition": map[string]any{"condition": "poor", "notes": "根区土壤板结"}, "environment": map[string]any{"notes": "邻近道路施工", "soil_compaction": true, "nearby_construction": true}, "observed_at": "2026-08-25T08:00:00Z", "observer_name": "自检调查员", "photo_refs": []map[string]any{{"name": "树冠.jpg", "url": "photos/self/crown.jpg", "part": "crown", "taken_at": "2026-08-25T07:30:00Z", "caption": "枯枝位置"}, {"name": "根区.jpg", "url": "photos/self/root.jpg", "part": "root_zone", "taken_at": "2026-08-25T07:35:00Z", "caption": "土壤板结"}, {"name": "环境.jpg", "url": "photos/self/environment.jpg", "part": "environment", "taken_at": "2026-08-25T07:40:00Z", "caption": "邻近施工"}}}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("提交现状: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/assessment", map[string]any{"request_id": "self-risk", "revision": current.Revision, "actor_name": "自检评估员", "assessor_name": "自检评估员", "manual_reason": ""}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("风险评估: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/plans", map[string]any{"request_id": "self-plan", "revision": current.Revision, "actor_name": "自检编制人", "prepared_by": "自检编制人", "measures": []map[string]any{{"title": "结构修剪与根区改良", "description": "清理枯枝、保护树干并实施土壤改良", "risk_rule_ids": []string{"condition.树冠.poor", "condition.树干.attention", "condition.根区.poor", "environment.soil_compaction", "environment.nearby_construction"}, "control_points": []string{"作业区完成围挡", "高空作业双人监护"}, "completion_standard": "枯枝清理且根区透气条件改善"}}, "materials": []string{"伤口保护材料", "有机覆盖物"}, "work_window": "2026-09-01 至 2026-09-03", "safety_controls": []string{"作业区完成围挡", "高空作业双人监护"}, "completion_criteria": []string{"枯枝清理且根区透气条件改善"}}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("保存方案: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/plans/submit", map[string]any{"request_id": "self-submit-plan", "revision": current.Revision, "actor_name": "自检编制人"}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("提交审核: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/plans/review", map[string]any{"request_id": "self-review", "revision": current.Revision, "actor_name": "自检审核人", "approved": true, "reviewer": "自检审核人", "opinions": []map[string]any{{"item": "风险措施", "result": "passed", "opinion": "风险措施完整"}, {"item": "材料适用性", "result": "passed", "opinion": "材料适用"}, {"item": "作业窗口", "result": "passed", "opinion": "窗口合理"}, {"item": "安全控制", "result": "passed", "opinion": "控制点完整"}, {"item": "完成标准", "result": "passed", "opinion": "标准可核验"}}}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("批准方案: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/executions", map[string]any{"request_id": "self-execution", "revision": current.Revision, "actor_name": "自检现场人员", "performed_at": "2026-09-02T09:00:00Z", "crew_names": []string{"作业员甲", "作业员乙"}, "actual_measures": []string{"结构修剪与根区改良"}, "control_checks": []map[string]any{{"control": "作业区完成围挡", "passed": true}, {"control": "高空作业双人监护", "passed": true}}, "evidence_refs": []map[string]any{{"name": "完工全景.jpg", "url": "evidence/self/done.jpg"}}, "submitted_by": "自检现场人员"}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("提交实施: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/executions/complete", map[string]any{"request_id": "self-complete", "revision": current.Revision, "actor_name": "自检负责人"}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("提交完工汇总: %w", err)
	}
	if err := postJSON(ctx, client, baseURL+"/api/cases/"+id+"/acceptance", map[string]any{"request_id": "self-accept", "revision": current.Revision, "actor_name": "自检负责人", "passed": true, "inspector": "自检负责人", "inspected_at": "2026-09-03T10:00:00Z", "criterion_results": []string{"现场核验完成，抽查合格"}, "nonconformities": []map[string]any{}, "notes": "自检闭环验收通过"}, &current, http.StatusOK); err != nil {
		return fmt.Errorf("验收关闭: %w", err)
	}
	var detail struct {
		Case     caseResponse `json:"case"`
		Timeline []any        `json:"timeline"`
	}
	if err := getJSON(ctx, client, baseURL+"/api/cases/"+id, &detail); err != nil {
		return fmt.Errorf("读取最终任务: %w", err)
	}
	if detail.Case.Status != "closed" || len(detail.Timeline) != 9 {
		return fmt.Errorf("最终任务未闭环：status=%s timeline=%d", detail.Case.Status, len(detail.Timeline))
	}
	return nil
}

type caseResponse struct {
	ID       string `json:"id"`
	Revision int64  `json:"revision"`
	Status   string `json:"status"`
}

func getJSON(ctx context.Context, client *http.Client, url string, target any) error {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, body)
	}
	return json.NewDecoder(response.Body).Decode(target)
}

func postJSON(ctx context.Context, client *http.Client, url string, payload, target any, wantStatus int) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != wantStatus {
		body, _ := io.ReadAll(io.LimitReader(response.Body, 4096))
		return fmt.Errorf("HTTP %d: %s", response.StatusCode, body)
	}
	return json.NewDecoder(response.Body).Decode(target)
}
