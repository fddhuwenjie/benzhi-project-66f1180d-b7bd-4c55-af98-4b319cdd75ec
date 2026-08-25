package httpui

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/application"
)

//go:embed assets/*
var assets embed.FS

type Handler struct {
	service *application.Service
	logger  *slog.Logger
	mux     *http.ServeMux
}

func NewHandler(service *application.Service, logger *slog.Logger) *Handler {
	if logger == nil {
		logger = slog.Default()
	}
	h := &Handler{service: service, logger: logger, mux: http.NewServeMux()}
	h.routes()
	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "same-origin")
	w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'; img-src 'self' data: https:; connect-src 'self'")
	h.mux.ServeHTTP(w, r)
}

func (h *Handler) routes() {
	staticFS, _ := fs.Sub(assets, "assets")
	h.mux.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))
	h.mux.HandleFunc("GET /", h.Workbench)
	h.mux.HandleFunc("GET /healthz", h.Health)
	h.mux.HandleFunc("GET /api/cases", h.ListCases)
	h.mux.HandleFunc("POST /api/cases", h.CreateCase)
	h.mux.HandleFunc("GET /api/cases/{id}", h.GetCase)
	h.mux.HandleFunc("PATCH /api/cases/{id}", h.ReviseCase)
	h.mux.HandleFunc("POST /api/cases/{id}/survey", h.SubmitSurvey)
	h.mux.HandleFunc("POST /api/cases/{id}/assessment", h.AssessRisk)
	h.mux.HandleFunc("GET /api/cases/{id}/assessment", h.RiskPreview)
	h.mux.HandleFunc("POST /api/cases/{id}/plans", h.SavePlan)
	h.mux.HandleFunc("POST /api/cases/{id}/plans/submit", h.SubmitPlan)
	h.mux.HandleFunc("POST /api/cases/{id}/plans/review", h.ReviewPlan)
	h.mux.HandleFunc("POST /api/cases/{id}/executions", h.RecordExecution)
	h.mux.HandleFunc("POST /api/cases/{id}/executions/complete", h.CompleteExecution)
	h.mux.HandleFunc("POST /api/cases/{id}/acceptance", h.AcceptCase)
}

func (h *Handler) Workbench(w http.ResponseWriter, r *http.Request) {
	data, err := assets.ReadFile("assets/index.html")
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}

func (h *Handler) Health(w http.ResponseWriter, _ *http.Request) {
	h.writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "古树养护闭环核验台"})
}

func (h *Handler) ListCases(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.QueryDashboard(r.Context(), application.CaseQuery{Keyword: r.URL.Query().Get("keyword"), OwnerName: r.URL.Query().Get("owner_name"), Status: r.URL.Query().Get("status"), RiskLevel: r.URL.Query().Get("risk_level"), DeadlineLevel: r.URL.Query().Get("deadline_level")})
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ReviseCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviseCaseCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.ReviseCase(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) GetCase(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.GetCase(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CreateCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.CreateCaseCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	result, err := h.service.CreateCase(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) SubmitSurvey(w http.ResponseWriter, r *http.Request) {
	var cmd application.SubmitSurveyCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.SubmitSurvey(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AssessRisk(w http.ResponseWriter, r *http.Request) {
	var cmd application.AssessRiskCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.AssessRisk(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RiskPreview(w http.ResponseWriter, r *http.Request) {
	result, err := h.service.RiskPreview(r.Context(), r.PathValue("id"))
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SavePlan(w http.ResponseWriter, r *http.Request) {
	var cmd application.SavePlanCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.SavePlan(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) SubmitPlan(w http.ResponseWriter, r *http.Request) {
	var cmd application.SubmitPlanCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.SubmitPlan(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) ReviewPlan(w http.ResponseWriter, r *http.Request) {
	var cmd application.ReviewPlanCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.ReviewPlan(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) RecordExecution(w http.ResponseWriter, r *http.Request) {
	var cmd application.RecordExecutionCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.RecordExecution(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) CompleteExecution(w http.ResponseWriter, r *http.Request) {
	var cmd application.CompleteExecutionCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.CompleteExecution(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}

func (h *Handler) AcceptCase(w http.ResponseWriter, r *http.Request) {
	var cmd application.AcceptCommand
	if !h.decode(w, r, &cmd) {
		return
	}
	cmd.CaseID = r.PathValue("id")
	result, err := h.service.Accept(r.Context(), cmd)
	if err != nil {
		h.writeError(w, r, err)
		return
	}
	h.writeJSON(w, http.StatusOK, result)
}
