package tasks

import (
	"context"
	"time"
)

const (
	maxFiles    = 1500
	maxDepth    = 8
	maxTasks    = 80
	runTimeout  = 2 * time.Minute
	outputLimit = 24 * 1024
)

type Service struct{}

func New() *Service {
	return &Service{}
}

func (s *Service) Discover(root string) (Summary, error) {
	return discover(root)
}

func (s *Service) Find(root string, taskID string) (Task, bool, error) {
	summary, err := discover(root)
	if err != nil {
		return Task{}, false, err
	}
	for _, task := range summary.Tasks {
		if task.ID == taskID {
			return task, true, nil
		}
	}
	return Task{}, false, nil
}

func (s *Service) Run(root string, taskID string) (RunResult, error) {
	return runDiscovered(context.Background(), root, taskID)
}

func (s *Service) RunContext(ctx context.Context, root string, taskID string) (RunResult, error) {
	return runDiscovered(ctx, root, taskID)
}
