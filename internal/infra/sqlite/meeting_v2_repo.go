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

	var roleID, contactID interface{}
	if meeting.RoleID != "" {
		roleID = meeting.RoleID
	}
	if meeting.ContactID != "" {
		contactID = meeting.ContactID
	}

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO meetings_v2 (id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		meeting.ID, meeting.OccurredAt.Format(time.RFC3339), meeting.Title,
		roleID, contactID, meeting.PathMD, meeting.CreatedAt, meeting.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting meeting_v2: %w", err)
	}
	return nil
}

// GetByID retrieves a meeting by ID
func (r *MeetingV2Repo) GetByID(ctx context.Context, id string) (*domain.MeetingV2, error) {
	meeting := &domain.MeetingV2{}
	var occurredAtStr string
	var roleID, contactID sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE id = ?`, id).Scan(
		&meeting.ID, &occurredAtStr, &meeting.Title, &roleID, &contactID,
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
	if contactID.Valid {
		meeting.ContactID = contactID.String
	}

	return meeting, nil
}

// ListByRole retrieves all meetings for a role ordered by occurred_at desc
func (r *MeetingV2Repo) ListByRole(ctx context.Context, roleID string) ([]*domain.MeetingV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE role_id = ? ORDER BY occurred_at DESC`, roleID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings_v2 by role: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// ListByContact retrieves all contact meetings ordered by occurred_at desc
func (r *MeetingV2Repo) ListByContact(ctx context.Context, contactID string) ([]*domain.MeetingV2, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings_v2 WHERE contact_id = ? ORDER BY occurred_at DESC`, contactID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings_v2 by contact: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// UpdatePathMD updates the path_md for a meeting
func (r *MeetingV2Repo) UpdatePathMD(ctx context.Context, meetingID, newPath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE meetings_v2 SET path_md = ?, updated_at = ? WHERE id = ?`,
		newPath, time.Now(), meetingID)
	if err != nil {
		return fmt.Errorf("updating meeting_v2 path_md: %w", err)
	}
	return nil
}

// SetContactID sets the contact_id on an existing meeting
func (r *MeetingV2Repo) SetContactID(ctx context.Context, meetingID, contactID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE meetings_v2 SET contact_id = ?, updated_at = ? WHERE id = ?`,
		contactID, time.Now(), meetingID)
	if err != nil {
		return fmt.Errorf("setting meeting_v2 contact_id: %w", err)
	}
	return nil
}

// scanMeetings is a helper to scan rows into MeetingV2 slice
func (r *MeetingV2Repo) scanMeetings(rows *sql.Rows) ([]*domain.MeetingV2, error) {
	var meetings []*domain.MeetingV2
	for rows.Next() {
		m := &domain.MeetingV2{}
		var occurredAtStr string
		var roleID, contactID sql.NullString
		if err := rows.Scan(&m.ID, &occurredAtStr, &m.Title, &roleID, &contactID,
			&m.PathMD, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting_v2: %w", err)
		}
		m.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		if roleID.Valid {
			m.RoleID = roleID.String
		}
		if contactID.Valid {
			m.ContactID = contactID.String
		}
		meetings = append(meetings, m)
	}

	return meetings, rows.Err()
}
