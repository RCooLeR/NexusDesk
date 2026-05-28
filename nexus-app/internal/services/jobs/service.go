package jobs

import (
	"context"
	"fmt"
	"sync"
	"time"
)

const maxLogLines = 12

type Service struct {
	mu      sync.Mutex
	nextID  int
	jobs    []Job
	cancels map[string]context.CancelFunc
	repo    Repository
}

type Repository interface {
	SaveJob(Job) error
	ListJobs() ([]Job, error)
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
	return job, true
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
	_ = s.repo.SaveJob(job)
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
