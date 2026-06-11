package model

import (
	"database/sql/driver"
	"fmt"
)

type PermissionEffect string

func (i *PermissionEffect) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*i = PermissionEffect(v)
	case []byte:
		*i = PermissionEffect(v)
	default:
		return fmt.Errorf("cannot scan %T into PermissionEffect", src)
	}
	return nil
}

func (i PermissionEffect) Value() (driver.Value, error) {
	return string(i), nil
}

const (
	PermissionEffectAllow PermissionEffect = "allow"
	PermissionEffectDeny  PermissionEffect = "deny"
)
