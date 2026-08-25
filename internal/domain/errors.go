package domain

import "fmt"

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e *ValidationError) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

type StateError struct {
	Status Status
	Action string
}

func (e *StateError) Error() string {
	return fmt.Sprintf("任务状态 %s 不允许执行%s", e.Status, e.Action)
}

type ConflictError struct {
	Expected int64
	Actual   int64
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("修订冲突：期望 %d，实际 %d", e.Expected, e.Actual)
}

type NotFoundError struct {
	ID string
}

func (e *NotFoundError) Error() string { return fmt.Sprintf("未找到任务 %s", e.ID) }

type DuplicateCaseError struct {
	TreeCode string
	CaseID   string
	Status   Status
}

func (e *DuplicateCaseError) Error() string {
	return fmt.Sprintf("树木编号 %s 已存在未关闭任务 %s（%s）", e.TreeCode, e.CaseID, e.Status.Label())
}

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
