package provider

import (
	"net/http"

	"github.com/supuwoerc/gapi-server/internal/router"
	"github.com/supuwoerc/gapi-server/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

var ServerSet = wire.NewSet(
	router.NewEngine,
	wire.Bind(new(http.Handler), new(*gin.Engine)),
	server.NewHttpServer,
)
