package provider

import (
	"github.com/supuwoerc/gapi-server/internal/server"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
)

func ProvideServerHooks(registry *etcd.Registry) []server.IServerHook {
	return []server.IServerHook{registry}
}
