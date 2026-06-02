package etcd

import (
	"context"
	"sort"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
	"go.uber.org/zap"
)

const (
	lockPrefix     = "/gapi/lock/"
	defaultLockTTL = 10
)

var (
	ErrLockTimeout     = errors.New("etcd lock: acquire timeout")
	ErrLockNotAcquired = errors.New("etcd lock: not acquired (try lock failed)")
	ErrEmptyKeys       = errors.New("etcd lock: keys must not be empty")
)

// LockOption 配置锁行为的可选参数。
type LockOption func(*lockOptions)

type lockOptions struct {
	ttl     int
	prefix  string
	timeout time.Duration
}

// WithTTL 设置锁的租约 TTL（秒）。Session 内置 keepalive 会自动续期，TTL 仅在进程异常退出时作为自动释放的兜底。
func WithTTL(seconds int) LockOption {
	return func(o *lockOptions) { o.ttl = seconds }
}

// WithPrefix 设置锁 key 的业务前缀，最终路径为 /gapi/lock/{prefix}/{key}。
func WithPrefix(prefix string) LockOption {
	return func(o *lockOptions) { o.prefix = prefix }
}

// WithTimeout 设置阻塞加锁的超时时间，超时返回 ErrLockTimeout。仅对 Lock/LockBatch 有效。
func WithTimeout(d time.Duration) LockOption {
	return func(o *lockOptions) { o.timeout = d }
}

func buildOptions(opts []LockOption) *lockOptions {
	o := &lockOptions{ttl: defaultLockTTL}
	for _, fn := range opts {
		fn(o)
	}
	return o
}

func buildKeyPath(prefix, key string) string {
	if prefix != "" {
		return lockPrefix + prefix + "/" + key
	}
	return lockPrefix + key
}

// Locker 提供基于 etcd 的分布式锁能力，线程安全，作为单例通过 Wire 注入。
type Locker struct {
	client *clientv3.Client
	logger Logger
}

// NewLocker 创建分布式锁实例。
func NewLocker(client *clientv3.Client, l Logger) *Locker {
	return &Locker{client: client, logger: l}
}

// Lock 阻塞获取单个分布式锁，可通过 WithTimeout 设置超时。
func (l *Locker) Lock(ctx context.Context, key string, opts ...LockOption) (*LockResult, error) {
	return l.LockBatch(ctx, []string{key}, opts...)
}

// TryLock 非阻塞尝试获取单个锁，获取失败立即返回 ErrLockNotAcquired。
func (l *Locker) TryLock(ctx context.Context, key string, opts ...LockOption) (*LockResult, error) {
	return l.TryLockBatch(ctx, []string{key}, opts...)
}

// LockBatch 阻塞获取多个锁。keys 按字典序排序后依次加锁防止死锁，共享同一个 session。任一 key 加锁失败则回滚已获取的锁。
func (l *Locker) LockBatch(ctx context.Context, keys []string, opts ...LockOption) (*LockResult, error) {
	return l.lockBatch(ctx, keys, false, opts...)
}

// TryLockBatch 非阻塞尝试获取多个锁。语义同 LockBatch，但不阻塞等待。
func (l *Locker) TryLockBatch(ctx context.Context, keys []string, opts ...LockOption) (*LockResult, error) {
	return l.lockBatch(ctx, keys, true, opts...)
}

func (l *Locker) lockBatch(ctx context.Context, keys []string, tryLock bool, opts ...LockOption) (*LockResult, error) {
	if len(keys) == 0 {
		return nil, ErrEmptyKeys
	}
	o := buildOptions(opts)

	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	if !tryLock && o.timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, o.timeout)
		defer cancel()
	}

	session, err := concurrency.NewSession(l.client, concurrency.WithTTL(o.ttl))
	if err != nil {
		return nil, errors.Wrap(err, "etcd lock: create session")
	}

	result := &LockResult{
		session: session,
		logger:  l.logger,
	}

	for _, key := range sorted {
		path := buildKeyPath(o.prefix, key)
		l.logger.Debug("etcd lock: acquiring", zap.String("key", path))

		mutex := concurrency.NewMutex(session, path)

		if tryLock {
			if err := mutex.TryLock(ctx); err != nil {
				result.rollback(ctx)
				if errors.Is(err, concurrency.ErrLocked) {
					return nil, ErrLockNotAcquired
				}
				return nil, errors.Wrap(err, "etcd lock: try acquire "+path)
			}
		} else {
			if err := mutex.Lock(ctx); err != nil {
				result.rollback(ctx)
				if errors.Is(err, context.DeadlineExceeded) {
					return nil, ErrLockTimeout
				}
				return nil, errors.Wrap(err, "etcd lock: acquire "+path)
			}
		}

		result.mutexes = append(result.mutexes, mutex)
		l.logger.Debug("etcd lock: acquired", zap.String("key", path))
	}

	return result, nil
}

// LockResult 持有一次加锁操作获取的所有锁。批量锁共享同一个 session（同一个 lease），
// 进程崩溃时 session 过期会原子释放所有锁。
type LockResult struct {
	session *concurrency.Session
	mutexes []*concurrency.Mutex
	logger  Logger
	done    uint32
}

// Unlock 反序释放所有锁并关闭 session。幂等：多次调用只执行一次释放。
func (r *LockResult) Unlock(ctx context.Context) error {
	if !atomic.CompareAndSwapUint32(&r.done, 0, 1) {
		return nil
	}
	var firstErr error
	for i := len(r.mutexes) - 1; i >= 0; i-- {
		if err := r.mutexes[i].Unlock(ctx); err != nil && firstErr == nil {
			firstErr = errors.Wrap(err, "etcd lock: release")
		}
	}
	if err := r.session.Close(); err != nil && firstErr == nil {
		firstErr = errors.Wrap(err, "etcd lock: close session")
	}
	r.logger.Debug("etcd lock: all released", zap.Int("count", len(r.mutexes)))
	return firstErr
}

func (r *LockResult) rollback(ctx context.Context) {
	if len(r.mutexes) == 0 {
		_ = r.session.Close()
		return
	}
	r.logger.Warn("etcd lock: rolling back", zap.Int("acquired", len(r.mutexes)))
	_ = r.Unlock(ctx)
}
