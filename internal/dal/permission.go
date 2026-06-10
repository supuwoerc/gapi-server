package dal

import (
	"context"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"
	"github.com/supuwoerc/gapi-server/pkg/database"

	"gorm.io/gorm"
)

type PermissionDal struct {
	DB *gorm.DB
}

func (d *PermissionDal) getQuery(ctx context.Context) *query.Query {
	return query.Use(database.TxFromContext(ctx, d.DB))
}

type codeWithEffect struct {
	Code   string
	Effect string
}

func resolvePermissions(rows []codeWithEffect) []string {
	allowSet := make(map[string]struct{})
	denySet := make(map[string]struct{})
	for _, row := range rows {
		switch row.Effect {
		case string(model.PermissionEffectDeny):
			denySet[row.Code] = struct{}{}
		default:
			allowSet[row.Code] = struct{}{}
		}
	}
	result := make([]string, 0, len(allowSet))
	for code := range allowSet {
		if _, denied := denySet[code]; !denied {
			result = append(result, code)
		}
	}
	return result
}

func (d *PermissionDal) FindCodesByRoleIDsAndResourceType(ctx context.Context, roleIDs []uint64, resourceType model.ResourceType) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	q := d.getQuery(ctx)
	rp := q.RolePermission
	p := q.Permission

	var rows []codeWithEffect
	err := p.WithContext(ctx).
		Select(p.Code, rp.Effect).
		Join(rp, rp.PermissionID.EqCol(p.ID)).
		Where(rp.RoleID.In(roleIDs...)).
		Where(p.ResourceType.Eq(int32(resourceType))).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	return resolvePermissions(rows), nil
}

func (d *PermissionDal) FindCodesByRoleIDsAndModule(ctx context.Context, roleIDs []uint64, module string) ([]string, error) {
	if len(roleIDs) == 0 {
		return []string{}, nil
	}
	q := d.getQuery(ctx)
	rp := q.RolePermission
	p := q.Permission

	var rows []codeWithEffect
	err := p.WithContext(ctx).
		Select(p.Code, rp.Effect).
		Join(rp, rp.PermissionID.EqCol(p.ID)).
		Where(rp.RoleID.In(roleIDs...)).
		Where(p.Module.Eq(module)).
		Scan(&rows)
	if err != nil {
		return nil, err
	}
	return resolvePermissions(rows), nil
}
