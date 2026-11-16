package app

import (
	"context"

	"github.com/cn8001/reforgepulse/internal/tasks"
)

// TaskRunner runs work as a tracked task, off the UI thread.
type TaskRunner interface {
	Do(ctx context.Context, title string, fn func(ctx context.Context, report tasks.Report) error) error
	Cancel(id string)
	List() []tasks.Status
}

// TaskService lets the status bar list and cancel running work.
type TaskService struct {
	runner TaskRunner
}

func NewTaskService(r TaskRunner) *TaskService {
	return &TaskService{runner: r}
}

func (s *TaskService) List() []tasks.Status { return s.runner.List() }

func (s *TaskService) Cancel(id string) { s.runner.Cancel(id) }
