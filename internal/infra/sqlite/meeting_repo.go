package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"jobtracker/internal/domain"
)

// MeetingRepo implements ports.MeetingRepository
type MeetingRepo struct {
	db *DB
}

// NewMeetingRepo creates a new MeetingRepo
func NewMeetingRepo(db *DB) *MeetingRepo {
	return &MeetingRepo{db: db}
}

// Create inserts a new meeting
func (r *MeetingRepo) Create(ctx context.Context, meeting *domain.Meeting) error {
	now := time.Now()
	meeting.CreatedAt = now
	meeting.UpdatedAt = now

	_, err := r.db.ExecContext(ctx,
		`INSERT INTO meetings (id, occurred_at, title, company_id, path_md, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		meeting.ID, meeting.OccurredAt.Format(time.RFC3339), meeting.Title,
		meeting.CompanyID, meeting.PathMD, meeting.CreatedAt, meeting.UpdatedAt)

	if err != nil {
		return fmt.Errorf("inserting meeting: %w", err)
	}
	return nil
}

// GetByID retrieves a meeting by ID
func (r *MeetingRepo) GetByID(ctx context.Context, id string) (*domain.Meeting, error) {
	meeting := &domain.Meeting{}
	var occurredAtStr string

	err := r.db.QueryRowContext(ctx,
		`SELECT id, occurred_at, title, company_id, path_md, created_at, updated_at
		 FROM meetings WHERE id = ?`, id).Scan(
		&meeting.ID, &occurredAtStr, &meeting.Title, &meeting.CompanyID,
		&meeting.PathMD, &meeting.CreatedAt, &meeting.UpdatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("querying meeting by id: %w", err)
	}

	meeting.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
	return meeting, nil
}

// ListByCompany retrieves all meetings for a company ordered by occurred_at desc
func (r *MeetingRepo) ListByCompany(ctx context.Context, companyID string) ([]*domain.Meeting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, occurred_at, title, company_id, path_md, created_at, updated_at
		 FROM meetings WHERE company_id = ? ORDER BY occurred_at DESC`, companyID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings by company: %w", err)
	}
	defer rows.Close()

	var meetings []*domain.Meeting
	for rows.Next() {
		m := &domain.Meeting{}
		var occurredAtStr string
		if err := rows.Scan(&m.ID, &occurredAtStr, &m.Title, &m.CompanyID,
			&m.PathMD, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting: %w", err)
		}
		m.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		meetings = append(meetings, m)
	}

	return meetings, rows.Err()
}

// ListByThread retrieves all meetings linked to a thread ordered by occurred_at desc
func (r *MeetingRepo) ListByThread(ctx context.Context, threadID string) ([]*domain.Meeting, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT m.id, m.occurred_at, m.title, m.company_id, m.path_md, m.created_at, m.updated_at
		 FROM meetings m
		 INNER JOIN meeting_threads mt ON mt.meeting_id = m.id
		 WHERE mt.thread_id = ?
		 ORDER BY m.occurred_at DESC`, threadID)
	if err != nil {
		return nil, fmt.Errorf("listing meetings by thread: %w", err)
	}
	defer rows.Close()

	var meetings []*domain.Meeting
	for rows.Next() {
		m := &domain.Meeting{}
		var occurredAtStr string
		if err := rows.Scan(&m.ID, &occurredAtStr, &m.Title, &m.CompanyID,
			&m.PathMD, &m.CreatedAt, &m.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scanning meeting: %w", err)
		}
		m.OccurredAt, _ = time.Parse(time.RFC3339, occurredAtStr)
		meetings = append(meetings, m)
	}

	return meetings, rows.Err()
}

// LinkThread links a meeting to a thread
func (r *MeetingRepo) LinkThread(ctx context.Context, meetingID, threadID string) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT OR IGNORE INTO meeting_threads (meeting_id, thread_id) VALUES (?, ?)`,
		meetingID, threadID)
	if err != nil {
		return fmt.Errorf("linking meeting to thread: %w", err)
	}
	return nil
}
