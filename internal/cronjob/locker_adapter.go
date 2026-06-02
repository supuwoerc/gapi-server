package cronjob

import (
	"context"

	"github.com/supuwoerc/gapi-server/pkg/etcd"
)

// LockerAdapter 将 *etcd.Locker 适配为 cronjob.DistLocker 接口，固定 prefix 和 TTL。
type LockerAdapter struct {
	locker *etcd.Locker
}

func NewLockerAdapter(locker *etcd.Locker) *LockerAdapter {
	return &LockerAdapter{locker: locker}
}

func (a *LockerAdapter) TryLock(ctx context.Context, key string) (LockResult, error) {
	result, err := a.locker.TryLock(ctx, key, etcd.WithPrefix("cron"), etcd.WithTTL(60))
	if err != nil {
		return nil, err
	}
	return result, nil
}
