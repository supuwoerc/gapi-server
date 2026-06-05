package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/supuwoerc/gapi-server/internal/config"
	"github.com/supuwoerc/gapi-server/pkg/logger"

	"github.com/facebookgo/grace/gracehttp"
	"go.uber.org/zap"
)

var isLinux = false

type HttpServer struct {
	server  *http.Server
	logger  *logger.Logger
	isLinux bool
	hooks   []IServerHook
}

func NewHttpServer(cfg *config.ServerConfig, handler http.Handler, l *logger.Logger, hooks []IServerHook) *HttpServer {
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	srv := &http.Server{Addr: addr, Handler: handler}
	return &HttpServer{server: srv, logger: l, isLinux: isLinux, hooks: hooks}
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

	go s.invokeOnReady()

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

	go s.invokeOnReady()

	if err := gracehttp.Serve(s.server); err != nil {
		s.logger.Fatal("grace server error", zap.Error(err))
	}
	s.logger.Info("grace server stopped")
}

func (s *HttpServer) invokeOnReady() {
	if len(s.hooks) == 0 {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	attempt := 0
	for {
		attempt++
		conn, err := net.DialTimeout("tcp", s.server.Addr, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			for _, h := range s.hooks {
				if err := h.OnReady(ctx); err != nil {
					s.logger.Fatal("hook OnReady failed", zap.Error(err))
				}
			}
			return
		}
		s.logger.Warn("server not ready, retrying", zap.Int("attempt", attempt), zap.Error(err))
		select {
		case <-ctx.Done():
			s.logger.Error("server did not become ready in time, skipping hooks")
			return
		case <-time.After(1 * time.Second):
		}
	}
}
