package domain

type Status string

const (
	StatusDraft             Status = "draft"
	StatusAssessed          Status = "assessed"
	StatusPendingReview     Status = "pending_review"
	StatusApproved          Status = "approved"
	StatusImplementing      Status = "implementing"
	StatusPendingAcceptance Status = "pending_acceptance"
	StatusClosed            Status = "closed"
)

func (s Status) Label() string {
	switch s {
	case StatusDraft:
		return "草稿"
	case StatusAssessed:
		return "已评估"
	case StatusPendingReview:
		return "待审核"
	case StatusApproved:
		return "已批准"
	case StatusImplementing:
		return "实施中"
	case StatusPendingAcceptance:
		return "待验收"
	case StatusClosed:
		return "已关闭"
	default:
		return string(s)
	}
}
