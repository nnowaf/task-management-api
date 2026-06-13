package test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/dto"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gdcpay/task-api/test/fakes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type noopNotifier struct{}

func (noopNotifier) Notify(context.Context, service.NotificationEvent) error { return nil }

func newTaskService() (*service.TaskService, *fakes.Store) {
	store := fakes.NewStore(24 * time.Hour)
	return service.NewTaskService(store, noopNotifier{}), store
}

// Sequential: a second request with the same key creates no new task and returns
// the identical original response.
func TestIdempotency_Sequential(t *testing.T) {
	svc, store := newTaskService()
	actor := service.Actor{ID: uuid.New(), Username: "nowaf"}
	key := uuid.New()
	req := dto.CreateTaskRequest{Title: "Write report"}

	first, err := svc.CreateTaskIdempotent(context.Background(), actor, key, req)
	require.NoError(t, err)
	require.Equal(t, 201, first.StatusCode)
	require.False(t, first.Replayed)
	require.Equal(t, 1, store.TaskCreateCount())

	second, err := svc.CreateTaskIdempotent(context.Background(), actor, key, req)
	require.NoError(t, err)
	require.True(t, second.Replayed)
	require.Equal(t, 1, store.TaskCreateCount(), "replay must not create a new task")
	require.Equal(t, first.StatusCode, second.StatusCode)
	require.Equal(t, first.Body, second.Body, "replay must return an identical body")
}

// Concurrent duplicate: N goroutines fire the same key simultaneously; exactly one
// task must be created and every response must be identical. Run with -race.
func TestIdempotency_ConcurrentDuplicate(t *testing.T) {
	svc, store := newTaskService()
	actor := service.Actor{ID: uuid.New(), Username: "nowaf"}
	key := uuid.New()
	req := dto.CreateTaskRequest{Title: "Concurrent task"}

	const n = 50
	var wg sync.WaitGroup
	bodies := make([][]byte, n)
	errs := make([]error, n)
	release := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-release // start all goroutines at once to maximize contention
			res, err := svc.CreateTaskIdempotent(context.Background(), actor, key, req)
			errs[i] = err
			if res != nil {
				bodies[i] = res.Body
			}
		}(i)
	}
	close(release)
	wg.Wait()

	for i := 0; i < n; i++ {
		require.NoError(t, errs[i])
		require.NotNil(t, bodies[i])
	}
	require.Equal(t, 1, store.TaskCreateCount(), "exactly one task under concurrency")
	for i := 1; i < n; i++ {
		require.Equal(t, bodies[0], bodies[i], "all concurrent responses must be identical")
	}
}

// The atomic insert primitive itself: hammered concurrently, exactly one caller wins
// the insert. This proves the database-level guarantee independent of singleflight.
func TestInsertIfAbsent_ConcurrentExactlyOnce(t *testing.T) {
	store := fakes.NewStore(time.Hour)
	userID, key := uuid.New(), uuid.New()

	const n = 100
	var wg sync.WaitGroup
	var inserted int64
	release := make(chan struct{})

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-release
			rec := &domain.IdempotencyRecord{
				UserID:         userID,
				IdempotencyKey: key,
				RequestHash:    "hash",
				RequestBody:    "{}",
				Status:         domain.IdempotencyProcessing,
				ResponseBody:   "null",
			}
			res, err := store.Idempotency().InsertIfAbsent(context.Background(), rec)
			require.NoError(t, err)
			if res.Inserted {
				atomic.AddInt64(&inserted, 1)
			}
		}()
	}
	close(release)
	wg.Wait()

	require.Equal(t, int64(1), inserted, "only one insert may win the race")
}

// Reusing a key with a different body is a conflict, not a silent replay.
func TestIdempotency_ReusedKeyDifferentBody(t *testing.T) {
	svc, _ := newTaskService()
	actor := service.Actor{ID: uuid.New(), Username: "nowaf"}
	key := uuid.New()

	_, err := svc.CreateTaskIdempotent(context.Background(), actor, key, dto.CreateTaskRequest{Title: "First"})
	require.NoError(t, err)

	_, err = svc.CreateTaskIdempotent(context.Background(), actor, key, dto.CreateTaskRequest{Title: "Different"})
	require.Error(t, err)
	ae, ok := domain.AsAppError(err)
	require.True(t, ok)
	require.Equal(t, domain.CodeIdempotencyKeyReused, ae.Code)
}
