package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// meetingRepo implements ports.MeetingRepository
type meetingRepo struct {
	db *DB
}

// NewMeetingRepo creates a new meetingRepo
func NewMeetingRepo(db *DB) *meetingRepo {
	return &meetingRepo{db: db}
}

// Create inserts a new meeting into meetings
func (r *meetingRepo) Create(ctx context.Context, meeting *domain.Meeting) error {
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
		`INSERT INTO meetings (id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		meeting.ID, meeting.OccurredAt.Format(time.RFC3339), meeting.Title,
		roleID, contactID, meeting.PathMD, meeting.CreatedAt, meeting.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting meeting: %w", err)
	}
	return nil
}

// GetByID retrieves a meeting by ID
func (r *meetingRepo) GetByID(ctx context.Context, id string) (*domain.Meeting, error) {
	meeting := &domain.Meeting{}
	var occurredAtStr string
	var roleID, contactID sql.NullString

	err := r.db.QueryRowContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings WHERE id = ?`, id).Scan(
		&meeting.ID, &occurredAtStr, &meeting.Title, &roleID, &contactID,
		&meeting.PathMD, &meeting.CreatedAt, &meeting.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying meeting by id: %w", err)
	}

	var parseErr error
	meeting.OccurredAt, parseErr = time.Parse(time.RFC3339, occurredAtStr)
	if parseErr != nil {
		return nil, fmt.Errorf("parsing meeting occurred_at %q: %w", occurredAtStr, parseErr)
	}
	if roleID.Valid {
		meeting.RoleID = roleID.String
	}
	if contactID.Valid {
		meeting.ContactID = contactID.String
	}

	return meeting, nil
}

// ListByRole retrieves all meetings for a role ordered by occurred_at desc
func (r *meetingRepo) ListByRole(ctx context.Context, roleID string) ([]*domain.Meeting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings WHERE role_id = ? ORDER BY occurred_at DESC`, roleID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings by role: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// ListByContact retrieves all contact meetings ordered by occurred_at desc
func (r *meetingRepo) ListByContact(ctx context.Context, contactID string) ([]*domain.Meeting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, role_id, contact_id, path_md, created_at, updated_at
		 FROM meetings WHERE contact_id = ? ORDER BY occurred_at DESC`, contactID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings by contact: %w", err)
	}
	defer rows.Close()

	return r.scanMeetings(rows)
}

// UpdatePathMD updates the path_md for a meeting
func (r *meetingRepo) UpdatePathMD(ctx context.Context, meetingID, newPath string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE meetings SET path_md = ?, updated_at = ? WHERE id = ?`,
		newPath, time.Now(), meetingID)
	if err != nil {
		return fmt.Errorf("updating meeting path_md: %w", err)
	}
	return nil
}

// SetContactID sets the contact_id on an existing meeting
func (r *meetingRepo) SetContactID(ctx context.Context, meetingID, contactID string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE meetings SET contact_id = ?, updated_at = ? WHERE id = ?`,
		contactID, time.Now(), meetingID)
	if err != nil {
		return fmt.Errorf("setting meeting contact_id: %w", err)
	}
	return nil
}

// scanMeetings is a helper to scan rows into Meeting slice
func (r *meetingRepo) scanMeetings(rows *sql.Rows) ([]*domain.Meeting, error) {
	var meetings []*domain.Meeting
	for rows.Next() {
		m := &domain.Meeting{}
		var occurredAtStr string
		var roleID, contactID sql.NullString
		if err := rows.Scan(&m.ID, &occurredAtStr, &m.Title, &roleID, &contactID,
			&m.PathMD, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting: %w", err)
		}
		var parseErr error
		m.OccurredAt, parseErr = time.Parse(time.RFC3339, occurredAtStr)
		if parseErr != nil {
			return nil, fmt.Errorf("parsing meeting occurred_at %q: %w", occurredAtStr, parseErr)
		}
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
