package provider

import (
	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/internal/dal"
	v1 "github.com/supuwoerc/gapi-server/internal/handler/v1"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/internal/service"
	"github.com/supuwoerc/gapi-server/pkg/database"
	"github.com/supuwoerc/gapi-server/pkg/jwt"

	"github.com/google/wire"
)

var AuthSet = wire.NewSet(
	ProvideJWTManager,
	database.NewTransactionManager,
	wire.Struct(new(dal.UserDal), "*"),
	wire.Struct(new(dal.TokenDal), "*"),
	wire.Struct(new(dal.PermissionDal), "*"),
	wire.Struct(new(service.AuthService), "*"),
	wire.Bind(new(service.UserRepository), new(*dal.UserDal)),
	wire.Bind(new(service.TokenRepository), new(*dal.TokenDal)),
	wire.Bind(new(service.PermissionRepository), new(*dal.PermissionDal)),
	wire.Bind(new(v1.AuthServiceInterface), new(*service.AuthService)),
	wire.Bind(new(v1.TourServiceInterface), new(*service.AuthService)),
	ProvideAuthHandler,
	ProvideTourHandler,
)

func ProvideJWTManager(cfg *config.JWTConfig) *jwt.Manager {
	if cfg.Secret == "" {
		panic("jwt.secret must not be empty, configure it via etcd or local config")
	}
	return jwt.NewManager(cfg.Secret, cfg.Issuer, cfg.AccessTokenExpiry, cfg.RefreshTokenExpiry)
}

func ProvideAuthHandler(svc v1.AuthServiceInterface, captchaSvc v1.CaptchaServiceInterface, m *jwt.Manager) *v1.AuthHandler {
	return &v1.AuthHandler{
		Service:        svc,
		CaptchaService: captchaSvc,
		JWTAuth:        middleware.JWTAuth(m),
	}
}

func ProvideTourHandler(svc v1.TourServiceInterface, m *jwt.Manager) *v1.TourHandler {
	return &v1.TourHandler{
		Service: svc,
		JWTAuth: middleware.JWTAuth(m),
	}
}
