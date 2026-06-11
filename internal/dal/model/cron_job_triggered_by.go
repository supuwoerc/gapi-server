package model

import (
	"database/sql/driver"
	"fmt"
)

type TriggeredBy string

func (i *TriggeredBy) Scan(src any) error {
	switch v := src.(type) {
	case string:
		*i = TriggeredBy(v)
	case []byte:
		*i = TriggeredBy(v)
	default:
		return fmt.Errorf("cannot scan %T into TriggeredBy", src)
	}
	return nil
}

func (i TriggeredBy) Value() (driver.Value, error) {
	return string(i), nil
}

const (
	TriggeredByScheduler TriggeredBy = "scheduler"
	TriggeredByManual    TriggeredBy = "manual"
)
