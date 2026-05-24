package service

import (
	"context"
	"errors"
	"time"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/model"
	"github.com/supuwoerc/gapi-server/internal/repository"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"gorm.io/gorm"
)

type CronJobService struct {
	repo   *repository.CronJobRepository
	logger *logger.Logger
}

// Ensure CronJobService implements cronjob.JobRecorder
var _ cronjob.JobRecorder = (*CronJobService)(nil)

func NewCronJobService(repo *repository.CronJobRepository, logger *logger.Logger) *CronJobService {
	return &CronJobService{repo: repo, logger: logger}
}

func (s *CronJobService) SyncJobDefinitions(ctx context.Context, jobs []cronjob.SystemJob) error {
	for _, j := range jobs {
		job := &model.CronJob{
			Name:     j.Name(),
			Interval: j.Interval(),
		}
		if err := s.repo.UpsertJob(ctx, job); err != nil {
			return err
		}
	}
	return nil
}

func (s *CronJobService) IsJobEnabled(ctx context.Context, name string) (bool, error) {
	job, err := s.repo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		return false, err
	}
	return job.Enabled, nil
}

func (s *CronJobService) SetEnabled(ctx context.Context, name string, enabled bool) error {
	return s.repo.UpdateEnabled(ctx, name, enabled)
}

func (s *CronJobService) RecordStart(ctx context.Context, jobName, triggeredBy string) (uint, error) {
	exec := &model.CronJobExecution{
		JobName:     jobName,
		Status:      cronjob.StatusRunning,
		StartedAt:   time.Now(),
		TriggeredBy: triggeredBy,
	}
	if err := s.repo.CreateExecution(ctx, exec); err != nil {
		return 0, err
	}
	return exec.ID, nil
}

func (s *CronJobService) RecordEnd(ctx context.Context, executionID uint, status string, jobErr error) error {
	now := time.Now()
	updates := map[string]any{
		"status":   status,
		"ended_at": now,
	}
	if jobErr != nil {
		updates["error"] = jobErr.Error()
	}
	return s.repo.UpdateExecution(ctx, executionID, updates)
}

func (s *CronJobService) UpdateLastRun(ctx context.Context, name string, status string) error {
	return s.repo.UpdateLastRun(ctx, name, status)
}

func (s *CronJobService) ListJobs(ctx context.Context) ([]*model.CronJob, error) {
	return s.repo.ListAll(ctx)
}

func (s *CronJobService) GetJob(ctx context.Context, name string) (*model.CronJob, error) {
	return s.repo.FindByName(ctx, name)
}

func (s *CronJobService) ListExecutions(ctx context.Context, jobName string, page, pageSize int) ([]*model.CronJobExecution, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	return s.repo.ListExecutions(ctx, jobName, page, pageSize)
}

func (s *CronJobService) Logger() *logger.Logger {
	return s.logger
}
