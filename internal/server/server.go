package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"gapi-server/internal/config"

	"github.com/facebookgo/grace/gracehttp"
	"go.uber.org/zap"
)

var isLinux = false

type HttpServer struct {
	server  *http.Server
	logger  *zap.Logger
	isLinux bool
}

func NewHttpServer(cfg *config.ServerConfig, handler http.Handler, logger *zap.Logger) *HttpServer {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{Addr: addr, Handler: handler}
	return &HttpServer{server: srv, logger: logger, isLinux: isLinux}
}

func (s *HttpServer) Run() {
	if s.isLinux {
		s.graceRunServe()
	} else {
		s.runServer()
	}
}

func (s *HttpServer) runServer() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	go func() {
		s.logger.Info("server running", zap.String("addr", s.server.Addr))
		if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Fatal("server error", zap.Error(err))
		}
	}()
	<-ctx.Done()
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.server.Shutdown(timeoutCtx); err != nil {
		s.logger.Error("server shutdown error", zap.Error(err))
	}
	s.logger.Info("server stopped")
}

func (s *HttpServer) graceRunServe() {
	s.logger.Info("grace server running", zap.String("addr", s.server.Addr))
	if err := gracehttp.Serve(s.server); err != nil {
		s.logger.Fatal("grace server error", zap.Error(err))
	}
	s.logger.Info("grace server stopped")
}
