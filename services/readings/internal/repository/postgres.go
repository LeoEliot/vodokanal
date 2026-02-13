package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vodokanal/readings/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create creates a new reading
func (r *Repository) Create(ctx context.Context, req *model.CreateReadingRequest, submittedBy *int) (*model.Reading, error) {
	readingDate := time.Now()
	if req.ReadingDate != "" {
		parsed, err := time.Parse("2006-01-02", req.ReadingDate)
		if err == nil {
			readingDate = parsed
		}
	}

	query := `
		INSERT INTO readings (subscriber_id, counter_id, value, reading_date, submitted_by)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, subscriber_id, counter_id, value, reading_date, submitted_by,
		          verified, verified_by, verified_at, created_at
	`

	var reading model.Reading
	err := r.pool.QueryRow(ctx, query,
		req.SubscriberID, req.CounterID, req.Value, readingDate, submittedBy,
	).Scan(
		&reading.ID, &reading.SubscriberID, &reading.CounterID, &reading.Value,
		&reading.ReadingDate, &reading.SubmittedBy, &reading.Verified,
		&reading.VerifiedBy, &reading.VerifiedAt, &reading.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create reading: %w", err)
	}

	return &reading, nil
}

// GetByID retrieves a reading by ID
func (r *Repository) GetByID(ctx context.Context, id int) (*model.Reading, error) {
	query := `
		SELECT id, subscriber_id, counter_id, value, reading_date, submitted_by,
		       verified, verified_by, verified_at, created_at
		FROM readings
		WHERE id = $1
	`

	var reading model.Reading
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&reading.ID, &reading.SubscriberID, &reading.CounterID, &reading.Value,
		&reading.ReadingDate, &reading.SubmittedBy, &reading.Verified,
		&reading.VerifiedBy, &reading.VerifiedAt, &reading.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get reading: %w", err)
	}

	return &reading, nil
}

// List retrieves readings with pagination
func (r *Repository) List(ctx context.Context, subscriberID, counterID *int, page, limit int) ([]model.ReadingWithDetails, error) {
	offset := (page - 1) * limit

	query := `
		SELECT r.id, r.subscriber_id, r.counter_id, r.value, r.reading_date,
		       r.submitted_by, r.verified, r.verified_by, r.verified_at, r.created_at,
		       s.last_name, s.first_name, s.middle_name,
		       c.serial_number, c.type
		FROM readings r
		LEFT JOIN subscribers s ON r.subscriber_id = s.id
		LEFT JOIN counters c ON r.counter_id = c.id
		WHERE 1=1
	`

	args := []interface{}{}
	argNum := 1

	if subscriberID != nil {
		query += fmt.Sprintf(" AND r.subscriber_id = $%d", argNum)
		args = append(args, *subscriberID)
		argNum++
	}

	if counterID != nil {
		query += fmt.Sprintf(" AND r.counter_id = $%d", argNum)
		args = append(args, *counterID)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY r.reading_date DESC, r.created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list readings: %w", err)
	}
	defer rows.Close()

	var readings []model.ReadingWithDetails
	for rows.Next() {
		var r model.ReadingWithDetails
		var lastName, firstName *string
		var middleName, serialNumber, counterType *string

		err := rows.Scan(
			&r.ID, &r.SubscriberID, &r.CounterID, &r.Value, &r.ReadingDate,
			&r.SubmittedBy, &r.Verified, &r.VerifiedBy, &r.VerifiedAt, &r.CreatedAt,
			&lastName, &firstName, &middleName, &serialNumber, &counterType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}

		// Build subscriber name
		if lastName != nil && firstName != nil {
			name := *lastName + " " + *firstName
			if middleName != nil {
				name += " " + *middleName
			}
			r.SubscriberName = name
		}

		// Set counter details
		if serialNumber != nil {
			r.CounterSerial = *serialNumber
		}
		if counterType != nil {
			r.CounterType = *counterType
		}

		readings = append(readings, r)
	}

	return readings, nil
}

// Count returns total count of readings for a subscriber/counter
func (r *Repository) Count(ctx context.Context, subscriberID, counterID *int) (int, error) {
	query := `SELECT COUNT(*) FROM readings WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if subscriberID != nil {
		query += fmt.Sprintf(" AND subscriber_id = $%d", argNum)
		args = append(args, *subscriberID)
		argNum++
	}

	if counterID != nil {
		query += fmt.Sprintf(" AND counter_id = $%d", argNum)
		args = append(args, *counterID)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// GetLatestByCounter retrieves the latest reading for a counter
func (r *Repository) GetLatestByCounter(ctx context.Context, counterID int) (*model.Reading, error) {
	query := `
		SELECT id, subscriber_id, counter_id, value, reading_date, submitted_by,
		       verified, verified_by, verified_at, created_at
		FROM readings
		WHERE counter_id = $1
		ORDER BY reading_date DESC, created_at DESC
		LIMIT 1
	`

	var reading model.Reading
	err := r.pool.QueryRow(ctx, query, counterID).Scan(
		&reading.ID, &reading.SubscriberID, &reading.CounterID, &reading.Value,
		&reading.ReadingDate, &reading.SubmittedBy, &reading.Verified,
		&reading.VerifiedBy, &reading.VerifiedAt, &reading.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get latest reading: %w", err)
	}

	return &reading, nil
}

// GetLatestBySubscriber retrieves all latest readings for a subscriber
func (r *Repository) GetLatestBySubscriber(ctx context.Context, subscriberID int) ([]model.Reading, error) {
	query := `
		SELECT DISTINCT ON (counter_id)
		       id, subscriber_id, counter_id, value, reading_date, submitted_by,
		       verified, verified_by, verified_at, created_at
		FROM readings
		WHERE subscriber_id = $1
		ORDER BY counter_id, reading_date DESC, created_at DESC
	`

	rows, err := r.pool.Query(ctx, query, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get latest readings: %w", err)
	}
	defer rows.Close()

	var readings []model.Reading
	for rows.Next() {
		var r model.Reading
		err := rows.Scan(
			&r.ID, &r.SubscriberID, &r.CounterID, &r.Value,
			&r.ReadingDate, &r.SubmittedBy, &r.Verified,
			&r.VerifiedBy, &r.VerifiedAt, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, r)
	}

	return readings, nil
}

// GetByDateRange retrieves readings within a date range
func (r *Repository) GetByDateRange(ctx context.Context, subscriberID int, startDate, endDate time.Time) ([]model.Reading, error) {
	query := `
		SELECT id, subscriber_id, counter_id, value, reading_date, submitted_by,
		       verified, verified_by, verified_at, created_at
		FROM readings
		WHERE subscriber_id = $1 AND reading_date BETWEEN $2 AND $3
		ORDER BY reading_date ASC, created_at ASC
	`

	rows, err := r.pool.Query(ctx, query, subscriberID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get readings by date range: %w", err)
	}
	defer rows.Close()

	var readings []model.Reading
	for rows.Next() {
		var r model.Reading
		err := rows.Scan(
			&r.ID, &r.SubscriberID, &r.CounterID, &r.Value,
			&r.ReadingDate, &r.SubmittedBy, &r.Verified,
			&r.VerifiedBy, &r.VerifiedAt, &r.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan reading: %w", err)
		}
		readings = append(readings, r)
	}

	return readings, nil
}

// Update updates a reading
func (r *Repository) Update(ctx context.Context, id int, req *model.UpdateReadingRequest) (*model.Reading, error) {
	query := `
		UPDATE readings
		SET value = COALESCE($1, value),
		    verified = COALESCE($2, verified),
		    verified_by = COALESCE($3, verified_by),
		    verified_at = CASE WHEN $3 IS NOT NULL THEN NOW() ELSE verified_at END
		WHERE id = $4
		RETURNING id, subscriber_id, counter_id, value, reading_date, submitted_by,
		          verified, verified_by, verified_at, created_at
	`

	var reading model.Reading
	err := r.pool.QueryRow(ctx, query,
		req.Value, req.Verified, req.VerifiedBy, id,
	).Scan(
		&reading.ID, &reading.SubscriberID, &reading.CounterID, &reading.Value,
		&reading.ReadingDate, &reading.SubmittedBy, &reading.Verified,
		&reading.VerifiedBy, &reading.VerifiedAt, &reading.CreatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update reading: %w", err)
	}

	return &reading, nil
}

// Delete deletes a reading
func (r *Repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM readings WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete reading: %w", err)
	}
	return nil
}

// GetMonthlyReadings retrieves readings grouped by month for a subscriber
func (r *Repository) GetMonthlyReadings(ctx context.Context, subscriberID int, months int) ([]model.MonthlyReadings, error) {
	query := `
		SELECT TO_CHAR(reading_date, 'YYYY-MM') as month,
		       json_agg(json_build_object(
			       'id', id,
			       'counter_id', counter_id,
			       'value', value,
			       'reading_date', reading_date
		       )) as readings,
		       SUM(value) as total
		FROM readings
		WHERE subscriber_id = $1
		      AND reading_date >= CURRENT_DATE - INTERVAL '1 year'
		GROUP BY TO_CHAR(reading_date, 'YYYY-MM')
		ORDER BY month DESC
		LIMIT $2
	`

	rows, err := r.pool.Query(ctx, query, subscriberID, months)
	if err != nil {
		return nil, fmt.Errorf("failed to get monthly readings: %w", err)
	}
	defer rows.Close()

	var results []model.MonthlyReadings
	for rows.Next() {
		var r model.MonthlyReadings
		err := rows.Scan(&r.Month, &r.Readings, &r.Total)
		if err != nil {
			return nil, fmt.Errorf("failed to scan monthly reading: %w", err)
		}
		results = append(results, r)
	}

	return results, nil
}
