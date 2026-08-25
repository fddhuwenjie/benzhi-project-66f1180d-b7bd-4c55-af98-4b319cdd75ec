package domain

import (
	"fmt"
	"strings"
	"time"
)

type CareCase struct {
	ID              string             `json:"id"`
	TreeCode        string             `json:"tree_code"`
	Species         string             `json:"species"`
	Location        string             `json:"location"`
	OwnerName       string             `json:"owner_name"`
	DueDate         string             `json:"due_date"`
	Status          Status             `json:"status"`
	Revision        int64              `json:"revision"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	Survey          *ConditionSurvey   `json:"survey,omitempty"`
	Risk            *RiskAssessment    `json:"risk,omitempty"`
	RiskAssessments []RiskAssessment   `json:"risk_assessments,omitempty"`
	Plans           []CarePlan         `json:"plans,omitempty"`
	Executions      []ExecutionRecord  `json:"executions,omitempty"`
	Acceptances     []AcceptanceRecord `json:"acceptances,omitempty"`
	AuditEvents     []AuditEvent       `json:"audit_events"`
}

type CaseProfileInput struct {
	Species, Location, OwnerName, DueDate string
}

func (c *CareCase) ReviseProfile(in CaseProfileInput, actor, requestID string, now time.Time) (bool, error) {
	if c.Status != StatusDraft {
		return false, &StateError{Status: c.Status, Action: "修订任务资料"}
	}
	if err := validateCaseFields(c.TreeCode, in.Species, in.Location, in.OwnerName, in.DueDate); err != nil {
		return false, err
	}
	changes := make([]string, 0, 4)
	if value := strings.TrimSpace(in.Location); value != c.Location {
		c.Location = value
		changes = append(changes, "位置")
	}
	if value := strings.TrimSpace(in.Species); value != c.Species {
		c.Species = value
		changes = append(changes, "树种")
	}
	if value := strings.TrimSpace(in.OwnerName); value != c.OwnerName {
		c.OwnerName = value
		changes = append(changes, "责任人")
	}
	if in.DueDate != c.DueDate {
		c.DueDate = in.DueDate
		changes = append(changes, "任务期限")
	}
	if len(changes) == 0 {
		return false, nil
	}
	c.record("case.profile_revised", actor, c.Status, c.Status, "修订任务资料："+strings.Join(changes, "、"), requestID, now)
	return true, nil
}

type NewCaseInput struct {
	ID, TreeCode, Species, Location, OwnerName, DueDate string
	Actor, RequestID                                    string
	Now                                                 time.Time
}

func NewCareCase(in NewCaseInput) (*CareCase, error) {
	if strings.TrimSpace(in.ID) == "" {
		return nil, invalid("id", "任务 ID 不能为空")
	}
	if err := validateCaseFields(in.TreeCode, in.Species, in.Location, in.OwnerName, in.DueDate); err != nil {
		return nil, err
	}
	if strings.TrimSpace(in.Actor) == "" {
		return nil, invalid("actor_name", "操作人不能为空")
	}
	if in.Now.IsZero() {
		return nil, invalid("created_at", "创建时间不能为空")
	}
	c := &CareCase{
		ID: in.ID, TreeCode: strings.TrimSpace(in.TreeCode), Species: strings.TrimSpace(in.Species),
		Location: strings.TrimSpace(in.Location), OwnerName: strings.TrimSpace(in.OwnerName),
		DueDate: in.DueDate, Status: StatusDraft, CreatedAt: in.Now.UTC(), UpdatedAt: in.Now.UTC(),
	}
	c.record("case.created", in.Actor, StatusDraft, StatusDraft, "创建养护任务草稿", in.RequestID, in.Now)
	return c, nil
}

func validateCaseFields(treeCode, species, location, owner, dueDate string) error {
	fields := []struct{ name, value, message string }{
		{"tree_code", treeCode, "树木编号不能为空"}, {"species", species, "树种不能为空"},
		{"location", location, "位置不能为空"}, {"owner_name", owner, "责任人不能为空"},
		{"due_date", dueDate, "任务期限不能为空"},
	}
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			return invalid(f.name, f.message)
		}
	}
	if _, err := time.Parse("2006-01-02", dueDate); err != nil {
		return invalid("due_date", "任务期限必须为 YYYY-MM-DD")
	}
	return nil
}

func (c *CareCase) SubmitSurvey(s ConditionSurvey, actor, requestID string, now time.Time) error {
	if c.Status != StatusDraft {
		return &StateError{Status: c.Status, Action: "现状采集"}
	}
	if err := ValidateSurvey(s); err != nil {
		return err
	}
	s.CaseID = c.ID
	s.ObservedAt = s.ObservedAt.UTC()
	c.Survey = &s
	c.record("survey.submitted", actor, c.Status, c.Status, "提交树冠、树干、根区和环境现状记录", requestID, now)
	return nil
}

func ValidateSurvey(s ConditionSurvey) error {
	if strings.TrimSpace(s.ID) == "" {
		return invalid("survey.id", "现状记录 ID 不能为空")
	}
	observations := []struct {
		name string
		obs  Observation
	}{{"crown_condition", s.Crown}, {"trunk_condition", s.Trunk}, {"root_zone_condition", s.RootZone}}
	for _, item := range observations {
		if !item.obs.Condition.Valid() {
			return invalid(item.name+".condition", "请选择有效的观测等级")
		}
		if strings.TrimSpace(item.obs.Notes) == "" {
			return invalid(item.name+".notes", "观测说明不能为空")
		}
	}
	if strings.TrimSpace(s.Environment.Notes) == "" {
		return invalid("environment.notes", "周边环境说明不能为空")
	}
	if s.ObservedAt.IsZero() {
		return invalid("observed_at", "观测时间不能为空")
	}
	if strings.TrimSpace(s.Observer) == "" {
		return invalid("observer_name", "观测人不能为空")
	}
	for i, p := range s.PhotoRefs {
		if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.URL) == "" {
			return invalid(fmt.Sprintf("photo_refs.%d", i), "照片名称和引用地址不能为空")
		}
	}
	return nil
}

func (c *CareCase) ApplyRisk(r RiskAssessment, actor, requestID string, now time.Time) error {
	if c.Status != StatusDraft {
		return &StateError{Status: c.Status, Action: "风险评估"}
	}
	if c.Survey == nil {
		return invalid("survey", "必须先提交现状记录")
	}
	if !r.AutomaticLevel.Valid() || !r.FinalLevel.Valid() || r.AutomaticScore < 0 {
		return invalid("risk", "风险评估结果无效")
	}
	if r.ManualLevel != "" && r.ManualLevel != r.AutomaticLevel && strings.TrimSpace(r.ManualReason) == "" {
		return invalid("manual_reason", "调整自动风险等级时必须填写人工判定说明")
	}
	r.AssessedAt = r.AssessedAt.UTC()
	c.Risk = &r
	from := c.Status
	c.Status = StatusAssessed
	c.record("risk.assessed", actor, from, c.Status, fmt.Sprintf("风险评估完成：%s，命中 %d 项因子", r.FinalLevel.Label(), len(r.Factors)), requestID, now)
	return nil
}

func (c *CareCase) SavePlan(p CarePlan, actor, requestID string, now time.Time) error {
	if c.Status != StatusAssessed {
		return &StateError{Status: c.Status, Action: "编制养护方案"}
	}
	if err := ValidatePlan(p); err != nil {
		return err
	}
	p.CaseID = c.ID
	p.Version = len(c.Plans) + 1
	p.ReviewStatus = ReviewDraft
	p.CreatedAt = now.UTC()
	c.Plans = append(c.Plans, p)
	c.record("plan.saved", actor, c.Status, c.Status, fmt.Sprintf("保存养护方案 V%d", p.Version), requestID, now)
	return nil
}

func ValidatePlan(p CarePlan) error {
	if strings.TrimSpace(p.ID) == "" {
		return invalid("plan.id", "方案 ID 不能为空")
	}
	if strings.TrimSpace(p.PreparedBy) == "" {
		return invalid("prepared_by", "编制人不能为空")
	}
	if len(p.Measures) == 0 {
		return invalid("measures", "至少填写一项养护措施")
	}
	for i, m := range p.Measures {
		if strings.TrimSpace(m.Title) == "" || strings.TrimSpace(m.Description) == "" {
			return invalid(fmt.Sprintf("measures.%d", i), "措施标题和说明不能为空")
		}
	}
	if len(cleanStrings(p.Materials)) == 0 {
		return invalid("materials", "至少填写一种材料或填写无需材料")
	}
	if strings.TrimSpace(p.WorkWindow) == "" {
		return invalid("work_window", "作业窗口不能为空")
	}
	if len(cleanStrings(p.SafetyControls)) == 0 {
		return invalid("safety_controls", "至少填写一个安全控制点")
	}
	if len(cleanStrings(p.CompletionCriteria)) == 0 {
		return invalid("completion_criteria", "至少填写一项完成标准")
	}
	return nil
}

func (c *CareCase) SubmitPlan(actor, requestID string, now time.Time) error {
	if c.Status != StatusAssessed || len(c.Plans) == 0 {
		return &StateError{Status: c.Status, Action: "提交方案审核"}
	}
	p := &c.Plans[len(c.Plans)-1]
	if p.ReviewStatus != ReviewDraft {
		return invalid("review_status", "只有草稿方案可以提交审核")
	}
	p.ReviewStatus = ReviewPending
	from := c.Status
	c.Status = StatusPendingReview
	c.record("plan.submitted", actor, from, c.Status, fmt.Sprintf("提交方案 V%d 技术审核", p.Version), requestID, now)
	return nil
}

func (c *CareCase) ReviewPlan(approved bool, opinions []ReviewOpinion, reviewer, requestID string, now time.Time) error {
	if c.Status != StatusPendingReview || len(c.Plans) == 0 {
		return &StateError{Status: c.Status, Action: "技术审核"}
	}
	if strings.TrimSpace(reviewer) == "" {
		return invalid("reviewer", "审核人不能为空")
	}
	if len(opinions) == 0 {
		return invalid("review_notes", "至少填写一项审核意见")
	}
	for i, opinion := range opinions {
		if strings.TrimSpace(opinion.Item) == "" || strings.TrimSpace(opinion.Opinion) == "" {
			return invalid(fmt.Sprintf("review_notes.%d", i), "审核项和意见不能为空")
		}
	}
	p := &c.Plans[len(c.Plans)-1]
	p.ReviewNotes = append([]ReviewOpinion(nil), opinions...)
	p.ReviewedBy = reviewer
	t := now.UTC()
	p.ReviewedAt = &t
	from := c.Status
	if approved {
		p.ReviewStatus = ReviewApproved
		c.Status = StatusApproved
		c.record("plan.approved", reviewer, from, c.Status, fmt.Sprintf("批准方案 V%d", p.Version), requestID, now)
	} else {
		p.ReviewStatus = ReviewRejected
		c.Status = StatusAssessed
		c.record("plan.rejected", reviewer, from, c.Status, fmt.Sprintf("驳回方案 V%d，需重新编制", p.Version), requestID, now)
	}
	return nil
}

func (c *CareCase) RecordExecution(e ExecutionRecord, actor, requestID string, now time.Time) error {
	if c.Status != StatusApproved && c.Status != StatusImplementing {
		return &StateError{Status: c.Status, Action: "登记现场实施"}
	}
	if len(c.Plans) == 0 || c.Plans[len(c.Plans)-1].ReviewStatus != ReviewApproved {
		return invalid("plan", "没有已批准的养护方案")
	}
	p := c.Plans[len(c.Plans)-1]
	if err := ValidateExecution(e, p); err != nil {
		return err
	}
	e.CaseID = c.ID
	e.PlanVersion = p.Version
	e.PerformedAt = e.PerformedAt.UTC()
	c.Executions = append(c.Executions, e)
	from := c.Status
	c.Status = StatusPendingAcceptance
	c.record("execution.submitted", actor, from, c.Status, fmt.Sprintf("提交第 %d 个现场实施批次，控制点已全部落实", len(c.Executions)), requestID, now)
	return nil
}

func ValidateExecution(e ExecutionRecord, p CarePlan) error {
	if strings.TrimSpace(e.ID) == "" {
		return invalid("execution.id", "实施记录 ID 不能为空")
	}
	if e.PerformedAt.IsZero() {
		return invalid("performed_at", "实施时间不能为空")
	}
	if strings.TrimSpace(e.SubmittedBy) == "" || len(cleanStrings(e.CrewNames)) == 0 {
		return invalid("crew_names", "提交人和作业人员不能为空")
	}
	if len(cleanStrings(e.ActualMeasures)) == 0 {
		return invalid("actual_measures", "实际措施不能为空")
	}
	if len(e.EvidenceRefs) == 0 {
		return invalid("evidence_refs", "至少提交一项实施证据")
	}
	checks := make(map[string]bool, len(e.ControlChecks))
	for _, check := range e.ControlChecks {
		checks[strings.TrimSpace(check.Control)] = check.Passed
	}
	for _, required := range p.SafetyControls {
		if !checks[strings.TrimSpace(required)] {
			return invalid("control_checks", "安全控制点未全部落实："+required)
		}
	}
	return nil
}

func (c *CareCase) Accept(a AcceptanceRecord, actor, requestID string, now time.Time) error {
	if c.Status != StatusPendingAcceptance {
		return &StateError{Status: c.Status, Action: "验收"}
	}
	if strings.TrimSpace(a.Inspector) == "" || a.InspectedAt.IsZero() {
		return invalid("inspector", "验收人和验收时间不能为空")
	}
	p := c.Plans[len(c.Plans)-1]
	if len(a.CriterionResults) != len(p.CompletionCriteria) {
		return invalid("criterion_results", "必须逐项记录全部完成标准的核验结果")
	}
	if len(cleanStrings(a.CriterionResults)) != len(a.CriterionResults) {
		return invalid("criterion_results", "完成标准核验结果不能为空")
	}
	if !a.Passed && len(cleanStrings(a.Nonconformities)) == 0 {
		return invalid("nonconformities", "验收不通过时必须填写不符合项")
	}
	a.InspectedAt = a.InspectedAt.UTC()
	c.Acceptances = append(c.Acceptances, a)
	last := &c.Executions[len(c.Executions)-1]
	from := c.Status
	if a.Passed {
		last.AcceptanceResult = "passed"
		c.Status = StatusClosed
		c.record("acceptance.passed", actor, from, c.Status, "验收通过，养护任务关闭", requestID, now)
	} else {
		last.AcceptanceResult = "rejected"
		last.Nonconformities = append([]string(nil), a.Nonconformities...)
		c.Status = StatusImplementing
		c.record("acceptance.rejected", actor, from, c.Status, fmt.Sprintf("验收退回，发现 %d 项不符合", len(a.Nonconformities)), requestID, now)
	}
	return nil
}

func (c *CareCase) record(eventType, actor string, from, to Status, summary, requestID string, now time.Time) {
	c.Revision++
	c.UpdatedAt = now.UTC()
	c.AuditEvents = append(c.AuditEvents, AuditEvent{
		ID: fmt.Sprintf("%s-%06d", c.ID, c.Revision), CaseID: c.ID, EventType: eventType,
		ActorName: strings.TrimSpace(actor), OccurredAt: now.UTC(), FromStatus: from, ToStatus: to,
		Summary: summary, RequestID: requestID, Revision: c.Revision,
	})
}

func (c *CareCase) LatestPlan() *CarePlan {
	if len(c.Plans) == 0 {
		return nil
	}
	return &c.Plans[len(c.Plans)-1]
}

func cleanStrings(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		if v := strings.TrimSpace(value); v != "" {
			result = append(result, v)
		}
	}
	return result
}
