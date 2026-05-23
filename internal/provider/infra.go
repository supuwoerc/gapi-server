package provider

import (
	"gapi-server/pkg/database"
	"gapi-server/pkg/logger"
	pkgRedis "gapi-server/pkg/redis"

	"github.com/google/wire"
)

var InfraSet = wire.NewSet(
	logger.NewLogger,
	database.NewConnection,
	pkgRedis.NewClient,
)
