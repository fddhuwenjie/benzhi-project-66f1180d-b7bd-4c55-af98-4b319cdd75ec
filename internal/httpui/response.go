package httpui

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

func (h *Handler) decode(w http.ResponseWriter, r *http.Request, target any) bool {
	if contentType := r.Header.Get("Content-Type"); !strings.HasPrefix(contentType, "application/json") {
		h.writeProblem(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Content-Type 必须为 application/json", "")
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		h.writeProblem(w, http.StatusBadRequest, "invalid_json", "请求 JSON 无效或超过 1 MiB", "")
		return false
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		h.writeProblem(w, http.StatusBadRequest, "invalid_json", "请求体只能包含一个 JSON 对象", "")
		return false
	} else if !errors.Is(err, io.EOF) {
		h.writeProblem(w, http.StatusBadRequest, "invalid_json", "JSON 对象后存在无效内容", "")
		return false
	}
	return true
}

func (h *Handler) writeError(w http.ResponseWriter, r *http.Request, err error) {
	var validation *domain.ValidationError
	var conflict *domain.ConflictError
	var notFound *domain.NotFoundError
	var state *domain.StateError
	var duplicate *domain.DuplicateCaseError
	switch {
	case errors.As(err, &duplicate):
		h.writeJSON(w, http.StatusConflict, map[string]any{"error": map[string]string{"code": "duplicate_active_case", "message": duplicate.Error(), "field": "tree_code", "existing_case_id": duplicate.CaseID, "existing_status": string(duplicate.Status)}})
	case errors.As(err, &validation):
		h.writeProblem(w, http.StatusBadRequest, "validation_error", validation.Message, validation.Field)
	case errors.As(err, &conflict):
		h.writeProblem(w, http.StatusConflict, "revision_conflict", conflict.Error(), "revision")
	case errors.As(err, &notFound):
		h.writeProblem(w, http.StatusNotFound, "not_found", notFound.Error(), "case_id")
	case errors.As(err, &state):
		h.writeProblem(w, http.StatusConflict, "invalid_state", state.Error(), "status")
	default:
		h.logger.Error("处理请求失败", "method", r.Method, "path", r.URL.Path, "error", err)
		h.writeProblem(w, http.StatusInternalServerError, "internal_error", "服务暂时无法处理请求", "")
	}
}

func (h *Handler) writeProblem(w http.ResponseWriter, status int, code, message, field string) {
	h.writeJSON(w, status, map[string]any{"error": map[string]string{"code": code, "message": message, "field": field}})
}

func (h *Handler) writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
