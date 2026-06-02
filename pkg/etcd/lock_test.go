package etcd

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type LockSuite struct {
	suite.Suite
	client *clientv3.Client
	logger Logger
}

func (s *LockSuite) SetupSuite() {
	client, err := clientv3.New(clientv3.Config{
		Endpoints:   []string{"127.0.0.1:2379"},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		s.T().Skipf("etcd not available: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err = client.Status(ctx, "127.0.0.1:2379"); err != nil {
		s.T().Skipf("etcd not reachable: %v", err)
	}
	s.client = client
	l, _ := zap.NewDevelopment()
	s.logger = l
}

func (s *LockSuite) TearDownSuite() {
	if s.client != nil {
		_ = s.client.Close()
	}
}

func (s *LockSuite) TestSingleLockUnlock() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock, err := locker.Lock(ctx, "single-test", WithTTL(10))
	s.Require().NoError(err)
	s.Require().NotNil(lock)

	err = lock.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestLockWithPrefix() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock, err := locker.Lock(ctx, "order-1", WithPrefix("order"), WithTTL(10))
	s.Require().NoError(err)

	err = lock.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestBatchLock() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock, err := locker.LockBatch(ctx, []string{"b-key", "a-key", "c-key"}, WithPrefix("batch"), WithTTL(10))
	s.Require().NoError(err)
	s.Require().NotNil(lock)
	s.Equal(3, len(lock.mutexes))

	err = lock.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestTryLockSuccess() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock, err := locker.TryLock(ctx, "try-success", WithTTL(10))
	s.Require().NoError(err)
	s.Require().NotNil(lock)

	err = lock.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestTryLockFail() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock1, err := locker.Lock(ctx, "try-fail", WithTTL(10))
	s.Require().NoError(err)

	_, err = locker.TryLock(ctx, "try-fail", WithTTL(10))
	s.Require().ErrorIs(err, ErrLockNotAcquired)

	err = lock1.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestBatchLockRollback() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	hold, err := locker.Lock(ctx, "b-key", WithPrefix("rollback"), WithTTL(10))
	s.Require().NoError(err)

	_, err = locker.TryLockBatch(ctx, []string{"a-key", "b-key", "c-key"}, WithPrefix("rollback"), WithTTL(10))
	s.Require().ErrorIs(err, ErrLockNotAcquired)

	err = hold.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestLockTimeout() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock1, err := locker.Lock(ctx, "timeout-test", WithTTL(10))
	s.Require().NoError(err)

	_, err = locker.Lock(ctx, "timeout-test", WithTTL(10), WithTimeout(500*time.Millisecond))
	s.Require().ErrorIs(err, ErrLockTimeout)

	err = lock1.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestUnlockIdempotent() {
	locker := NewLocker(s.client, s.logger)
	ctx := context.Background()

	lock, err := locker.Lock(ctx, "idempotent-test", WithTTL(10))
	s.Require().NoError(err)

	err = lock.Unlock(ctx)
	s.Require().NoError(err)

	err = lock.Unlock(ctx)
	s.Require().NoError(err)
}

func (s *LockSuite) TestMutualExclusion() {
	locker := NewLocker(s.client, s.logger)
	var counter int64
	var wg sync.WaitGroup
	const goroutines = 5

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			ctx := context.Background()
			lock, err := locker.Lock(ctx, "mutex-test", WithTTL(10))
			s.Require().NoError(err)

			atomic.AddInt64(&counter, 1)
			time.Sleep(50 * time.Millisecond)
			s.Equal(int64(1), atomic.LoadInt64(&counter))
			atomic.AddInt64(&counter, -1)

			_ = lock.Unlock(ctx)
		}()
	}
	wg.Wait()
}

func TestLockSuite(t *testing.T) {
	suite.Run(t, new(LockSuite))
}
