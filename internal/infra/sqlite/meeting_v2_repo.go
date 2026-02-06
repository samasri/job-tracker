package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// MeetingV2Repo implements ports.MeetingV2Repository
type MeetingV2Repo struct {
	db *DB
}

// NewMeetingV2Repo creates a new MeetingV2Repo
func NewMeetingV2Repo(db *DB) *MeetingV2Repo {
	return &MeetingV2Repo{db: db}
}

// Create inserts a new meeting into meetings_v2
func (r *MeetingV2Repo) Create(ctx context.Context, meeting *domain.MeetingV2) error {
	now := time.Now()
	meeting.CreatedAt = now
	meeting.UpdatedAt = now

	// Handle nullable fields
	var roleID, threadID interface{}
	if meeting.RoleID != "" {
		roleID = meeting.RoleID
	}
	if meeting.ThreadID != "" {
		threadID = meeting.ThreadID
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO meetings_v2 (id, occurred_at, title, role_id, thread_id, path_md, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		meeting.ID, meeting.OccurredAt.Format(time.RFC3339), meeting.Title,
		roleID, threadID, meeting.PathMD, meeting.CreatedAt, meeting.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting meeting_v2: %w", err)
	}
	return nil
}

// GetByID retrieves a meeting by ID
func (r *MeetingV2Repo) GetByID(ctx context.Context, id string) (*domain.MeetingV2, error) {
	meeting := &domain.MeetingV2{}
	var occurredAtStr string
	var roleID, threadID sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, occurred_at, title, role_id, thread_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE id = ?`, id).Scan(
		&meeting.ID, &occurredAtStr, &meeting.Title, &roleID, &threadID,
		&meeting.PathMD, &meeting.CreatedAt, &meeting.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying meeting_v2 by id: %w", err)
	}

	meeting.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	if roleID.Valid {
		meeting.RoleID = roleID.String
	}
	if threadID.Valid {
		meeting.ThreadID = threadID.String
	}

	return meeting, nil
}

// ListByRole retrieves all meetings for a role ordered by occurred_at desc
func (r *MeetingV2Repo) ListByRole(ctx context.Context, roleID string) ([]*domain.MeetingV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, thread_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE role_id = ? ORDER BY occurred_at DESC`, roleID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings_v2 by role: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// ListByThread retrieves all thread-only meetings for a thread ordered by occurred_at desc
func (r *MeetingV2Repo) ListByThread(ctx context.Context, threadID string) ([]*domain.MeetingV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, thread_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE thread_id = ? ORDER BY occurred_at DESC`, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings_v2 by thread: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// scanMeetings is a helper to scan rows into MeetingV2 slice
func (r *MeetingV2Repo) scanMeetings(rows *sql.Rows) ([]*domain.MeetingV2, error) {
	var meetings []*domain.MeetingV2
	for rows.Next() {
		m := &domain.MeetingV2{}
		var occurredAtStr string
		var roleID, threadID sql.NullString
		if err := rows.Scan(&m.ID, &occurredAtStr, &m.Title, &roleID, &threadID,
			&m.PathMD, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting_v2: %w", err)
		}
		m.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		if roleID.Valid {
			m.RoleID = roleID.String
		}
		if threadID.Valid {
			m.ThreadID = threadID.String
		}
		meetings = append(meetings, m)
	}

	return meetings, rows.Err()
}
