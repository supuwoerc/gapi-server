package dal

import (
	"context"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"
	"github.com/supuwoerc/gapi-server/pkg/database"

	"gorm.io/datatypes"
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
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.LoginFailCount.SetCol(q.LoginFailCount.Add(1)),
	)
	return err
}

func (d *UserDal) LockUser(ctx context.Context, id uint64, until time.Time) error {
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.LockedUntil.Value(until),
	)
	return err
}

func (d *UserDal) UpdateCompletedTours(ctx context.Context, id uint64, tours []string) error {
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.CompletedTours.Value(datatypes.JSONSlice[string](tours)),
	)
	return err
}

func (d *UserDal) UpdateProfile(ctx context.Context, id uint64, username, bio, avatar string) error {
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).UpdateSimple(
		q.Username.Value(username),
		q.Avatar.Value(avatar),
		q.Bio.Value(bio),
	)
	return err
}

func (d *UserDal) EnableUser(ctx context.Context, email string) error {
	q := d.getQuery(ctx).User
	_, err := q.WithContext(ctx).Where(q.Email.Eq(email)).UpdateSimple(
		q.Enabled.Value(true),
	)
	return err
}
