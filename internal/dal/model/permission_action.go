package model

import "database/sql/driver"

// PermissionAction 权限的操作类型，存进 sys_permission.action
// （VARCHAR(32)，CHECK IN ('create','read','update','delete')）。
// 常量值即落库值，改拼写要同步 migrations 里的 CHECK 与 seed SQL。
type PermissionAction string

const (
	PermissionActionCreate PermissionAction = "create"
	PermissionActionRead   PermissionAction = "read"
	PermissionActionUpdate PermissionAction = "update"
	PermissionActionDelete PermissionAction = "delete"
)

func (i *PermissionAction) Scan(src any) error          { return enumStringFromDB(i, src) }
func (i PermissionAction) Value() (driver.Value, error) { return enumStringIntoDB(i) }
func (i PermissionAction) Text() string                 { return enumStringText(i) }
