package app

import (
	"context"
	"fmt"

	"github.com/cn8001/reforgepulse/internal/domain"
	"github.com/cn8001/reforgepulse/internal/platform"
	"github.com/cn8001/reforgepulse/internal/tasks"
)

type ProgramFinder interface {
	List(ctx context.Context) ([]domain.InstalledProgram, error)
	Find(ctx context.Context, id string) (platform.Program, error)
}

type Uninstaller interface {
	Uninstall(ctx context.Context, uninstallString string) error
}

// ProgramService lists installed programs with how long they sat unused,
// and runs a program's own uninstaller when the user asks.
type ProgramService struct {
	finder  ProgramFinder
	un      Uninstaller
	runner  TaskRunner
	journal Recorder
}

// ProgramDeps are the program service's dependencies.
type ProgramDeps struct {
	Finder      ProgramFinder
	Uninstaller Uninstaller
	Runner      TaskRunner
	Journal     Recorder
}

func NewProgramService(d ProgramDeps) *ProgramService {
	return &ProgramService{finder: d.Finder, un: d.Uninstaller, runner: d.Runner, journal: d.Journal}
}

func (s *ProgramService) List(ctx context.Context) ([]domain.InstalledProgram, error) {
	var out []domain.InstalledProgram
	err := s.runner.Do(ctx, "Reading installed programs", func(ctx context.Context, _ tasks.Report) error {
		var err error
		out, err = s.finder.List(ctx)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("list programs: %w", err)
	}
	return out, nil
}

// Uninstall runs the program's registered uninstaller. It is looked up here
// by ID, never taken from the window, and the change log records it.
func (s *ProgramService) Uninstall(ctx context.Context, id string) error {
	p, err := s.finder.Find(ctx, id)
	if err != nil {
		return err
	}
	if p.UninstallString == "" {
		return fmt.Errorf("%s has no registered uninstaller", p.Name)
	}
	return s.runner.Do(ctx, "Uninstalling "+p.Name, func(ctx context.Context, _ tasks.Report) error {
		cid, err := s.journal.Begin(ctx, domain.Change{
			Action: "uninstall", Summary: "Uninstall " + p.Name, Target: p.UninstallString, Before: "version " + p.Version,
		})
		if err != nil {
			return fmt.Errorf("log uninstall: %w", err)
		}
		err = s.un.Uninstall(ctx, p.UninstallString)
		// The uninstaller may be cancelled by the user; check what is left.
		after := "uninstaller finished"
		if _, ferr := s.finder.Find(ctx, id); ferr == nil {
			after = "still installed after the uninstaller closed"
		}
		if ferr := s.journal.Finish(ctx, cid, after, err); ferr != nil && err == nil {
			err = ferr
		}
		return err
	})
}
