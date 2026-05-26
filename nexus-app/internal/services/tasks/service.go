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

func (s *Service) Run(root string, taskID string) (RunResult, error) {
	return runDiscovered(context.Background(), root, taskID)
}

func (s *Service) RunContext(ctx context.Context, root string, taskID string) (RunResult, error) {
	return runDiscovered(ctx, root, taskID)
}
