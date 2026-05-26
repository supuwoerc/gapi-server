package provider

import (
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	pkgRedis "github.com/supuwoerc/gapi-server/pkg/redis"

	"github.com/google/wire"
)

var InfraSet = wire.NewSet(
	logger.NewLogger,
	database.NewConnection,
	pkgRedis.NewClient,
	etcd.NewClient,
)
