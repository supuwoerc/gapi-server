package model

//go:generate stringer -type=ResourceType -linecomment -output resource_type_string.go
type ResourceType int

const (
	ResourceTypeAPI            ResourceType = 1 // api
	ResourceTypeFrontendMenu   ResourceType = 2 // frontend-menu
	ResourceTypeFrontendRoute  ResourceType = 3 // frontend-route
	ResourceTypeFrontendButton ResourceType = 4 // frontend-button
	ResourceTypeData           ResourceType = 5 // data
)
