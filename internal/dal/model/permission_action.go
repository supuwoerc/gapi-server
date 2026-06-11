package model

import (
	"database/sql/driver"
	"fmt"
)

type PermissionAction string

func (i *PermissionAction) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*i = PermissionAction(v)
	case []byte:
		*i = PermissionAction(v)
	default:
		return fmt.Errorf("cannot scan %T into PermissionAction", src)
	}
	return nil
}

func (i PermissionAction) Value() (driver.Value, error) {
	return string(i), nil
}

const (
	PermissionActionCreate PermissionAction = "create"
	PermissionActionRead   PermissionAction = "read"
	PermissionActionUpdate PermissionAction = "update"
	PermissionActionDelete PermissionAction = "delete"
)
