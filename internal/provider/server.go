package provider

import (
	"net/http"

	"gapi-server/internal/router"
	"gapi-server/internal/server"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
)

var ServerSet = wire.NewSet(
	router.NewEngine,
	wire.Bind(new(http.Handler), new(*gin.Engine)),
	server.NewHttpServer,
)
