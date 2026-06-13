package repository

import (
	"context"
	"strconv"
	"strings"

	"github.com/gdcpay/task-api/internal/domain"
	"github.com/google/uuid"
)

type teamRepo struct{ db DBTX }

const teamCols = `id, name, created_at, updated_at, created_by, updated_by`

// memberSelect joins each membership to its user for the response summary.
const memberSelect = `
	SELECT d.id, d.id_team_master, d.id_user, d.expired_at, d.status,
	       d.created_at, d.updated_at, d.created_by, d.updated_by,
	       mu.id, mu.name, mu.username
	FROM team_details d
	LEFT JOIN users mu ON mu.id = d.id_user AND mu.deleted_at IS NULL`

func scanTeam(s scannable) (*domain.TeamMaster, error) {
	var t domain.TeamMaster
	err := s.Scan(&t.ID, &t.Name, &t.CreatedAt, &t.UpdatedAt, &t.CreatedBy, &t.UpdatedBy)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	return &t, nil
}

func scanMember(s scannable) (*domain.TeamDetail, error) {
	var (
		d                domain.TeamDetail
		status           string
		uID              *uuid.UUID
		uName, uUsername *string
	)
	err := s.Scan(&d.ID, &d.IDTeamMaster, &d.IDUser, &d.ExpiredAt, &status,
		&d.CreatedAt, &d.UpdatedAt, &d.CreatedBy, &d.UpdatedBy,
		&uID, &uName, &uUsername)
	if err != nil {
		if isNoRows(err) {
			return nil, nil
		}
		return nil, err
	}
	d.Status = domain.TeamMemberStatus(status)
	if uID != nil {
		d.User = &domain.User{Base: domain.Base{ID: *uID}, Name: derefStr(uName), Username: derefStr(uUsername)}
	}
	return &d, nil
}

func (r *teamRepo) CreateMaster(ctx context.Context, t *domain.TeamMaster) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	now := nowUTC()
	t.CreatedAt, t.UpdatedAt = now, now
	_, err := r.db.Exec(ctx,
		`INSERT INTO team_masters (id, name, created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6)`,
		t.ID, t.Name, t.CreatedAt, t.UpdatedAt, t.CreatedBy, t.UpdatedBy)
	return err
}

func (r *teamRepo) GetMaster(ctx context.Context, id uuid.UUID) (*domain.TeamMaster, error) {
	t, err := scanTeam(r.db.QueryRow(ctx,
		`SELECT `+teamCols+` FROM team_masters WHERE id = $1 AND deleted_at IS NULL`, id))
	if err != nil || t == nil {
		return t, err
	}
	members, err := r.ListMembers(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Members = members
	return t, nil
}

func (r *teamRepo) ListMastersForUser(ctx context.Context, userID uuid.UUID) ([]domain.TeamMaster, error) {
	ids, err := r.TeamIDsForUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return []domain.TeamMaster{}, nil
	}

	args := make([]any, len(ids))
	ph := make([]string, len(ids))
	for i, id := range ids {
		args[i] = id
		ph[i] = "$" + strconv.Itoa(i+1)
	}
	rows, err := r.db.Query(ctx,
		`SELECT `+teamCols+` FROM team_masters
		 WHERE id IN (`+strings.Join(ph, ",")+`) AND deleted_at IS NULL
		 ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	teams := make([]domain.TeamMaster, 0, len(ids))
	for rows.Next() {
		t, err := scanTeam(rows)
		if err != nil {
			return nil, err
		}
		teams = append(teams, *t)
	}
	return teams, rows.Err()
}

func (r *teamRepo) AddMember(ctx context.Context, d *domain.TeamDetail) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	if d.Status == "" {
		d.Status = domain.TeamMemberActive
	}
	now := nowUTC()
	d.CreatedAt, d.UpdatedAt = now, now
	_, err := r.db.Exec(ctx,
		`INSERT INTO team_details (id, id_team_master, id_user, expired_at, status,
		                           created_at, updated_at, created_by, updated_by)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		d.ID, d.IDTeamMaster, d.IDUser, d.ExpiredAt, string(d.Status),
		d.CreatedAt, d.UpdatedAt, d.CreatedBy, d.UpdatedBy)
	if err != nil && isUniqueViolation(err) {
		return domain.NewConflict(domain.CodeConflict, "user is already a member of this team")
	}
	return err
}

func (r *teamRepo) GetMember(ctx context.Context, teamID, userID uuid.UUID) (*domain.TeamDetail, error) {
	return scanMember(r.db.QueryRow(ctx,
		memberSelect+` WHERE d.id_team_master = $1 AND d.id_user = $2 AND d.deleted_at IS NULL`,
		teamID, userID))
}

func (r *teamRepo) ListMembers(ctx context.Context, teamID uuid.UUID) ([]domain.TeamDetail, error) {
	rows, err := r.db.Query(ctx,
		memberSelect+` WHERE d.id_team_master = $1 AND d.deleted_at IS NULL ORDER BY d.created_at ASC`,
		teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	members := make([]domain.TeamDetail, 0)
	for rows.Next() {
		m, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		members = append(members, *m)
	}
	return members, rows.Err()
}

func (r *teamRepo) UpdateMember(ctx context.Context, d *domain.TeamDetail) error {
	d.UpdatedAt = nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE team_details SET status=$2, expired_at=$3, updated_at=$4, updated_by=$5
		 WHERE id=$1 AND deleted_at IS NULL`,
		d.ID, string(d.Status), d.ExpiredAt, d.UpdatedAt, d.UpdatedBy)
	return err
}

func (r *teamRepo) RemoveMember(ctx context.Context, teamID, userID uuid.UUID) error {
	now := nowUTC()
	_, err := r.db.Exec(ctx,
		`UPDATE team_details SET deleted_at=$3, updated_at=$3
		 WHERE id_team_master=$1 AND id_user=$2 AND deleted_at IS NULL`, teamID, userID, now)
	return err
}

// TeamIDsForUser returns the teams the user can see: teams they actively belong to
// plus teams they own.
func (r *teamRepo) TeamIDsForUser(ctx context.Context, userID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.db.Query(ctx,
		`SELECT DISTINCT id_team_master FROM team_details
		   WHERE id_user = $1 AND status = $2 AND deleted_at IS NULL
		 UNION
		 SELECT id FROM team_masters
		   WHERE created_by = $1 AND deleted_at IS NULL`,
		userID, string(domain.TeamMemberActive))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]uuid.UUID, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
