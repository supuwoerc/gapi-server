package router

import "github.com/gin-gonic/gin"

// Registrar is the interface for registering routes on a router group.
type Registrar interface {
	Register(r *gin.RouterGroup)
}

// V1Handlers holds all v1 API route registrars.
type V1Handlers struct {
	Registrars []Registrar
}

// NewV1Handlers creates V1Handlers from a slice of registrars.
func NewV1Handlers(registrars []Registrar) *V1Handlers {
	return &V1Handlers{Registrars: registrars}
}

// Register delegates route registration to each registrar.
func (h *V1Handlers) Register(r *gin.RouterGroup) {
	for _, reg := range h.Registrars {
		reg.Register(r)
	}
}
