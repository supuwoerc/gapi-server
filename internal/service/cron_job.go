package service

import (
	"context"
	"errors"
	"time"

	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type CronJobRepository interface {
	UpsertJob(ctx context.Context, job *model.CronJob) error
	FindByName(ctx context.Context, name string) (*model.CronJob, error)
	ListAll(ctx context.Context) ([]*model.CronJob, error)
	UpdateEnabled(ctx context.Context, name string, enabled bool) error
	UpdateLastRun(ctx context.Context, name string, status string) error
	CreateExecution(ctx context.Context, exec *model.CronJobExecution) error
	FinishExecution(ctx context.Context, id uint64, status string, endedAt time.Time, errMsg string) error
	ListExecutions(ctx context.Context, jobName string, page, pageSize int) ([]*model.CronJobExecution, int64, error)
}

type CronJobService struct {
	Repo   CronJobRepository
	Logger *logger.Logger
}

func (s *CronJobService) SyncJobDefinitions(ctx context.Context, jobs []cronjob.SystemJob) error {
	for _, j := range jobs {
		job := &model.CronJob{
			Name:     j.Name(),
			Interval: j.Interval(),
		}
		if err := s.Repo.UpsertJob(ctx, job); err != nil {
			s.Logger.Ctx(ctx).Error("failed to upsert job", zap.String("job", j.Name()), zap.Error(err))
			return err
		}
	}
	return nil
}

func (s *CronJobService) IsJobEnabled(ctx context.Context, name string) (bool, error) {
	job, err := s.Repo.FindByName(ctx, name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return true, nil
		}
		s.Logger.Ctx(ctx).Error("failed to find job", zap.String("name", name), zap.Error(err))
		return false, err
	}
	return job.Enabled, nil
}

func (s *CronJobService) SetEnabled(ctx context.Context, name string, enabled bool) error {
	if err := s.Repo.UpdateEnabled(ctx, name, enabled); err != nil {
		s.Logger.Ctx(ctx).Error("failed to update job enabled", zap.String("name", name), zap.Bool("enabled", enabled), zap.Error(err))
		return err
	}
	return nil
}

func (s *CronJobService) RecordStart(ctx context.Context, jobName string, triggeredBy model.TriggeredBy) (uint64, error) {
	exec := &model.CronJobExecution{
		JobName:     jobName,
		Status:      cronjob.StatusRunning,
		StartedAt:   time.Now(),
		TriggeredBy: triggeredBy,
	}
	if err := s.Repo.CreateExecution(ctx, exec); err != nil {
		s.Logger.Ctx(ctx).Error("failed to record execution start", zap.String("job", jobName), zap.Error(err))
		return 0, err
	}
	return exec.ID, nil
}

func (s *CronJobService) RecordEnd(ctx context.Context, executionID uint64, status string, jobErr error) error {
	errMsg := ""
	if jobErr != nil {
		errMsg = jobErr.Error()
	}
	if err := s.Repo.FinishExecution(ctx, executionID, status, time.Now(), errMsg); err != nil {
		s.Logger.Ctx(ctx).Error("failed to record execution end", zap.Uint64("executionID", executionID), zap.Error(err))
		return err
	}
	return nil
}

func (s *CronJobService) UpdateLastRun(ctx context.Context, name string, status string) error {
	if err := s.Repo.UpdateLastRun(ctx, name, status); err != nil {
		s.Logger.Ctx(ctx).Error("failed to update last run", zap.String("name", name), zap.Error(err))
		return err
	}
	return nil
}

func (s *CronJobService) ListJobs(ctx context.Context) ([]*model.CronJob, error) {
	jobs, err := s.Repo.ListAll(ctx)
	if err != nil {
		s.Logger.Ctx(ctx).Error("failed to list jobs", zap.Error(err))
		return nil, err
	}
	return jobs, nil
}

func (s *CronJobService) GetJob(ctx context.Context, name string) (*model.CronJob, error) {
	job, err := s.Repo.FindByName(ctx, name)
	if err != nil {
		s.Logger.Ctx(ctx).Error("failed to get job", zap.String("name", name), zap.Error(err))
		return nil, err
	}
	return job, nil
}

func (s *CronJobService) ListExecutions(ctx context.Context, jobName string, page, pageSize int) ([]*model.CronJobExecution, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	executions, total, err := s.Repo.ListExecutions(ctx, jobName, page, pageSize)
	if err != nil {
		s.Logger.Ctx(ctx).Error("failed to list executions", zap.String("job", jobName), zap.Error(err))
		return nil, 0, err
	}
	return executions, total, nil
}
