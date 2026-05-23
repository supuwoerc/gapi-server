package provider

import (
	"gapi-server/pkg/database"
	"gapi-server/pkg/logger"

	"github.com/google/wire"
)

var InfraSet = wire.NewSet(
	logger.NewZapLogger,
	database.NewConnection,
)
