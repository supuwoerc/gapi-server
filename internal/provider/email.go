package provider

import (
	"github.com/supuwoerc/gapi-server/internal/service"
	"github.com/supuwoerc/gapi-server/pkg/email"

	"github.com/google/wire"
)

var EmailSet = wire.NewSet(
	email.NewClient,
	email.NewTemplateRenderer,
	wire.Bind(new(email.Sender), new(*email.Client)),
	wire.Struct(new(service.EmailService), "*"),
)
