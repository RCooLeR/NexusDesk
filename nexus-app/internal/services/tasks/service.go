package tasks

import "time"

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
	return runDiscovered(root, taskID)
}
