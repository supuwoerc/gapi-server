package model

import "database/sql/driver"

// PermissionEffect 角色权限的效果，存进 sys_role_permission.effect
// （VARCHAR(8)，CHECK IN ('allow','deny')，默认 allow）。
// 常量值即落库值，改拼写要同步 migrations 里的 CHECK 与 seed SQL。
type PermissionEffect string

const (
	PermissionEffectAllow PermissionEffect = "allow"
	PermissionEffectDeny  PermissionEffect = "deny"
)

func (i *PermissionEffect) Scan(src any) error          { return enumStringFromDB(i, src) }
func (i PermissionEffect) Value() (driver.Value, error) { return enumStringIntoDB(i) }
func (i PermissionEffect) Text() string                 { return enumStringText(i) }
