package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/risk"
)

type Service struct {
	repo  CaseRepository
	risk  *risk.Engine
	clock Clock
	ids   IDGenerator
}

func NewService(repo CaseRepository, riskEngine *risk.Engine, clock Clock, ids IDGenerator) *Service {
	if clock == nil {
		clock = SystemClock{}
	}
	if ids == nil {
		ids = RandomIDs{}
	}
	return &Service{repo: repo, risk: riskEngine, clock: clock, ids: ids}
}

func validateMeta(m CommandMeta) error {
	if strings.TrimSpace(m.RequestID) == "" {
		return &domain.ValidationError{Field: "request_id", Message: "request_id 不能为空"}
	}
	if strings.TrimSpace(m.Actor) == "" {
		return &domain.ValidationError{Field: "actor_name", Message: "操作人不能为空"}
	}
	if m.ExpectedRevision < 1 {
		return &domain.ValidationError{Field: "revision", Message: "revision 必须为正整数"}
	}
	return nil
}

func (s *Service) CreateCase(ctx context.Context, cmd CreateCaseCommand) (*domain.CareCase, error) {
	if strings.TrimSpace(cmd.RequestID) == "" {
		return nil, &domain.ValidationError{Field: "request_id", Message: "request_id 不能为空"}
	}
	if strings.TrimSpace(cmd.Actor) == "" {
		return nil, &domain.ValidationError{Field: "actor_name", Message: "操作人不能为空"}
	}
	if prior, ok, err := s.repo.LookupRequest(ctx, cmd.RequestID); err != nil {
		return nil, err
	} else if ok {
		if err := verifyReplay(prior, cmd.RequestID, opCreateCase, fingerprintCreate(cmd)); err != nil {
			return nil, err
		}
		return prior, nil
	}
	if existing, ok, err := s.repo.LookupActiveTreeCode(ctx, strings.TrimSpace(cmd.TreeCode)); err != nil {
		return nil, err
	} else if ok {
		return nil, &domain.DuplicateCaseError{TreeCode: strings.TrimSpace(cmd.TreeCode), CaseID: existing.ID, Status: existing.Status}
	}
	now := s.clock.Now()
	c, err := domain.NewCareCase(domain.NewCaseInput{
		ID: s.ids.NewID("case"), TreeCode: cmd.TreeCode, Species: cmd.Species, Location: cmd.Location,
		OwnerName: cmd.OwnerName, DueDate: cmd.DueDate, Actor: cmd.Actor, RequestID: cmd.RequestID, Now: now,
	})
	if err != nil {
		return nil, err
	}
	recordRequest(c, cmd.RequestID, opCreateCase, fingerprintCreate(cmd))
	result, _, err := s.repo.Commit(ctx, c, 0, cmd.RequestID)
	return result, err
}

func (s *Service) ReviseCase(ctx context.Context, cmd ReviseCaseCommand) (*domain.CareCase, error) {
	return s.changeOptional(ctx, cmd.CaseID, cmd.CommandMeta, opReviseCase, fingerprintRevise(cmd), func(c *domain.CareCase, now time.Time) (bool, error) {
		return c.ReviseProfile(domain.CaseProfileInput{Species: cmd.Species, Location: cmd.Location, OwnerName: cmd.OwnerName, DueDate: cmd.DueDate}, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) SubmitSurvey(ctx context.Context, cmd SubmitSurveyCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opSubmitSurvey, fingerprintSurvey(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.SubmitSurveyAt(domain.ConditionSurvey{
			ID: s.ids.NewID("survey"), Crown: cmd.Crown, Trunk: cmd.Trunk, RootZone: cmd.RootZone,
			Environment: cmd.Environment, ObservedAt: cmd.ObservedAt, Observer: cmd.Observer, PhotoRefs: cmd.PhotoRefs,
		}, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) AssessRisk(ctx context.Context, cmd AssessRiskCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opAssessRisk, fingerprintAssess(cmd), func(c *domain.CareCase, now time.Time) error {
		if c.Survey == nil {
			return &domain.ValidationError{Field: "survey", Message: "必须先提交现状记录"}
		}
		result := s.risk.Evaluate(*c.Survey)
		assessment, err := risk.BuildAssessment(result, cmd.ManualLevel, cmd.ManualReason, cmd.Assessor)
		if err != nil {
			return err
		}
		assessment.AssessedAt = now
		assessment.ID = s.ids.NewID("assessment")
		return c.ApplyRiskReview(assessment, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) SavePlan(ctx context.Context, cmd SavePlanCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opSavePlan, fingerprintSavePlan(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.SavePlanForAssessment(domain.CarePlan{
			ID: s.ids.NewID("plan"), Measures: cmd.Measures, Materials: cmd.Materials,
			WorkWindow: cmd.WorkWindow, SafetyControls: cmd.SafetyControls,
			CompletionCriteria: cmd.CompletionCriteria, PreparedBy: cmd.PreparedBy,
			Exemptions: cmd.Exemptions, RemediationResponses: cmd.RemediationResponses,
		}, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) SubmitPlan(ctx context.Context, cmd SubmitPlanCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opSubmitPlan, fingerprintSubmitPlan(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.SubmitPlanCovered(cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) ReviewPlan(ctx context.Context, cmd ReviewPlanCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opReviewPlan, fingerprintReviewPlan(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.ReviewPlanChecklist(cmd.Approved, cmd.Opinions, cmd.Reviewer, cmd.RequestID, now)
	})
}

func (s *Service) RecordExecution(ctx context.Context, cmd RecordExecutionCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opRecordExecution, fingerprintRecordExecution(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.RegisterExecution(domain.ExecutionRecord{
			ID: s.ids.NewID("execution"), PerformedAt: cmd.PerformedAt, CrewNames: cmd.CrewNames,
			ActualMeasures: cmd.ActualMeasures, ControlChecks: cmd.ControlChecks,
			EvidenceRefs: cmd.EvidenceRefs, SubmittedBy: cmd.SubmittedBy, Remediations: cmd.Remediations,
		}, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) CompleteExecution(ctx context.Context, cmd CompleteExecutionCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opCompleteExecution, fingerprintCompleteExecution(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.CompleteExecution(cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) Accept(ctx context.Context, cmd AcceptCommand) (*domain.CareCase, error) {
	return s.change(ctx, cmd.CaseID, cmd.CommandMeta, opAccept, fingerprintAccept(cmd), func(c *domain.CareCase, now time.Time) error {
		return c.AcceptStructured(domain.AcceptanceRecord{
			Passed: cmd.Passed, Inspector: cmd.Inspector, InspectedAt: cmd.InspectedAt,
			CriterionResults: cmd.CriterionResults, Items: cmd.Nonconformities, Notes: cmd.Notes,
		}, cmd.Actor, cmd.RequestID, now)
	})
}

func (s *Service) RiskPreview(ctx context.Context, caseID string) (risk.Result, error) {
	c, err := s.repo.Get(ctx, caseID)
	if err != nil {
		return risk.Result{}, err
	}
	if c.Survey == nil {
		return risk.Result{}, &domain.ValidationError{Field: "survey", Message: "必须先提交现状记录"}
	}
	return s.risk.Evaluate(*c.Survey), nil
}

func (s *Service) change(ctx context.Context, caseID string, meta CommandMeta, operation, fingerprint string, mutate func(*domain.CareCase, time.Time) error) (*domain.CareCase, error) {
	if err := validateMeta(meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(caseID) == "" {
		return nil, &domain.ValidationError{Field: "case_id", Message: "任务 ID 不能为空"}
	}
	if prior, ok, err := s.repo.LookupRequest(ctx, meta.RequestID); err != nil {
		return nil, err
	} else if ok {
		if prior.ID != caseID {
			return nil, &domain.ValidationError{Field: "request_id", Message: "request_id 已用于其他任务"}
		}
		if err := verifyReplay(prior, meta.RequestID, operation, fingerprint); err != nil {
			return nil, err
		}
		return prior, nil
	}
	c, err := s.repo.Get(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if c.Revision != meta.ExpectedRevision {
		return nil, &domain.ConflictError{Expected: meta.ExpectedRevision, Actual: c.Revision}
	}
	if err := mutate(c, s.clock.Now()); err != nil {
		return nil, err
	}
	recordRequest(c, meta.RequestID, operation, fingerprint)
	result, _, err := s.repo.Commit(ctx, c, meta.ExpectedRevision, meta.RequestID)
	return result, err
}

func (s *Service) changeOptional(ctx context.Context, caseID string, meta CommandMeta, operation, fingerprint string, mutate func(*domain.CareCase, time.Time) (bool, error)) (*domain.CareCase, error) {
	if err := validateMeta(meta); err != nil {
		return nil, err
	}
	if strings.TrimSpace(caseID) == "" {
		return nil, &domain.ValidationError{Field: "case_id", Message: "任务 ID 不能为空"}
	}
	if prior, ok, err := s.repo.LookupRequest(ctx, meta.RequestID); err != nil {
		return nil, err
	} else if ok {
		if prior.ID != caseID {
			return nil, &domain.ValidationError{Field: "request_id", Message: "request_id 已用于其他任务"}
		}
		if err := verifyReplay(prior, meta.RequestID, operation, fingerprint); err != nil {
			return nil, err
		}
		return prior, nil
	}
	c, err := s.repo.Get(ctx, caseID)
	if err != nil {
		return nil, err
	}
	if c.Revision != meta.ExpectedRevision {
		return nil, &domain.ConflictError{Expected: meta.ExpectedRevision, Actual: c.Revision}
	}
	changed, err := mutate(c, s.clock.Now())
	if err != nil {
		return nil, err
	}
	if !changed {
		return c, nil
	}
	recordRequest(c, meta.RequestID, operation, fingerprint)
	result, _, err := s.repo.Commit(ctx, c, meta.ExpectedRevision, meta.RequestID)
	return result, err
}

func (s *Service) GetCase(ctx context.Context, id string) (CaseDetail, error) {
	c, err := s.repo.Get(ctx, id)
	if err != nil {
		return CaseDetail{}, err
	}
	if c.Survey != nil {
		c.Survey.Expired = surveyIsExpired(c.Survey.ObservedAt, s.clock.Now())
	}
	timeline := append([]domain.AuditEvent(nil), c.AuditEvents...)
	sort.SliceStable(timeline, func(i, j int) bool {
		if timeline[i].Revision == timeline[j].Revision {
			return timeline[i].OccurredAt.Before(timeline[j].OccurredAt)
		}
		return timeline[i].Revision < timeline[j].Revision
	})
	detail := CaseDetail{Case: c, StatusLabel: c.Status.Label(), NextAction: nextAction(c.Status), Timeline: timeline}
	if c.Risk != nil {
		detail.RiskLabel = c.Risk.FinalLevel.Label()
	}
	return detail, nil
}

func surveyIsExpired(observedAt, now time.Time) bool {
	observedDate := time.Date(observedAt.UTC().Year(), observedAt.UTC().Month(), observedAt.UTC().Day(), 0, 0, 0, 0, time.UTC)
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return observedDate.Before(today.AddDate(0, 0, -30))
}

func (s *Service) Dashboard(ctx context.Context) (Dashboard, error) {
	return s.QueryDashboard(ctx, CaseQuery{})
}

func (s *Service) QueryDashboard(ctx context.Context, query CaseQuery) (Dashboard, error) {
	cases, err := s.repo.List(ctx)
	if err != nil {
		return Dashboard{}, err
	}
	query.Keyword, query.OwnerName, query.Status = strings.TrimSpace(query.Keyword), strings.TrimSpace(query.OwnerName), strings.TrimSpace(query.Status)
	query.RiskLevel, query.DeadlineLevel = strings.TrimSpace(query.RiskLevel), strings.TrimSpace(query.DeadlineLevel)
	result := Dashboard{Cases: make([]CaseSummary, 0, len(cases)), Counts: map[string]int{}, Query: query, Statistics: DashboardStatistics{Statuses: map[string]int{}, RiskLevels: map[string]int{}, DeadlineLevels: map[string]int{}, NextActions: map[string]int{}}}
	now := s.clock.Now()
	for _, c := range cases {
		deadline, deadlineLabel := classifyDeadline(c, now)
		summary := CaseSummary{ID: c.ID, TreeCode: c.TreeCode, Species: c.Species, Location: c.Location, OwnerName: c.OwnerName, DueDate: c.DueDate, Status: c.Status, StatusLabel: c.Status.Label(), Revision: c.Revision, UpdatedAt: c.UpdatedAt, NextAction: nextAction(c.Status), DeadlineLevel: deadline, DeadlineLabel: deadlineLabel}
		if c.Risk != nil {
			summary.RiskLevel, summary.RiskLabel = c.Risk.FinalLevel, c.Risk.FinalLevel.Label()
		}
		if !matchesQuery(summary, query) {
			continue
		}
		result.Cases = append(result.Cases, summary)
		result.Counts[string(c.Status)]++
		result.Statistics.Statuses[string(c.Status)]++
		if summary.RiskLevel != "" {
			result.Statistics.RiskLevels[string(summary.RiskLevel)]++
		}
		result.Statistics.DeadlineLevels[summary.DeadlineLevel]++
		result.Statistics.NextActions[summary.NextAction]++
	}
	result.Counts["all"] = len(result.Cases)
	deadlineRank := map[string]int{"overdue": 0, "due_soon": 1, "normal": 2}
	riskRank := map[domain.RiskLevel]int{domain.RiskCritical: 0, domain.RiskHigh: 1, domain.RiskMedium: 2, domain.RiskLow: 3, "": 4}
	sort.SliceStable(result.Cases, func(i, j int) bool {
		a, b := result.Cases[i], result.Cases[j]
		if deadlineRank[a.DeadlineLevel] != deadlineRank[b.DeadlineLevel] {
			return deadlineRank[a.DeadlineLevel] < deadlineRank[b.DeadlineLevel]
		}
		if riskRank[a.RiskLevel] != riskRank[b.RiskLevel] {
			return riskRank[a.RiskLevel] < riskRank[b.RiskLevel]
		}
		if a.DueDate != b.DueDate {
			return a.DueDate < b.DueDate
		}
		if !a.UpdatedAt.Equal(b.UpdatedAt) {
			return a.UpdatedAt.After(b.UpdatedAt)
		}
		return a.ID < b.ID
	})
	return result, nil
}

func classifyDeadline(c *domain.CareCase, now time.Time) (string, string) {
	if c.Status == domain.StatusClosed {
		return "normal", "已关闭"
	}
	due, err := time.Parse("2006-01-02", c.DueDate)
	if err != nil {
		return "normal", "正常"
	}
	today := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	if due.Before(today) {
		return "overdue", "已逾期"
	}
	if !due.After(today.AddDate(0, 0, 7)) {
		return "due_soon", "七日内到期"
	}
	return "normal", "正常"
}

func matchesQuery(c CaseSummary, q CaseQuery) bool {
	keyword := strings.ToLower(q.Keyword)
	if keyword != "" && !strings.Contains(strings.ToLower(c.TreeCode), keyword) && !strings.Contains(strings.ToLower(c.Location), keyword) {
		return false
	}
	if q.OwnerName != "" && c.OwnerName != q.OwnerName {
		return false
	}
	if q.Status != "" && string(c.Status) != q.Status {
		return false
	}
	if q.RiskLevel != "" && string(c.RiskLevel) != q.RiskLevel {
		return false
	}
	return q.DeadlineLevel == "" || c.DeadlineLevel == q.DeadlineLevel
}

func nextAction(status domain.Status) string {
	switch status {
	case domain.StatusDraft:
		return "提交现状并完成风险评估"
	case domain.StatusAssessed:
		return "编制并提交养护方案"
	case domain.StatusPendingReview:
		return "等待技术审核"
	case domain.StatusApproved:
		return "登记现场实施批次"
	case domain.StatusImplementing:
		return "补充实施并重新提交"
	case domain.StatusPendingAcceptance:
		return "逐项执行验收"
	case domain.StatusClosed:
		return "流程已完成"
	default:
		return "查看任务"
	}
}
