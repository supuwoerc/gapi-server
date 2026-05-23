package service

import (
	"github.com/supuwoerc/gapi-server/internal/repository"
	"github.com/supuwoerc/gapi-server/pkg/logger"
)

type Service struct {
	repo   *repository.Repository
	logger *logger.Logger
}

func NewService(repo *repository.Repository, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}
