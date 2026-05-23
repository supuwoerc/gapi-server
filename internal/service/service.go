package service

import (
	"gapi-server/internal/repository"
	"gapi-server/pkg/logger"
)

type Service struct {
	repo   *repository.Repository
	logger *logger.Logger
}

func NewService(repo *repository.Repository, logger *logger.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}
