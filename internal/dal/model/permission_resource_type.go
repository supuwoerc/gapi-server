package model

import "database/sql/driver"

// ResourceType 权限的资源类型，存进 sys_permission.resource_type
// （SMALLINT，CHECK BETWEEN 1 AND 5）。
// 行尾注释即 String() 的文案，新增类型后需重新执行 go generate ./...
//
// ⚠️ 常量值直接落库，只能末尾追加，绝不能重排或复用已废弃的值——重排不报错，
// 但库里存量行的含义会被静默改写（2 一直是 frontend-menu）。新增类型还要同步
// 放宽 migrations 里 ck_permission_resource_type 的上界。
//
//go:generate stringer -type=ResourceType -linecomment -output resource_type_string.go
type ResourceType int

const (
	ResourceTypeAPI            ResourceType = 1 // api
	ResourceTypeFrontendMenu   ResourceType = 2 // frontend-menu
	ResourceTypeFrontendRoute  ResourceType = 3 // frontend-route
	ResourceTypeFrontendButton ResourceType = 4 // frontend-button
	ResourceTypeData           ResourceType = 5 // data
)

func (i *ResourceType) Scan(src any) error          { return enumIntFromDB(i, src) }
func (i ResourceType) Value() (driver.Value, error) { return enumIntIntoDB(i) }
func (i ResourceType) Text() string                 { return enumIntText(i) }
