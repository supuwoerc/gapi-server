package provider

import (
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
	"github.com/supuwoerc/gapi-server/pkg/logger"
	pkgRedis "github.com/supuwoerc/gapi-server/pkg/redis"

	"github.com/google/wire"
)

var BaseInfraSet = wire.NewSet(
	logger.NewLogger,
	etcd.NewClient,
	etcd.NewLocker,
	wire.Bind(new(etcd.Logger), new(*logger.Logger)),
	database.NewConnection,
	pkgRedis.NewClient,
)

var InfraSet = wire.NewSet(
	BaseInfraSet,
	etcd.NewDynConfig,
)
