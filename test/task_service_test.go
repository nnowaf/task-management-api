package test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/gdcpay/task-api/internal/service"
	"github.com/gdcpay/task-api/test/fakes"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type failingNotifier struct{}

func (failingNotifier) Notify(context.Context, service.NotificationEvent) error {
	return errors.New("notification gateway down")
}

// teamFixture seeds a creator, a target member, a team, and a task assigned to the
// creator, returning the store plus the key ids.
func teamFixture(notifier service.Notifier) (*service.TaskService, *fakes.Store, uuid.UUID, uuid.UUID, uuid.UUID) {
	store := fakes.NewStore(time.Hour)
	creatorID, targetID, teamID, taskID := uuid.New(), uuid.New(), uuid.New(), uuid.New()

	store.SeedUser(&domain.User{Base: domain.Base{ID: creatorID}, Username: "creator"})
	store.SeedUser(&domain.User{Base: domain.Base{ID: targetID}, Username: "target"})
	store.SeedTeam(&domain.TeamMaster{Base: domain.Base{ID: teamID, CreatedBy: &creatorID}, Name: "Team A"})
	store.SeedMember(&domain.TeamDetail{Base: domain.Base{ID: uuid.New()}, IDTeamMaster: teamID, IDUser: creatorID, Status: domain.TeamMemberActive})
	store.SeedMember(&domain.TeamDetail{Base: domain.Base{ID: uuid.New()}, IDTeamMaster: teamID, IDUser: targetID, Status: domain.TeamMemberActive})
	store.SeedTask(&domain.Task{
		Base:       domain.Base{ID: taskID, CreatedBy: &creatorID},
		Title:      "Ship release",
		IDTeam:     &teamID,
		AssignedTo: &creatorID,
		Status:     domain.TaskTODO,
		Priority:   domain.PriorityMedium,
	})

	return service.NewTaskService(store, notifier), store, creatorID, targetID, taskID
}

// The assign transaction must commit assignee + audit log atomically.
func TestAssign_Success(t *testing.T) {
	svc, store, creatorID, targetID, taskID := teamFixture(noopNotifier{})
	actor := service.Actor{ID: creatorID, Username: "creator"}

	updated, err := svc.Assign(context.Background(), actor, taskID, targetID)
	require.NoError(t, err)
	require.Equal(t, targetID, *updated.AssignedTo)
	require.Equal(t, 1, store.TaskLogCount(), "assignment writes exactly one audit log")
}

// If the notification step fails, the whole transaction rolls back: assignee unchanged
// and no audit log persisted.
func TestAssign_RollbackOnNotifierFailure(t *testing.T) {
	svc, store, creatorID, targetID, taskID := teamFixture(failingNotifier{})
	actor := service.Actor{ID: creatorID, Username: "creator"}

	_, err := svc.Assign(context.Background(), actor, taskID, targetID)
	require.Error(t, err)

	got, _ := store.Tasks().GetByID(context.Background(), taskID)
	require.NotNil(t, got)
	require.Equal(t, creatorID, *got.AssignedTo, "assignee must be unchanged after rollback")
	require.Equal(t, 0, store.TaskLogCount(), "no audit log may persist after rollback")
}

// Assigning a task that has no team is rejected.
func TestAssign_RejectedWithoutTeam(t *testing.T) {
	store := fakes.NewStore(time.Hour)
	creatorID := uuid.New()
	taskID := uuid.New()
	store.SeedTask(&domain.Task{
		Base:       domain.Base{ID: taskID, CreatedBy: &creatorID},
		Title:      "solo",
		AssignedTo: &creatorID,
		Status:     domain.TaskTODO,
		Priority:   domain.PriorityMedium,
	})
	svc := service.NewTaskService(store, noopNotifier{})

	_, err := svc.Assign(context.Background(), service.Actor{ID: creatorID, Username: "creator"}, taskID, uuid.New())
	require.Error(t, err)
	ae, ok := domain.AsAppError(err)
	require.True(t, ok)
	require.Equal(t, 422, ae.HTTPStatus)
}
