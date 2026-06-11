package model

import (
	"database/sql/driver"
	"fmt"
	"strconv"
)

//go:generate stringer -type=ResourceType -linecomment -output resource_type_string.go
type ResourceType int

func (i *ResourceType) Scan(src any) error {
	switch v := src.(type) {
	case int64:
		*i = ResourceType(v)
	case int:
		*i = ResourceType(v)
	case []byte:
		n, err := strconv.Atoi(string(v))
		if err != nil {
			return fmt.Errorf("cannot scan %T (%v) into ResourceType", src, src)
		}
		*i = ResourceType(n)
	default:
		return fmt.Errorf("cannot scan %T into ResourceType", src)
	}
	return nil
}

func (i ResourceType) Value() (driver.Value, error) {
	return int64(i), nil
}

const (
	ResourceTypeAPI            ResourceType = 1 // api
	ResourceTypeFrontendMenu   ResourceType = 2 // frontend-menu
	ResourceTypeFrontendRoute  ResourceType = 3 // frontend-route
	ResourceTypeFrontendButton ResourceType = 4 // frontend-button
	ResourceTypeData           ResourceType = 5 // data
)
