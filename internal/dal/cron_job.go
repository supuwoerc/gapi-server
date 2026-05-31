package dal

import (
	"context"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"
	"github.com/supuwoerc/gapi-server/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CronJobDal struct {
	DB *gorm.DB
}

func (d *CronJobDal) getQuery(ctx context.Context) *query.Query {
	return query.Use(database.TxFromContext(ctx, d.DB))
}

func (d *CronJobDal) UpsertJob(ctx context.Context, job *model.CronJob) error {
	return d.DB.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"interval", "description", "updated_at"}),
	}).Create(job).Error
}

func (d *CronJobDal) FindByName(ctx context.Context, name string) (*model.CronJob, error) {
	q := d.getQuery(ctx).CronJob
	return q.WithContext(ctx).Where(q.Name.Eq(name)).First()
}

func (d *CronJobDal) ListAll(ctx context.Context) ([]*model.CronJob, error) {
	q := d.getQuery(ctx).CronJob
	return q.WithContext(ctx).Order(q.ID).Find()
}

func (d *CronJobDal) UpdateEnabled(ctx context.Context, name string, enabled bool) error {
	q := d.getQuery(ctx).CronJob
	_, err := q.WithContext(ctx).Where(q.Name.Eq(name)).Update(q.Enabled, enabled)
	return err
}

func (d *CronJobDal) UpdateLastRun(ctx context.Context, name string, status string) error {
	q := d.getQuery(ctx).CronJob
	now := time.Now()
	_, err := q.WithContext(ctx).Where(q.Name.Eq(name)).UpdateColumns(map[string]any{
		"last_run_at": now,
		"last_status": status,
	})
	return err
}

func (d *CronJobDal) CreateExecution(ctx context.Context, exec *model.CronJobExecution) error {
	q := d.getQuery(ctx).CronJobExecution
	return q.WithContext(ctx).Create(exec)
}

func (d *CronJobDal) UpdateExecution(ctx context.Context, id uint64, updates map[string]any) error {
	q := d.getQuery(ctx).CronJobExecution
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateColumns(updates)
	return err
}

func (d *CronJobDal) ListExecutions(ctx context.Context, jobName string, page, pageSize int) ([]*model.CronJobExecution, int64, error) {
	q := d.getQuery(ctx).CronJobExecution
	offset := (page - 1) * pageSize
	return q.WithContext(ctx).Where(q.JobName.Eq(jobName)).Order(q.ID.Desc()).FindByPage(offset, pageSize)
}
