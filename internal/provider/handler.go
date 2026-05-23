package provider

import (
	"gapi-server/internal/handler"

	"github.com/google/wire"
)

var HandlerSet = wire.NewSet(
	handler.NewHealthHandler,
)
