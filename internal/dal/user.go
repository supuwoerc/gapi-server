package dal

import (
	"context"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"
	"github.com/supuwoerc/gapi-server/pkg/database"

	"gorm.io/gorm"
)

type UserDal struct {
	DB *gorm.DB
}

func (d *UserDal) getQuery(ctx context.Context) *query.Query {
	return query.Use(database.TxFromContext(ctx, d.DB))
}

func (d *UserDal) Create(ctx context.Context, user *model.User) error {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Create(user)
}

func (d *UserDal) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Where(q.Email.Eq(email)).First()
}

func (d *UserDal) FindByEmailWithRoles(ctx context.Context, email string) (*model.User, error) {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Preload(q.Roles).Where(q.Email.Eq(email)).First()
}

func (d *UserDal) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Where(q.Username.Eq(username)).First()
}

func (d *UserDal) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Where(q.ID.Eq(id)).First()
}

func (d *UserDal) FindByIDWithRoles(ctx context.Context, id uint64) (*model.User, error) {
	q := d.getQuery(ctx).User
	return q.WithContext(ctx).Preload(q.Roles).Where(q.ID.Eq(id)).First()
}

func (d *UserDal) UpdateLastLogin(ctx context.Context, id uint64) error {
	q := d.getQuery(ctx).User
	now := time.Now()
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.LastLoginAt.Value(now),
		q.LoginFailCount.Value(0),
	)
	return err
}

func (d *UserDal) IncrementLoginFail(ctx context.Context, id uint64) error {
	db := database.TxFromContext(ctx, d.DB)
	return db.WithContext(ctx).Model(&model.User{}).
		Where("id = ?", id).
		Update("login_fail_count", gorm.Expr("login_fail_count + 1")).Error
}

func (d *UserDal) LockUser(ctx context.Context, id uint64, until time.Time) error {
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.LockedUntil.Value(until),
	)
	return err
}
