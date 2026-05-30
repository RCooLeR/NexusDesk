package jobs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxLogLines             = 64
	defaultFullLogReadBytes = 256 * 1024
)

type Service struct {
	mu               sync.Mutex
	nextID           int
	jobs             []Job
	cancels          map[string]context.CancelFunc
	repo             Repository
	logRoot          string
	persistenceIssue PersistenceIssue
}

type Repository interface {
	SaveJob(Job) error
	ListJobs() ([]Job, error)
	DeleteJobs([]string) error
}

func New() *Service {
	return &Service{cancels: map[string]context.CancelFunc{}}
}

func NewWithRepository(repo Repository) *Service {
	service := New()
	service.SetRepository(repo, true)
	return service
}

func (s *Service) SetRepository(repo Repository, load bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo = repo
	if !load || repo == nil {
		return
	}
	jobs, err := repo.ListJobs()
	if err != nil {
		return
	}
	s.jobs = jobs
	s.nextID = nextIDFromJobs(jobs)
}

func (s *Service) SetLogRoot(workspaceRoot string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	workspaceRoot = strings.TrimSpace(workspaceRoot)
	if workspaceRoot == "" {
		s.logRoot = ""
		return nil
	}
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return err
	}
	logRoot := filepath.Join(absRoot, ".nexusdesk", "jobs")
	if err := os.MkdirAll(logRoot, 0o755); err != nil {
		return err
	}
	s.logRoot = logRoot
	for index := range s.jobs {
		s.jobs[index].LogPath = s.fullLogPathLocked(s.jobs[index].ID)
	}
	return nil
}

func (s *Service) Start(kind string, label string) (Job, context.Context) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kind = NormalizeKind(kind)
	s.nextID++
	ctx, cancel := context.WithCancel(context.Background())
	job := Job{
		ID:        fmt.Sprintf("job-%04d", s.nextID),
		Kind:      kind,
		Label:     label,
		Status:    StatusRunning,
		Message:   "Running " + label + ".",
		StartedAt: time.Now().UTC(),
	}
	job.LogPath = s.fullLogPathLocked(job.ID)
	s.jobs = append([]Job{job}, s.jobs...)
	s.cancels[job.ID] = cancel
	s.persistLocked(job)
	return job, ctx
}

func (s *Service) StartWorkflow(kind string, label string, options StartOptions) (Job, context.Context, error) {
	if err := ValidateWorkflowStart(kind, options); err != nil {
		return Job{}, nil, err
	}
	job, ctx := s.Start(kind, label)
	return job, ctx, nil
}

func (s *Service) AppendLog(id string, line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOf(id)
	if index < 0 || line == "" {
		return
	}
	s.jobs[index].LogTail = append(s.jobs[index].LogTail, line)
	if len(s.jobs[index].LogTail) > maxLogLines {
		s.jobs[index].LogTail = s.jobs[index].LogTail[len(s.jobs[index].LogTail)-maxLogLines:]
	}
	s.persistLocked(s.jobs[index])
	s.appendFullLogLocked(s.jobs[index].ID, line)
}

func (s *Service) Finish(id string, status Status, message string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOf(id)
	if index < 0 {
		return
	}
	if s.jobs[index].Status == StatusCanceled && status != StatusCanceled {
		delete(s.cancels, id)
		return
	}
	s.jobs[index].Status = status
	s.jobs[index].Message = message
	if err != nil {
		s.jobs[index].Error = err.Error()
	}
	s.jobs[index].CompletedAt = time.Now().UTC()
	delete(s.cancels, id)
	s.persistLocked(s.jobs[index])
}

func (s *Service) Cancel(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOf(id)
	if index < 0 || s.jobs[index].Status != StatusRunning {
		return false
	}
	cancel := s.cancels[id]
	if cancel != nil {
		cancel()
	}
	s.jobs[index].Status = StatusCanceled
	s.jobs[index].Message = "Cancel requested."
	s.jobs[index].CompletedAt = time.Now().UTC()
	delete(s.cancels, id)
	s.persistLocked(s.jobs[index])
	return true
}

func (s *Service) List() []Job {
	s.mu.Lock()
	defer s.mu.Unlock()
	jobs := make([]Job, len(s.jobs))
	copy(jobs, s.jobs)
	for index := range jobs {
		jobs[index].LogTail = append([]string(nil), jobs[index].LogTail...)
		jobs[index].LogPath = s.fullLogPathLocked(jobs[index].ID)
	}
	return jobs
}

func (s *Service) Get(id string) (Job, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	index := s.indexOf(id)
	if index < 0 {
		return Job{}, false
	}
	job := s.jobs[index]
	job.LogTail = append([]string(nil), job.LogTail...)
	job.LogPath = s.fullLogPathLocked(job.ID)
	return job, true
}

func (s *Service) FullLogPath(id string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.indexOf(id) < 0 {
		return "", false
	}
	path := s.fullLogPathLocked(id)
	return path, path != ""
}

func (s *Service) ReadFullLog(id string, maxBytes int) (string, string, error) {
	s.mu.Lock()
	if s.indexOf(id) < 0 {
		s.mu.Unlock()
		return "", "", fmt.Errorf("job %q was not found", id)
	}
	path := s.fullLogPathLocked(id)
	s.mu.Unlock()
	if path == "" {
		return "", "", nil
	}
	text, err := readTailFile(path, maxBytes)
	if os.IsNotExist(err) {
		return "", path, nil
	}
	return text, path, err
}

func (s *Service) PersistenceIssue() (PersistenceIssue, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.persistenceIssue.Error == "" {
		return PersistenceIssue{}, false
	}
	return s.persistenceIssue, true
}

func (s *Service) Prune(policy RetentionPolicy) (RetentionResult, error) {
	policy = normalizeRetentionPolicy(policy)
	s.mu.Lock()
	defer s.mu.Unlock()

	candidates := make([]Job, 0, len(s.jobs))
	result := RetentionResult{}
	for _, job := range s.jobs {
		switch {
		case job.Status == StatusRunning:
			result.RunningKept++
		case isFailureStatus(job.Status) && !policy.IncludeFailures:
			result.FailuresKept++
		case isTerminalStatus(job.Status):
			candidates = append(candidates, job)
		default:
			result.Kept++
		}
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return jobRetentionTime(candidates[i]).After(jobRetentionTime(candidates[j]))
	})

	pruneIDs := map[string]bool{}
	for index, job := range candidates {
		keepByRecent := policy.KeepRecent > 0 && index < policy.KeepRecent
		olderThanMaxAge := policy.MaxAge > 0 && policy.Now.Sub(jobRetentionTime(job)) > policy.MaxAge
		if keepByRecent && !olderThanMaxAge {
			result.Kept++
			continue
		}
		if !olderThanMaxAge && policy.KeepRecent <= 0 {
			result.Kept++
			continue
		}
		if !olderThanMaxAge && index < policy.KeepRecent {
			result.Kept++
			continue
		}
		pruneIDs[job.ID] = true
		result.RepositoryIDs = append(result.RepositoryIDs, job.ID)
	}
	if len(pruneIDs) == 0 {
		return result, nil
	}
	if s.repo != nil {
		if err := s.repo.DeleteJobs(result.RepositoryIDs); err != nil {
			return result, err
		}
	}
	filtered := s.jobs[:0]
	for _, job := range s.jobs {
		if pruneIDs[job.ID] {
			delete(s.cancels, job.ID)
			continue
		}
		filtered = append(filtered, job)
	}
	s.jobs = filtered
	result.Removed = len(pruneIDs)
	return result, nil
}

func (s *Service) indexOf(id string) int {
	for index := range s.jobs {
		if s.jobs[index].ID == id {
			return index
		}
	}
	return -1
}

func (s *Service) persistLocked(job Job) {
	if s.repo == nil {
		return
	}
	if err := s.repo.SaveJob(job); err != nil {
		s.persistenceIssue = PersistenceIssue{
			JobID:     job.ID,
			Operation: "save job",
			Error:     err.Error(),
			At:        time.Now().UTC(),
		}
		return
	}
	s.persistenceIssue = PersistenceIssue{}
}

func (s *Service) appendFullLogLocked(id string, line string) {
	path := s.fullLogPathLocked(id)
	if path == "" || strings.TrimSpace(line) == "" {
		return
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		s.recordPersistenceIssueLocked(id, "create job log directory", err)
		return
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		s.recordPersistenceIssueLocked(id, "append job log", err)
		return
	}
	_, writeErr := fmt.Fprintf(file, "%s\t%s\n", time.Now().UTC().Format(time.RFC3339Nano), line)
	closeErr := file.Close()
	if writeErr != nil {
		s.recordPersistenceIssueLocked(id, "append job log", writeErr)
		return
	}
	if closeErr != nil {
		s.recordPersistenceIssueLocked(id, "close job log", closeErr)
		return
	}
}

func (s *Service) fullLogPathLocked(id string) string {
	if s.logRoot == "" || !safeJobID(id) {
		return ""
	}
	return filepath.Join(s.logRoot, id, "job.log")
}

func (s *Service) recordPersistenceIssueLocked(id string, operation string, err error) {
	if err == nil {
		return
	}
	s.persistenceIssue = PersistenceIssue{
		JobID:     id,
		Operation: operation,
		Error:     err.Error(),
		At:        time.Now().UTC(),
	}
}

func safeJobID(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" || strings.Contains(id, "..") {
		return false
	}
	for _, r := range id {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			continue
		}
		return false
	}
	return true
}

func readTailFile(path string, maxBytes int) (string, error) {
	if maxBytes <= 0 {
		maxBytes = defaultFullLogReadBytes
	}
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	prefix := ""
	if info.Size() > int64(maxBytes) {
		if _, err := file.Seek(-int64(maxBytes), io.SeekEnd); err != nil {
			return "", err
		}
		prefix = "[log truncated]\n"
	}
	data, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}
	return prefix + string(data), nil
}

func DefaultRetentionPolicy() RetentionPolicy {
	return RetentionPolicy{
		KeepRecent:      100,
		MaxAge:          30 * 24 * time.Hour,
		IncludeFailures: false,
	}
}

func normalizeRetentionPolicy(policy RetentionPolicy) RetentionPolicy {
	defaults := DefaultRetentionPolicy()
	if policy.KeepRecent < 0 {
		policy.KeepRecent = 0
	}
	if policy.KeepRecent == 0 && policy.MaxAge == 0 {
		policy.KeepRecent = defaults.KeepRecent
		policy.MaxAge = defaults.MaxAge
	}
	if policy.Now.IsZero() {
		policy.Now = time.Now().UTC()
	} else {
		policy.Now = policy.Now.UTC()
	}
	return policy
}

func isTerminalStatus(status Status) bool {
	return status == StatusSuccess || status == StatusCanceled || status == StatusFailed || status == StatusTimedOut
}

func isFailureStatus(status Status) bool {
	return status == StatusFailed || status == StatusTimedOut
}

func jobRetentionTime(job Job) time.Time {
	if !job.CompletedAt.IsZero() {
		return job.CompletedAt.UTC()
	}
	if !job.StartedAt.IsZero() {
		return job.StartedAt.UTC()
	}
	return time.Time{}
}

func nextIDFromJobs(jobs []Job) int {
	maxID := 0
	for _, job := range jobs {
		var number int
		if _, err := fmt.Sscanf(job.ID, "job-%04d", &number); err == nil && number > maxID {
			maxID = number
		}
	}
	return maxID
}
