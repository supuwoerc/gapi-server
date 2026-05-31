package app

import (
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type Gen struct {
	Logger *logger.Logger
	DB     *gorm.DB
}

func (g *Gen) Close() {
	defer func() {
		_ = g.Logger.Sync()
	}()
	defer g.Logger.Info("gen cli clean is executed")
	if sqlDB, err := g.DB.DB(); err != nil {
		g.Logger.Error("failed to get sql.DB", zap.Error(err))
	} else if err := sqlDB.Close(); err != nil {
		g.Logger.Error("failed to close database", zap.Error(err))
	}
}
