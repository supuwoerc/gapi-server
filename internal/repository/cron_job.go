package repository

import (
	"context"
	"time"

	"github.com/supuwoerc/gapi-server/internal/model"

	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CronJobRepository struct {
	db *gorm.DB
}

func NewCronJobRepository(db *gorm.DB) *CronJobRepository {
	return &CronJobRepository{db: db}
}

func (r *CronJobRepository) UpsertJob(ctx context.Context, job *model.CronJob) error {
	err := r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"interval", "description", "updated_at"}),
	}).Create(job).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CronJobRepository) FindByName(ctx context.Context, name string) (*model.CronJob, error) {
	var job model.CronJob
	err := r.db.WithContext(ctx).Where("name = ?", name).First(&job).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &job, nil
}

func (r *CronJobRepository) ListAll(ctx context.Context) ([]*model.CronJob, error) {
	var jobs []*model.CronJob
	err := r.db.WithContext(ctx).Order("id ASC").Find(&jobs).Error
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return jobs, nil
}

func (r *CronJobRepository) UpdateEnabled(ctx context.Context, name string, enabled bool) error {
	err := r.db.WithContext(ctx).Model(&model.CronJob{}).Where("name = ?", name).Update("enabled", enabled).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CronJobRepository) UpdateLastRun(ctx context.Context, name string, status string) error {
	now := time.Now()
	err := r.db.WithContext(ctx).Model(&model.CronJob{}).Where("name = ?", name).
		Updates(map[string]any{"last_run_at": now, "last_status": status}).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CronJobRepository) CreateExecution(ctx context.Context, exec *model.CronJobExecution) error {
	err := r.db.WithContext(ctx).Create(exec).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CronJobRepository) UpdateExecution(ctx context.Context, id uint, updates map[string]any) error {
	err := r.db.WithContext(ctx).Model(&model.CronJobExecution{}).Where("id = ?", id).Updates(updates).Error
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func (r *CronJobRepository) ListExecutions(ctx context.Context, jobName string, page, pageSize int) ([]*model.CronJobExecution, int64, error) {
	var executions []*model.CronJobExecution
	var total int64

	query := r.db.WithContext(ctx).Model(&model.CronJobExecution{}).Where("job_name = ?", jobName)
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, errors.WithStack(err)
	}

	offset := (page - 1) * pageSize
	err := query.Order("id DESC").Offset(offset).Limit(pageSize).Find(&executions).Error
	if err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return executions, total, nil
}
