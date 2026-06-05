package provider

import (
	"github.com/supuwoerc/gapi-server/internal/app"
	"github.com/supuwoerc/gapi-server/internal/cronjob"
	"github.com/supuwoerc/gapi-server/internal/server"
	"github.com/supuwoerc/gapi-server/pkg/etcd"
)

func ProvideServerHooks(
	dynConfig *etcd.DynConfig,
	discovery *etcd.Discovery,
	jobManager *cronjob.JobManager,
	registry *etcd.Registry,
) []server.IServerHook {
	return []server.IServerHook{dynConfig, discovery, jobManager, registry}
}

func ProvideCliHooks(discovery *etcd.Discovery) []app.ICliHook {
	return []app.ICliHook{discovery}
}
