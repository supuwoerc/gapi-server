package service

import (
	"gapi-server/internal/repository"

	"go.uber.org/zap"
)

// Service is the base business logic struct.
type Service struct {
	repo   *repository.Repository
	logger *zap.Logger
}

// NewService creates a new base Service.
func NewService(repo *repository.Repository, logger *zap.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}
