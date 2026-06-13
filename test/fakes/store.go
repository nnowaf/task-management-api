// Package fakes provides an in-memory implementation of domain.Store so services
// can be unit-tested without a database or any external service. The backing maps
// are mutex-guarded (race-safe under `go test -race`) and Atomic snapshots/restores
// state to faithfully simulate transactional rollback.
package fakes

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

func idemKey(userID, key uuid.UUID) string { return userID.String() + ":" + key.String() }

type data struct {
	mu   sync.Mutex // guards all maps
	txMu sync.Mutex // serializes Atomic blocks (simulates serializable isolation)
	ttl  time.Duration

	users    map[uuid.UUID]*domain.User
	tasks    map[uuid.UUID]*domain.Task
	taskLogs map[uuid.UUID]*domain.TaskLog
	teams    map[uuid.UUID]*domain.TeamMaster
	members  map[uuid.UUID]*domain.TeamDetail
	comments map[uuid.UUID]*domain.CommentTask
	idem     map[string]*domain.IdempotencyRecord

	taskCreateCount int
}

// Store is the fake domain.Store. Repository accessors share the same data.
type Store struct{ d *data }

var _ domain.Store = (*Store)(nil)

func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &Store{d: &data{
		ttl:      ttl,
		users:    map[uuid.UUID]*domain.User{},
		tasks:    map[uuid.UUID]*domain.Task{},
		taskLogs: map[uuid.UUID]*domain.TaskLog{},
		teams:    map[uuid.UUID]*domain.TeamMaster{},
		members:  map[uuid.UUID]*domain.TeamDetail{},
		comments: map[uuid.UUID]*domain.CommentTask{},
		idem:     map[string]*domain.IdempotencyRecord{},
	}}
}

// TaskCreateCount reports how many task rows were inserted — the key assertion for
// the idempotency exactly-once tests.
func (s *Store) TaskCreateCount() int {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	return s.d.taskCreateCount
}

func (s *Store) TaskLogCount() int {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	return len(s.d.taskLogs)
}

// --- seed helpers for tests ---

func (s *Store) SeedUser(u *domain.User) {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	cp := *u
	s.d.users[u.ID] = &cp
}

func (s *Store) SeedTeam(t *domain.TeamMaster) {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	cp := *t
	s.d.teams[t.ID] = &cp
}

func (s *Store) SeedMember(m *domain.TeamDetail) {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	cp := *m
	s.d.members[m.ID] = &cp
}

func (s *Store) SeedTask(t *domain.Task) {
	s.d.mu.Lock()
	defer s.d.mu.Unlock()
	cp := *t
	s.d.tasks[t.ID] = &cp
}

// --- domain.Store interface ---

func (s *Store) Users() domain.UserRepository              { return &userRepo{s.d} }
func (s *Store) Tasks() domain.TaskRepository              { return &taskRepo{s.d} }
func (s *Store) TaskLogs() domain.TaskLogRepository        { return &taskLogRepo{s.d} }
func (s *Store) Teams() domain.TeamRepository              { return &teamRepo{s.d} }
func (s *Store) Comments() domain.CommentRepository        { return &commentRepo{s.d} }
func (s *Store) Idempotency() domain.IdempotencyRepository { return &idemRepo{s.d} }

func (s *Store) Atomic(ctx context.Context, fn func(domain.Store) error) error {
	s.d.txMu.Lock()
	defer s.d.txMu.Unlock()

	snap := s.d.snapshot()
	if err := fn(s); err != nil {
		s.d.restore(snap)
		return err
	}
	return nil
}

// --- snapshot / restore for rollback simulation ---

type snapshot struct {
	users    map[uuid.UUID]*domain.User
	tasks    map[uuid.UUID]*domain.Task
	taskLogs map[uuid.UUID]*domain.TaskLog
	teams    map[uuid.UUID]*domain.TeamMaster
	members  map[uuid.UUID]*domain.TeamDetail
	comments map[uuid.UUID]*domain.CommentTask
	idem     map[string]*domain.IdempotencyRecord
	count    int
}

func (d *data) snapshot() snapshot {
	d.mu.Lock()
	defer d.mu.Unlock()
	return snapshot{
		users:    cloneMap(d.users),
		tasks:    cloneMap(d.tasks),
		taskLogs: cloneMap(d.taskLogs),
		teams:    cloneMap(d.teams),
		members:  cloneMap(d.members),
		comments: cloneMap(d.comments),
		idem:     cloneStrMap(d.idem),
		count:    d.taskCreateCount,
	}
}

func (d *data) restore(s snapshot) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.users = s.users
	d.tasks = s.tasks
	d.taskLogs = s.taskLogs
	d.teams = s.teams
	d.members = s.members
	d.comments = s.comments
	d.idem = s.idem
	d.taskCreateCount = s.count
}

func cloneMap[T any](in map[uuid.UUID]*T) map[uuid.UUID]*T {
	out := make(map[uuid.UUID]*T, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func cloneStrMap[T any](in map[string]*T) map[string]*T {
	out := make(map[string]*T, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

// --- repositories ---

type userRepo struct{ d *data }

func (r *userRepo) Create(_ context.Context, u *domain.User) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	cp := *u
	r.d.users[u.ID] = &cp
	return nil
}

func (r *userRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.User, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	u, ok := r.d.users[id]
	if !ok {
		return nil, nil
	}
	cp := *u
	return &cp, nil
}

func (r *userRepo) GetByLogin(_ context.Context, login string) (*domain.User, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for _, u := range r.d.users {
		if u.Username == login || u.Email == login {
			cp := *u
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *userRepo) ExistsByUsernameOrEmail(_ context.Context, username, email string) (bool, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for _, u := range r.d.users {
		if u.Username == username || u.Email == email {
			return true, nil
		}
	}
	return false, nil
}

type taskRepo struct{ d *data }

func (r *taskRepo) Create(_ context.Context, t *domain.Task) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := time.Now()
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now
	}
	t.UpdatedAt = now
	cp := *t
	r.d.tasks[t.ID] = &cp
	r.d.taskCreateCount++
	return nil
}

func (r *taskRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.Task, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	t, ok := r.d.tasks[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	if cp.AssignedTo != nil {
		if u, ok := r.d.users[*cp.AssignedTo]; ok {
			uc := *u
			cp.Assignee = &uc
		}
	}
	if cp.IDTeam != nil {
		if tm, ok := r.d.teams[*cp.IDTeam]; ok {
			tc := *tm
			cp.Team = &tc
		}
	}
	return &cp, nil
}

func (r *taskRepo) Update(_ context.Context, t *domain.Task) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	t.UpdatedAt = time.Now()
	cp := *t
	cp.Assignee = nil
	cp.Team = nil
	r.d.tasks[t.ID] = &cp
	return nil
}

func (r *taskRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	delete(r.d.tasks, id)
	return nil
}

func (r *taskRepo) List(_ context.Context, f domain.TaskFilter) ([]domain.Task, int64, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()

	teamSet := map[uuid.UUID]struct{}{}
	for _, id := range f.TeamIDs {
		teamSet[id] = struct{}{}
	}

	var matched []domain.Task
	for _, t := range r.d.tasks {
		visible := (t.CreatedBy != nil && *t.CreatedBy == f.OwnerID)
		if !visible && t.IDTeam != nil {
			if _, ok := teamSet[*t.IDTeam]; ok {
				visible = true
			}
		}
		if !visible {
			continue
		}
		if f.Status != "" && string(t.Status) != f.Status {
			continue
		}
		if f.Priority != "" && string(t.Priority) != f.Priority {
			continue
		}
		if f.AssignedTo != nil && (t.AssignedTo == nil || *t.AssignedTo != *f.AssignedTo) {
			continue
		}
		if f.Search != "" && !strings.Contains(strings.ToLower(t.Title), strings.ToLower(f.Search)) {
			continue
		}
		matched = append(matched, *t)
	}

	total := int64(len(matched))
	start := (f.Page - 1) * f.Limit
	if start < 0 || start >= len(matched) {
		return []domain.Task{}, total, nil
	}
	end := start + f.Limit
	if f.Limit <= 0 || end > len(matched) {
		end = len(matched)
	}
	return matched[start:end], total, nil
}

type taskLogRepo struct{ d *data }

func (r *taskLogRepo) Create(_ context.Context, l *domain.TaskLog) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now()
	}
	cp := *l
	r.d.taskLogs[l.ID] = &cp
	return nil
}

func (r *taskLogRepo) ListByTask(_ context.Context, taskID uuid.UUID) ([]domain.TaskLog, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	var logs []domain.TaskLog
	for _, l := range r.d.taskLogs {
		if l.IDTask == taskID {
			logs = append(logs, *l)
		}
	}
	return logs, nil
}

type teamRepo struct{ d *data }

func (r *teamRepo) CreateMaster(_ context.Context, t *domain.TeamMaster) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	cp := *t
	cp.Members = nil
	r.d.teams[t.ID] = &cp
	return nil
}

func (r *teamRepo) GetMaster(_ context.Context, id uuid.UUID) (*domain.TeamMaster, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	t, ok := r.d.teams[id]
	if !ok {
		return nil, nil
	}
	cp := *t
	cp.Members = nil
	for _, m := range r.d.members {
		if m.IDTeamMaster == id {
			cp.Members = append(cp.Members, *m)
		}
	}
	return &cp, nil
}

func (r *teamRepo) ListMastersForUser(_ context.Context, userID uuid.UUID) ([]domain.TeamMaster, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	idset := map[uuid.UUID]struct{}{}
	for _, t := range r.d.teams {
		if t.CreatedBy != nil && *t.CreatedBy == userID {
			idset[t.ID] = struct{}{}
		}
	}
	for _, m := range r.d.members {
		if m.IDUser == userID && m.Status == domain.TeamMemberActive {
			idset[m.IDTeamMaster] = struct{}{}
		}
	}
	var out []domain.TeamMaster
	for id := range idset {
		if t, ok := r.d.teams[id]; ok {
			out = append(out, *t)
		}
	}
	return out, nil
}

func (r *teamRepo) AddMember(_ context.Context, d *domain.TeamDetail) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for _, m := range r.d.members {
		if m.IDTeamMaster == d.IDTeamMaster && m.IDUser == d.IDUser {
			return domain.NewConflict(domain.CodeConflict, "user is already a member of this team")
		}
	}
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	cp := *d
	cp.User = nil
	r.d.members[d.ID] = &cp
	return nil
}

func (r *teamRepo) GetMember(_ context.Context, teamID, userID uuid.UUID) (*domain.TeamDetail, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for _, m := range r.d.members {
		if m.IDTeamMaster == teamID && m.IDUser == userID {
			cp := *m
			return &cp, nil
		}
	}
	return nil, nil
}

func (r *teamRepo) ListMembers(_ context.Context, teamID uuid.UUID) ([]domain.TeamDetail, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	var out []domain.TeamDetail
	for _, m := range r.d.members {
		if m.IDTeamMaster == teamID {
			cp := *m
			if u, ok := r.d.users[m.IDUser]; ok {
				uc := *u
				cp.User = &uc
			}
			out = append(out, cp)
		}
	}
	return out, nil
}

func (r *teamRepo) UpdateMember(_ context.Context, d *domain.TeamDetail) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	cp := *d
	cp.User = nil
	r.d.members[d.ID] = &cp
	return nil
}

func (r *teamRepo) RemoveMember(_ context.Context, teamID, userID uuid.UUID) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for id, m := range r.d.members {
		if m.IDTeamMaster == teamID && m.IDUser == userID {
			delete(r.d.members, id)
		}
	}
	return nil
}

func (r *teamRepo) TeamIDsForUser(_ context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	idset := map[uuid.UUID]struct{}{}
	for _, m := range r.d.members {
		if m.IDUser == userID && m.Status == domain.TeamMemberActive {
			idset[m.IDTeamMaster] = struct{}{}
		}
	}
	for _, t := range r.d.teams {
		if t.CreatedBy != nil && *t.CreatedBy == userID {
			idset[t.ID] = struct{}{}
		}
	}
	out := make([]uuid.UUID, 0, len(idset))
	for id := range idset {
		out = append(out, id)
	}
	return out, nil
}

type commentRepo struct{ d *data }

func (r *commentRepo) Create(_ context.Context, c *domain.CommentTask) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	now := time.Now()
	if c.CreatedAt.IsZero() {
		c.CreatedAt = now
	}
	c.UpdatedAt = now
	cp := *c
	r.d.comments[c.ID] = &cp
	return nil
}

func (r *commentRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.CommentTask, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	c, ok := r.d.comments[id]
	if !ok {
		return nil, nil
	}
	cp := *c
	return &cp, nil
}

func (r *commentRepo) ListByTask(_ context.Context, taskID uuid.UUID) ([]domain.CommentTask, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	var out []domain.CommentTask
	for _, c := range r.d.comments {
		if c.IDTask == taskID {
			out = append(out, *c)
		}
	}
	return out, nil
}

func (r *commentRepo) Update(_ context.Context, c *domain.CommentTask) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	c.UpdatedAt = time.Now()
	cp := *c
	r.d.comments[c.ID] = &cp
	return nil
}

func (r *commentRepo) SoftDelete(_ context.Context, id uuid.UUID) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	delete(r.d.comments, id)
	return nil
}

func (r *commentRepo) SoftDeleteByTask(_ context.Context, taskID uuid.UUID) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for id, c := range r.d.comments {
		if c.IDTask == taskID {
			delete(r.d.comments, id)
		}
	}
	return nil
}

type idemRepo struct{ d *data }

// InsertIfAbsent is the atomic primitive whose correctness the concurrency tests
// verify: under the mutex, exactly one caller observes an empty slot and inserts.
func (r *idemRepo) InsertIfAbsent(_ context.Context, rec *domain.IdempotencyRecord) (domain.IdempotencyResult, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()

	k := idemKey(rec.UserID, rec.IdempotencyKey)
	if existing, ok := r.d.idem[k]; ok {
		if time.Since(existing.CreatedAt) > r.d.ttl {
			delete(r.d.idem, k)
		} else {
			cp := *existing
			return domain.IdempotencyResult{Inserted: false, Record: &cp}, nil
		}
	}

	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = time.Now()
	}
	cp := *rec
	r.d.idem[k] = &cp
	return domain.IdempotencyResult{Inserted: true}, nil
}

func (r *idemRepo) Complete(_ context.Context, id uuid.UUID, code int, body string) error {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	for _, rec := range r.d.idem {
		if rec.ID == id {
			rec.Status = domain.IdempotencyCompleted
			rec.ResponseCode = code
			rec.ResponseBody = body
			return nil
		}
	}
	return nil
}

func (r *idemRepo) Get(_ context.Context, userID, key uuid.UUID) (*domain.IdempotencyRecord, error) {
	r.d.mu.Lock()
	defer r.d.mu.Unlock()
	rec, ok := r.d.idem[idemKey(userID, key)]
	if !ok {
		return nil, nil
	}
	if time.Since(rec.CreatedAt) > r.d.ttl {
		return nil, nil
	}
	cp := *rec
	return &cp, nil
}
