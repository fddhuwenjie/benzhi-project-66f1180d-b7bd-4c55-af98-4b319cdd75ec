package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

const (
	PhotoPartCrown       = "crown"
	PhotoPartTrunk       = "trunk"
	PhotoPartRootZone    = "root_zone"
	PhotoPartEnvironment = "environment"
)

var ReviewChecklist = []string{"风险措施", "材料适用性", "作业窗口", "安全控制", "完成标准"}

func (c *CareCase) SubmitSurveyAt(s ConditionSurvey, actor, requestID string, now time.Time) error {
	if c.Status != StatusDraft {
		return &StateError{Status: c.Status, Action: "现状采集"}
	}
	if err := ValidateSurveyAt(s, now); err != nil {
		return err
	}
	s.CaseID = c.ID
	s.ObservedAt = s.ObservedAt.UTC()
	s.SubmittedAt = now.UTC()
	s.Expired = surveyExpired(s.ObservedAt, now)
	c.Survey = &s
	summary := "提交树冠、树干、根区和环境现状记录"
	if s.Expired {
		summary += "（观测已超过三十个自然日）"
	}
	c.record("survey.submitted", actor, c.Status, c.Status, summary, requestID, now)
	return nil
}

func ValidateSurveyAt(s ConditionSurvey, now time.Time) error {
	if err := ValidateSurvey(s); err != nil {
		return err
	}
	if s.ObservedAt.After(now) {
		return invalid("observed_at", "观测时间不得晚于提交时间")
	}
	parts := map[string]bool{}
	seen := map[string]bool{}
	for i, photo := range s.PhotoRefs {
		part := strings.TrimSpace(photo.Part)
		if part != PhotoPartCrown && part != PhotoPartTrunk && part != PhotoPartRootZone && part != PhotoPartEnvironment {
			return invalid(fmt.Sprintf("photo_refs.%d.part", i), "照片对应部位必须是 crown、trunk、root_zone 或 environment")
		}
		taken, err := time.Parse(time.RFC3339, strings.TrimSpace(photo.TakenAt))
		if err != nil {
			return invalid(fmt.Sprintf("photo_refs.%d.taken_at", i), "照片拍摄时间必须为 RFC3339 格式")
		}
		if taken.After(now) {
			return invalid(fmt.Sprintf("photo_refs.%d.taken_at", i), "照片拍摄时间不得晚于提交时间")
		}
		key := strings.TrimSpace(photo.Name) + "\x00" + strings.TrimSpace(photo.URL)
		if seen[key] {
			return invalid("photo_refs", "照片名称与引用地址组合不得重复")
		}
		seen[key], parts[part] = true, true
	}
	missing := make([]string, 0, 4)
	if (s.Crown.Condition == ConditionPoor || s.Crown.Condition == ConditionCritical) && !parts[PhotoPartCrown] {
		missing = append(missing, "树冠")
	}
	if (s.Trunk.Condition == ConditionPoor || s.Trunk.Condition == ConditionCritical) && !parts[PhotoPartTrunk] {
		missing = append(missing, "树干")
	}
	if (s.RootZone.Condition == ConditionPoor || s.RootZone.Condition == ConditionCritical) && !parts[PhotoPartRootZone] {
		missing = append(missing, "根区")
	}
	if (s.Environment.SoilCompaction || s.Environment.NearbyConstruction || s.Environment.HeavyTraffic || s.Environment.DrainageProblem) && !parts[PhotoPartEnvironment] {
		missing = append(missing, "周边环境")
	}
	if len(missing) > 0 {
		return invalid("photo_refs", "缺少对应部位照片："+strings.Join(missing, "、"))
	}
	return nil
}

func surveyExpired(observedAt, now time.Time) bool {
	observedDate := time.Date(observedAt.UTC().Year(), observedAt.UTC().Month(), observedAt.UTC().Day(), 0, 0, 0, 0, time.UTC)
	nowDate := time.Date(now.UTC().Year(), now.UTC().Month(), now.UTC().Day(), 0, 0, 0, 0, time.UTC)
	return observedDate.Before(nowDate.AddDate(0, 0, -30))
}

func (c *CareCase) ApplyRiskReview(r RiskAssessment, actor, requestID string, now time.Time) error {
	if c.Status != StatusDraft && !(c.Status == StatusAssessed && len(c.Plans) == 0) {
		return &StateError{Status: c.Status, Action: "风险评估复核"}
	}
	if c.Survey == nil {
		return invalid("survey", "必须先提交现状记录")
	}
	if surveyExpired(c.Survey.ObservedAt, now) {
		return invalid("survey.observed_at", "现状观测已超过三十个自然日，请重新观测后再评估")
	}
	if err := validateRiskOverride(r); err != nil {
		return err
	}
	previous := c.Risk
	r.Sequence = len(c.RiskAssessments) + 1
	r.SurveyID = c.Survey.ID
	r.AssessedAt = now.UTC()
	if previous != nil {
		r.DifferenceSummary = fmt.Sprintf("最终等级：%s → %s；风险因子：%d → %d 项", previous.FinalLevel.Label(), r.FinalLevel.Label(), len(previous.Factors), len(r.Factors))
	}
	from := c.Status
	c.RiskAssessments = append(c.RiskAssessments, r)
	c.Risk = &c.RiskAssessments[len(c.RiskAssessments)-1]
	c.Status = StatusAssessed
	event := "risk.assessed"
	summary := fmt.Sprintf("风险评估完成：自动%s，最终%s，命中 %d 项因子", r.AutomaticLevel.Label(), r.FinalLevel.Label(), len(r.Factors))
	if previous != nil {
		event = "risk.reassessed"
		summary = "风险评估复核完成；" + r.DifferenceSummary
	}
	c.record(event, actor, from, c.Status, summary, requestID, now)
	c.Risk.Revision = c.Revision
	c.RiskAssessments[len(c.RiskAssessments)-1].Revision = c.Revision
	for i := range c.Plans {
		if c.Plans[i].AssessmentRevision != c.Revision {
			c.Plans[i].AssessmentMismatch = true
		}
	}
	return nil
}

func validateRiskOverride(r RiskAssessment) error {
	if !r.AutomaticLevel.Valid() || !r.FinalLevel.Valid() || r.AutomaticScore < 0 {
		return invalid("risk", "风险评估结果无效")
	}
	if r.FinalLevel == r.AutomaticLevel {
		return nil
	}
	if strings.TrimSpace(r.ManualReason) == "" {
		return invalid("manual_reason", "人工等级不同于自动等级时必须填写具体理由")
	}
	ranks := map[RiskLevel]int{RiskLow: 1, RiskMedium: 2, RiskHigh: 3, RiskCritical: 4}
	if ranks[r.FinalLevel] < ranks[r.AutomaticLevel]-1 {
		return invalid("manual_level", "人工降低风险不得跨越两个等级")
	}
	if r.AutomaticLevel == RiskCritical && r.FinalLevel != RiskCritical && r.FinalLevel != RiskHigh {
		return invalid("manual_level", "自动重大风险只能维持重大风险或降低为高风险")
	}
	return nil
}

func (c *CareCase) SavePlanForAssessment(p CarePlan, actor, requestID string, now time.Time) error {
	if c.Status != StatusAssessed || c.Risk == nil {
		return &StateError{Status: c.Status, Action: "编制养护方案"}
	}
	if err := ValidatePlan(p); err != nil {
		return err
	}
	p.CaseID = c.ID
	p.Version = len(c.Plans) + 1
	p.ReviewStatus = ReviewDraft
	p.CreatedAt = now.UTC()
	p.AssessmentRevision = c.Risk.Revision
	p.RiskRuleIDs = factorIDs(c.Risk.Factors)
	p.Coverage = buildCoverage(c.Risk.Factors, p.Measures, p.Exemptions)
	if previous := c.LatestPlan(); previous != nil && previous.ReviewStatus == ReviewRejected {
		p.RemediationResponses = append([]RemediationResponse(nil), p.RemediationResponses...)
	}
	c.Plans = append(c.Plans, p)
	c.record("plan.saved", actor, c.Status, c.Status, fmt.Sprintf("保存养护方案 V%d，%d/%d 项风险已覆盖或说明", p.Version, coveredCount(p.Coverage), len(p.Coverage)), requestID, now)
	return nil
}

func buildCoverage(factors []RiskFactor, measures []PlanMeasure, exemptions []RiskExemption) []RiskCoverage {
	exemptionByID := map[string]string{}
	for _, item := range exemptions {
		exemptionByID[strings.TrimSpace(item.RiskRuleID)] = strings.TrimSpace(item.Reason)
	}
	result := make([]RiskCoverage, 0, len(factors))
	for _, factor := range factors {
		entry := RiskCoverage{RiskRuleID: factor.RuleID, RiskTitle: factor.Title, RiskLevel: factor.Level, ExemptionReason: exemptionByID[factor.RuleID]}
		for _, measure := range measures {
			ids := append([]string(nil), measure.RiskRuleIDs...)
			if measure.RiskRuleID != "" {
				ids = append(ids, measure.RiskRuleID)
			}
			if contains(ids, factor.RuleID) {
				entry.MeasureTitles = append(entry.MeasureTitles, measure.Title)
				entry.ControlPoints = append(entry.ControlPoints, measure.ControlPoints...)
				if strings.TrimSpace(measure.CompletionStandard) != "" {
					entry.CompletionStandards = append(entry.CompletionStandards, measure.CompletionStandard)
				}
			}
		}
		entry.Covered = len(entry.MeasureTitles) > 0 && len(cleanStrings(entry.ControlPoints)) > 0 && len(cleanStrings(entry.CompletionStandards)) > 0
		if !entry.Covered && entry.ExemptionReason != "" && factor.Level != RiskHigh && factor.Level != RiskCritical {
			entry.Covered = true
		}
		result = append(result, entry)
	}
	return result
}

func (c *CareCase) SubmitPlanCovered(actor, requestID string, now time.Time) error {
	if c.Status != StatusAssessed || len(c.Plans) == 0 || c.Risk == nil {
		return &StateError{Status: c.Status, Action: "提交方案审核"}
	}
	p := &c.Plans[len(c.Plans)-1]
	if p.ReviewStatus != ReviewDraft {
		return invalid("review_status", "只有草稿方案可以提交审核")
	}
	if p.AssessmentMismatch || p.AssessmentRevision != c.Risk.Revision || !sameStrings(p.RiskRuleIDs, factorIDs(c.Risk.Factors)) {
		p.AssessmentMismatch = true
		return invalid("assessment_revision", "方案与当前有效评估失配，请基于最新风险重新编制")
	}
	missing := make([]string, 0)
	for _, item := range p.Coverage {
		if !item.Covered {
			missing = append(missing, item.RiskTitle+"（"+item.RiskRuleID+"）")
		}
	}
	if len(missing) > 0 {
		return invalid("coverage", "风险因子尚未覆盖或不允许豁免："+strings.Join(missing, "、"))
	}
	if err := validateRemediationResponses(c.Plans); err != nil {
		return err
	}
	p.ReviewStatus = ReviewPending
	from := c.Status
	c.Status = StatusPendingReview
	c.record("plan.submitted", actor, from, c.Status, fmt.Sprintf("提交方案 V%d 技术审核，风险覆盖完整", p.Version), requestID, now)
	return nil
}

func validateRemediationResponses(plans []CarePlan) error {
	if len(plans) < 2 {
		return nil
	}
	previous, current := plans[len(plans)-2], plans[len(plans)-1]
	if previous.ReviewStatus != ReviewRejected {
		return nil
	}
	responses := map[string]bool{}
	for _, response := range current.RemediationResponses {
		if strings.TrimSpace(response.Response) != "" {
			responses[response.OpinionID] = true
		}
	}
	missing := make([]string, 0)
	for _, opinion := range previous.ReviewNotes {
		if opinion.Result == "needs_changes" && !responses[opinion.ID] {
			missing = append(missing, opinion.Item)
		}
	}
	if len(missing) > 0 {
		return invalid("remediation_responses", "尚未回应上一版本审核意见："+strings.Join(missing, "、"))
	}
	return nil
}

func (c *CareCase) ReviewPlanChecklist(approved bool, opinions []ReviewOpinion, reviewer, requestID string, now time.Time) error {
	if c.Status != StatusPendingReview || len(c.Plans) == 0 {
		return &StateError{Status: c.Status, Action: "技术审核"}
	}
	if strings.TrimSpace(reviewer) == "" {
		return invalid("reviewer", "审核人不能为空")
	}
	byItem := map[string]ReviewOpinion{}
	for _, opinion := range opinions {
		if opinion.Result != "passed" && opinion.Result != "needs_changes" {
			return invalid("opinions", "审核结论必须为 passed 或 needs_changes")
		}
		if strings.TrimSpace(opinion.Opinion) == "" {
			return invalid("opinions", "每项审核结论都必须填写具体意见")
		}
		if _, exists := byItem[opinion.Item]; exists {
			return invalid("opinions", "审核检查项不得重复")
		}
		byItem[opinion.Item] = opinion
	}
	normalized := make([]ReviewOpinion, 0, len(ReviewChecklist))
	hasChanges := false
	for i, item := range ReviewChecklist {
		opinion, ok := byItem[item]
		if !ok {
			return invalid("opinions", "审核清单缺少检查项："+item)
		}
		opinion.ID = fmt.Sprintf("V%d-R%02d", c.Plans[len(c.Plans)-1].Version, i+1)
		opinion.Item = item
		hasChanges = hasChanges || opinion.Result == "needs_changes"
		normalized = append(normalized, opinion)
	}
	if approved && hasChanges {
		return invalid("approved", "批准要求五类检查项全部通过")
	}
	if !approved && !hasChanges {
		return invalid("approved", "驳回要求至少一项结论为需修改")
	}
	p := &c.Plans[len(c.Plans)-1]
	p.ReviewNotes, p.ReviewedBy = normalized, strings.TrimSpace(reviewer)
	t := now.UTC()
	p.ReviewedAt = &t
	from := c.Status
	if approved {
		p.ReviewStatus, c.Status = ReviewApproved, StatusApproved
		c.record("plan.approved", reviewer, from, c.Status, fmt.Sprintf("方案 V%d 通过五类技术审核", p.Version), requestID, now)
	} else {
		p.ReviewStatus, c.Status = ReviewRejected, StatusAssessed
		c.record("plan.rejected", reviewer, from, c.Status, fmt.Sprintf("驳回方案 V%d，%d 项需要整改", p.Version, changeCount(normalized)), requestID, now)
	}
	return nil
}

func (c *CareCase) RegisterExecution(e ExecutionRecord, actor, requestID string, now time.Time) error {
	if c.Status != StatusApproved && c.Status != StatusImplementing {
		return &StateError{Status: c.Status, Action: "登记现场实施批次"}
	}
	p := c.approvedPlan()
	if p == nil {
		return invalid("plan", "没有已批准的养护方案")
	}
	if err := validateExecutionBatch(e, *p, c.Executions, c.openNonconformities()); err != nil {
		return err
	}
	e.CaseID, e.PlanVersion, e.BatchNumber = c.ID, p.Version, len(c.Executions)+1
	e.PerformedAt = e.PerformedAt.UTC()
	c.Executions = append(c.Executions, e)
	from := c.Status
	c.Status = StatusImplementing
	c.record("execution.batch_recorded", actor, from, c.Status, fmt.Sprintf("登记实施批次 %d，包含 %d 项实际措施和 %d 项证据", e.BatchNumber, len(e.ActualMeasures), len(e.EvidenceRefs)), requestID, now)
	return nil
}

func validateExecutionBatch(e ExecutionRecord, p CarePlan, existing []ExecutionRecord, open map[string]*Nonconformity) error {
	if strings.TrimSpace(e.ID) == "" {
		return invalid("execution.id", "实施记录 ID 不能为空")
	}
	if e.PerformedAt.IsZero() {
		return invalid("performed_at", "实施时间不能为空")
	}
	if p.ReviewedAt != nil && e.PerformedAt.Before(*p.ReviewedAt) {
		return invalid("performed_at", "实施时间不得早于方案批准时间")
	}
	if len(existing) > 0 && e.PerformedAt.Before(existing[len(existing)-1].PerformedAt) {
		return invalid("performed_at", "实施时间不得早于上一批次")
	}
	if strings.TrimSpace(e.SubmittedBy) == "" || len(cleanStrings(e.CrewNames)) == 0 {
		return invalid("crew_names", "提交人和作业人员不能为空")
	}
	if len(cleanStrings(e.ActualMeasures)) == 0 {
		return invalid("actual_measures", "实际措施不能为空")
	}
	if len(e.EvidenceRefs) == 0 {
		return invalid("evidence_refs", "每个批次至少提交一项证据")
	}
	allowedControls := map[string]bool{}
	for _, v := range p.SafetyControls {
		allowedControls[strings.TrimSpace(v)] = true
	}
	for _, check := range e.ControlChecks {
		if !allowedControls[strings.TrimSpace(check.Control)] {
			return invalid("control_checks", "控制点不属于批准方案版本："+check.Control)
		}
	}
	seenEvidence := map[string]bool{}
	for _, old := range existing {
		for _, ref := range old.EvidenceRefs {
			seenEvidence[strings.TrimSpace(ref.URL)] = true
		}
	}
	for _, ref := range e.EvidenceRefs {
		if strings.TrimSpace(ref.Name) == "" || strings.TrimSpace(ref.URL) == "" {
			return invalid("evidence_refs", "证据名称和引用地址不能为空")
		}
		if seenEvidence[strings.TrimSpace(ref.URL)] {
			return invalid("evidence_refs", "证据引用不得重复："+ref.URL)
		}
		seenEvidence[strings.TrimSpace(ref.URL)] = true
	}
	if len(open) > 0 && len(e.Remediations) == 0 {
		return invalid("remediations", "存在待整改问题时，补充实施批次必须关联至少一个待整改编号")
	}
	for _, remediation := range e.Remediations {
		if open[remediation.NonconformityID] == nil {
			return invalid("remediations", "整改编号不存在或已销项："+remediation.NonconformityID)
		}
		if strings.TrimSpace(remediation.Description) == "" {
			return invalid("remediations", "整改说明不能为空")
		}
	}
	return nil
}

func (c *CareCase) CompleteExecution(actor, requestID string, now time.Time) error {
	if c.Status != StatusImplementing {
		return &StateError{Status: c.Status, Action: "提交完工汇总"}
	}
	p := c.approvedPlan()
	if p == nil {
		return invalid("plan", "没有已批准的养护方案")
	}
	measureAvailable, controlHits := map[string]int{}, map[string]bool{}
	for _, execution := range c.Executions {
		if execution.PlanVersion != p.Version {
			continue
		}
		for _, measure := range execution.ActualMeasures {
			measureAvailable[strings.TrimSpace(measure)]++
		}
		for _, check := range execution.ControlChecks {
			if check.Passed {
				controlHits[strings.TrimSpace(check.Control)] = true
			}
		}
	}
	missing := make([]string, 0)
	for _, measure := range p.Measures {
		title := strings.TrimSpace(measure.Title)
		if measureAvailable[title] <= 0 {
			missing = append(missing, "措施："+measure.Title)
		} else {
			measureAvailable[title]--
		}
	}
	for _, control := range p.SafetyControls {
		if !controlHits[strings.TrimSpace(control)] {
			missing = append(missing, "控制点："+control)
		}
	}
	open := c.openNonconformities()
	for id := range open {
		resolved := false
		for i := range c.Executions {
			for _, remediation := range c.Executions[i].Remediations {
				if remediation.NonconformityID == id && len(c.Executions[i].EvidenceRefs) > 0 {
					resolved = true
					open[id].ResolvedByBatches = append(open[id].ResolvedByBatches, c.Executions[i].BatchNumber)
				}
			}
		}
		if !resolved {
			missing = append(missing, "待整改："+id)
		} else {
			open[id].Status = "resolved"
		}
	}
	if len(missing) > 0 {
		return invalid("completion", "尚未满足完工条件："+strings.Join(missing, "、"))
	}
	from := c.Status
	c.Status = StatusPendingAcceptance
	c.record("execution.completed", actor, from, c.Status, fmt.Sprintf("汇总 %d 个实施批次，措施、控制点、证据和整改项核验完成", len(c.Executions)), requestID, now)
	return nil
}

func (c *CareCase) AcceptStructured(a AcceptanceRecord, actor, requestID string, now time.Time) error {
	if c.Status != StatusPendingAcceptance {
		return &StateError{Status: c.Status, Action: "验收"}
	}
	if strings.TrimSpace(a.Inspector) == "" || a.InspectedAt.IsZero() {
		return invalid("inspector", "验收人和验收时间不能为空")
	}
	p := c.approvedPlan()
	if p == nil {
		return invalid("plan", "没有已批准的养护方案")
	}
	if len(a.CriterionResults) != len(p.CompletionCriteria) || len(cleanStrings(a.CriterionResults)) != len(a.CriterionResults) {
		return invalid("criterion_results", "必须逐项记录全部完成标准的核验结果")
	}
	if a.Passed && len(a.Items) > 0 {
		return invalid("nonconformities", "验收通过时不能包含不符合项")
	}
	if !a.Passed && len(a.Items) == 0 {
		return invalid("nonconformities", "验收不通过时必须填写不符合项")
	}
	seen := map[string]bool{}
	for i := range a.Items {
		item := &a.Items[i]
		if strings.TrimSpace(item.CompletionCriterion) == "" || !contains(p.CompletionCriteria, item.CompletionCriterion) {
			return invalid("nonconformities", "不符合项必须对应批准方案的完成标准")
		}
		item.Description = strings.TrimSpace(item.Description)
		if item.Description == "" || seen[item.Description] {
			return invalid("nonconformities", "不符合项说明不能为空或重复")
		}
		seen[item.Description] = true
		item.ID = fmt.Sprintf("A%d-NC%d", len(c.Acceptances)+1, i+1)
		item.Status = "pending"
	}
	a.Sequence = len(c.Acceptances) + 1
	a.InspectedAt = a.InspectedAt.UTC()
	if len(a.Items) > 0 {
		for _, item := range a.Items {
			a.Nonconformities = append(a.Nonconformities, item.Description)
		}
	}
	c.Acceptances = append(c.Acceptances, a)
	from := c.Status
	if a.Passed {
		c.Status = StatusClosed
		c.record("acceptance.passed", actor, from, c.Status, fmt.Sprintf("第 %d 次验收通过，养护任务关闭", a.Sequence), requestID, now)
	} else {
		c.Status = StatusImplementing
		c.record("acceptance.rejected", actor, from, c.Status, fmt.Sprintf("第 %d 次验收退回，生成 %d 项待整改问题", a.Sequence, len(a.Items)), requestID, now)
	}
	return nil
}

func (c *CareCase) approvedPlan() *CarePlan {
	for i := len(c.Plans) - 1; i >= 0; i-- {
		if c.Plans[i].ReviewStatus == ReviewApproved {
			return &c.Plans[i]
		}
	}
	return nil
}

func (c *CareCase) openNonconformities() map[string]*Nonconformity {
	result := map[string]*Nonconformity{}
	for i := range c.Acceptances {
		for j := range c.Acceptances[i].Items {
			item := &c.Acceptances[i].Items[j]
			if item.Status == "pending" {
				result[item.ID] = item
			}
		}
	}
	return result
}

func factorIDs(factors []RiskFactor) []string {
	result := make([]string, 0, len(factors))
	for _, factor := range factors {
		result = append(result, factor.RuleID)
	}
	sort.Strings(result)
	return result
}
func coveredCount(items []RiskCoverage) int {
	count := 0
	for _, item := range items {
		if item.Covered {
			count++
		}
	}
	return count
}
func changeCount(items []ReviewOpinion) int {
	count := 0
	for _, item := range items {
		if item.Result == "needs_changes" {
			count++
		}
	}
	return count
}
func contains(values []string, expected string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == strings.TrimSpace(expected) {
			return true
		}
	}
	return false
}
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	aa, bb := append([]string(nil), a...), append([]string(nil), b...)
	sort.Strings(aa)
	sort.Strings(bb)
	for i := range aa {
		if aa[i] != bb[i] {
			return false
		}
	}
	return true
}
