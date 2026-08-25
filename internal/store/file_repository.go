package store

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec/internal/domain"
)

type FileRepository struct {
	mu       sync.RWMutex
	root     string
	snapDir  string
	auditDir string
	cases    map[string]*domain.CareCase
	requests map[string]string
}

func Open(root string) (*FileRepository, error) {
	if strings.TrimSpace(root) == "" {
		return nil, errors.New("存储目录不能为空")
	}
	r := &FileRepository{
		root: root, snapDir: filepath.Join(root, "snapshots"), auditDir: filepath.Join(root, "audit"),
		cases: make(map[string]*domain.CareCase), requests: make(map[string]string),
	}
	if err := os.MkdirAll(r.snapDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建快照目录: %w", err)
	}
	if err := os.MkdirAll(r.auditDir, 0o750); err != nil {
		return nil, fmt.Errorf("创建审计目录: %w", err)
	}
	if err := r.load(); err != nil {
		return nil, err
	}
	return r, nil
}

func (r *FileRepository) load() error {
	entries, err := os.ReadDir(r.snapDir)
	if err != nil {
		return fmt.Errorf("读取快照目录: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(r.snapDir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("读取快照 %s: %w", entry.Name(), err)
		}
		var c domain.CareCase
		if err := json.Unmarshal(data, &c); err != nil {
			return fmt.Errorf("解析快照 %s: %w", entry.Name(), err)
		}
		if err := validateSnapshot(&c); err != nil {
			return fmt.Errorf("校验快照 %s: %w", entry.Name(), err)
		}
		r.cases[c.ID] = &c
		for _, event := range c.AuditEvents {
			if event.RequestID == "" {
				continue
			}
			if owner, exists := r.requests[event.RequestID]; exists && owner != c.ID {
				return fmt.Errorf("request_id %s 同时属于任务 %s 和 %s", event.RequestID, owner, c.ID)
			}
			r.requests[event.RequestID] = c.ID
		}
		if err := r.reconcileAudit(&c); err != nil {
			return err
		}
	}
	return nil
}

func validateSnapshot(c *domain.CareCase) error {
	if c.ID == "" || c.Revision < 1 || len(c.AuditEvents) == 0 {
		return errors.New("任务标识、修订号或审计事件缺失")
	}
	var previous int64
	for _, event := range c.AuditEvents {
		if event.CaseID != c.ID || event.Revision <= previous || event.Revision > c.Revision {
			return errors.New("审计事件的任务或修订序列无效")
		}
		previous = event.Revision
	}
	if previous != c.Revision {
		return errors.New("审计事件末尾修订号与快照不一致")
	}
	return nil
}

func (r *FileRepository) reconcileAudit(c *domain.CareCase) error {
	path := r.auditPath(c.ID)
	events, err := readAudit(path)
	if err == nil && auditEqual(events, c.AuditEvents) {
		return nil
	}
	if err := writeAuditAtomically(path, c.AuditEvents); err != nil {
		return fmt.Errorf("修复任务 %s 审计日志: %w", c.ID, err)
	}
	return nil
}

func (r *FileRepository) Get(_ context.Context, id string) (*domain.CareCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.cases[id]
	if !ok {
		return nil, &domain.NotFoundError{ID: id}
	}
	return cloneCase(c)
}

func (r *FileRepository) List(_ context.Context) ([]*domain.CareCase, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*domain.CareCase, 0, len(r.cases))
	for _, c := range r.cases {
		copy, err := cloneCase(c)
		if err != nil {
			return nil, err
		}
		result = append(result, copy)
	}
	sort.SliceStable(result, func(i, j int) bool { return result[i].UpdatedAt.After(result[j].UpdatedAt) })
	return result, nil
}

func (r *FileRepository) LookupRequest(_ context.Context, requestID string) (*domain.CareCase, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.requests[requestID]
	if !ok {
		return nil, false, nil
	}
	c, ok := r.cases[id]
	if !ok {
		return nil, false, fmt.Errorf("幂等索引指向不存在的任务 %s", id)
	}
	copy, err := cloneCase(c)
	return copy, true, err
}

func (r *FileRepository) LookupActiveTreeCode(_ context.Context, treeCode string) (*domain.CareCase, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	code := strings.TrimSpace(treeCode)
	for _, c := range r.cases {
		if c.Status != domain.StatusClosed && strings.TrimSpace(c.TreeCode) == code {
			copy, err := cloneCase(c)
			return copy, true, err
		}
	}
	return nil, false, nil
}

func (r *FileRepository) Commit(_ context.Context, next *domain.CareCase, expected int64, requestID string) (*domain.CareCase, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if owner, ok := r.requests[requestID]; ok {
		if owner != next.ID {
			return nil, false, &domain.ValidationError{Field: "request_id", Message: "request_id 已用于其他任务"}
		}
		copy, err := cloneCase(r.cases[owner])
		return copy, true, err
	}
	current, exists := r.cases[next.ID]
	if !exists && expected == 0 {
		for _, c := range r.cases {
			if c.Status != domain.StatusClosed && strings.TrimSpace(c.TreeCode) == strings.TrimSpace(next.TreeCode) {
				return nil, false, &domain.DuplicateCaseError{TreeCode: strings.TrimSpace(next.TreeCode), CaseID: c.ID, Status: c.Status}
			}
		}
	}
	actual := int64(0)
	if exists {
		actual = current.Revision
	}
	if actual != expected {
		return nil, false, &domain.ConflictError{Expected: expected, Actual: actual}
	}
	if err := validateSnapshot(next); err != nil {
		return nil, false, fmt.Errorf("拒绝无效快照: %w", err)
	}
	if next.Revision <= expected {
		return nil, false, errors.New("新快照必须推进修订号")
	}
	if !strings.HasPrefix(filepath.Base(next.ID), "case-") || filepath.Base(next.ID) != next.ID {
		return nil, false, errors.New("任务 ID 不适合作为持久化文件名")
	}
	newEvents := eventsAfter(next.AuditEvents, expected)
	if len(newEvents) == 0 {
		return nil, false, errors.New("提交没有新增审计事件")
	}
	requestRecorded := false
	for _, event := range newEvents {
		if event.RequestID == requestID {
			requestRecorded = true
		}
	}
	if !requestRecorded {
		return nil, false, errors.New("新增审计事件未记录本次 request_id")
	}
	auditPath := r.auditPath(next.ID)
	previousSize, err := appendAudit(auditPath, newEvents)
	if err != nil {
		return nil, false, fmt.Errorf("追加审计日志: %w", err)
	}
	for _, event := range newEvents {
		if event.RequestID != "" {
			r.requests[event.RequestID] = next.ID
		}
	}
	if err := writeSnapshotAtomically(r.snapshotPath(next.ID), next); err != nil {
		if rollbackErr := truncateAudit(auditPath, previousSize); rollbackErr != nil {
			return nil, false, fmt.Errorf("保存快照失败: %v；回滚审计日志失败: %w", err, rollbackErr)
		}
		return nil, false, fmt.Errorf("保存快照: %w", err)
	}
	copy, err := cloneCase(next)
	if err != nil {
		return nil, false, err
	}
	r.cases[next.ID] = copy
	result, err := cloneCase(copy)
	return result, false, err
}

func (r *FileRepository) snapshotPath(id string) string { return filepath.Join(r.snapDir, id+".json") }
func (r *FileRepository) auditPath(id string) string    { return filepath.Join(r.auditDir, id+".jsonl") }

func eventsAfter(events []domain.AuditEvent, revision int64) []domain.AuditEvent {
	result := make([]domain.AuditEvent, 0)
	for _, event := range events {
		if event.Revision > revision {
			result = append(result, event)
		}
	}
	return result
}

func writeSnapshotAtomically(path string, c *domain.CareCase) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(path, data)
}

func atomicWrite(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	ok := false
	defer func() {
		if !ok {
			_ = os.Remove(tmpName)
		}
	}()
	if err := tmp.Chmod(0o640); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		return err
	}
	ok = true
	return syncDir(dir)
}

func appendAudit(path string, events []domain.AuditEvent) (int64, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640)
	if err != nil {
		return 0, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return 0, err
	}
	previous := info.Size()
	fail := func(cause error) (int64, error) {
		_ = f.Truncate(previous)
		_ = f.Sync()
		_ = f.Close()
		return previous, cause
	}
	encoder := json.NewEncoder(f)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return fail(err)
		}
	}
	if err := f.Sync(); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		return previous, err
	}
	return previous, syncDir(filepath.Dir(path))
}

func truncateAudit(path string, size int64) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	if err := f.Truncate(size); err != nil {
		_ = f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}

func writeAuditAtomically(path string, events []domain.AuditEvent) error {
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	for _, event := range events {
		if err := encoder.Encode(event); err != nil {
			return err
		}
	}
	return atomicWrite(path, buffer.Bytes())
}

func readAudit(path string) ([]domain.AuditEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	result := make([]domain.AuditEvent, 0)
	reader := bufio.NewReader(f)
	decoder := json.NewDecoder(reader)
	for {
		var event domain.AuditEvent
		if err := decoder.Decode(&event); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		result = append(result, event)
	}
	return result, nil
}

func auditEqual(a, b []domain.AuditEvent) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		left, _ := json.Marshal(a[i])
		right, _ := json.Marshal(b[i])
		if !bytes.Equal(left, right) {
			return false
		}
	}
	return true
}

func syncDir(path string) error {
	d, err := os.Open(path)
	if err != nil {
		return err
	}
	defer d.Close()
	return d.Sync()
}
