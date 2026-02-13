package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vodokanal/counters/internal/model"
)

type Repository struct {
	pool              *pgxpool.Pool
	verificationDays  int
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{
		pool:             pool,
		verificationDays: 1460, // Default 4 years
	}
}

// Create creates a new counter
func (r *Repository) Create(ctx context.Context, req *model.CreateCounterRequest) (*model.Counter, error) {
	installationDate, err := time.Parse("2006-01-02", req.InstallationDate)
	if err != nil {
		return nil, fmt.Errorf("invalid installation date: %w", err)
	}

	query := `
		INSERT INTO counters (subscriber_id, serial_number, type, installation_date, initial_value)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, subscriber_id, serial_number, type, installation_date,
		          verification_date, is_active, initial_value, created_at, updated_at
	`

	var counter model.Counter
	err = r.pool.QueryRow(ctx, query,
		req.SubscriberID, req.SerialNumber, req.Type, installationDate, req.InitialValue,
	).Scan(
		&counter.ID, &counter.SubscriberID, &counter.SerialNumber, &counter.Type,
		&counter.InstallationDate, &counter.VerificationDate, &counter.IsActive,
		&counter.InitialValue, &counter.CreatedAt, &counter.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create counter: %w", err)
	}

	return &counter, nil
}

// GetByID retrieves a counter by ID
func (r *Repository) GetByID(ctx context.Context, id int) (*model.Counter, error) {
	query := `
		SELECT id, subscriber_id, serial_number, type, installation_date,
		       verification_date, is_active, initial_value, created_at, updated_at
		FROM counters
		WHERE id = $1
	`

	var counter model.Counter
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&counter.ID, &counter.SubscriberID, &counter.SerialNumber, &counter.Type,
		&counter.InstallationDate, &counter.VerificationDate, &counter.IsActive,
		&counter.InitialValue, &counter.CreatedAt, &counter.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get counter: %w", err)
	}

	return &counter, nil
}

// GetBySerialNumber retrieves a counter by serial number
func (r *Repository) GetBySerialNumber(ctx context.Context, serialNumber string) (*model.Counter, error) {
	query := `
		SELECT id, subscriber_id, serial_number, type, installation_date,
		       verification_date, is_active, initial_value, created_at, updated_at
		FROM counters
		WHERE serial_number = $1
	`

	var counter model.Counter
	err := r.pool.QueryRow(ctx, query, serialNumber).Scan(
		&counter.ID, &counter.SubscriberID, &counter.SerialNumber, &counter.Type,
		&counter.InstallationDate, &counter.VerificationDate, &counter.IsActive,
		&counter.InitialValue, &counter.CreatedAt, &counter.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get counter by serial number: %w", err)
	}

	return &counter, nil
}

// List retrieves counters with pagination
func (r *Repository) List(ctx context.Context, subscriberID *int, counterType *string, page, limit int) ([]model.CounterWithDetails, error) {
	offset := (page - 1) * limit

	query := `
		SELECT c.id, c.subscriber_id, c.serial_number, c.type, c.installation_date,
		       c.verification_date, c.is_active, c.initial_value, c.created_at, c.updated_at,
		       s.last_name, s.first_name, s.middle_name, s.account_number
		FROM counters c
		LEFT JOIN subscribers s ON c.subscriber_id = s.id
		WHERE 1=1
	`

	args := []interface{}{}
	argNum := 1

	if subscriberID != nil {
		query += fmt.Sprintf(" AND c.subscriber_id = $%d", argNum)
		args = append(args, *subscriberID)
		argNum++
	}

	if counterType != nil {
		query += fmt.Sprintf(" AND c.type = $%d", argNum)
		args = append(args, *counterType)
		argNum++
	}

	query += fmt.Sprintf(" ORDER BY c.created_at DESC LIMIT $%d OFFSET $%d", argNum, argNum+1)
	args = append(args, limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list counters: %w", err)
	}
	defer rows.Close()

	var counters []model.CounterWithDetails
	for rows.Next() {
		var c model.CounterWithDetails
		var lastName, firstName, middleName *string

		err := rows.Scan(
			&c.ID, &c.SubscriberID, &c.SerialNumber, &c.Type,
			&c.InstallationDate, &c.VerificationDate, &c.IsActive,
			&c.InitialValue, &c.CreatedAt, &c.UpdatedAt,
			&lastName, &firstName, &middleName, &c.AccountNumber,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan counter: %w", err)
		}

		// Build subscriber name
		if lastName != nil && firstName != nil {
			name := *lastName + " " + *firstName
			if middleName != nil {
				name += " " + *middleName
			}
			c.SubscriberName = name
		}

		counters = append(counters, c)
	}

	return counters, nil
}

// Count returns total count of counters
func (r *Repository) Count(ctx context.Context, subscriberID *int, counterType *string) (int, error) {
	query := `SELECT COUNT(*) FROM counters WHERE 1=1`
	args := []interface{}{}
	argNum := 1

	if subscriberID != nil {
		query += fmt.Sprintf(" AND subscriber_id = $%d", argNum)
		args = append(args, *subscriberID)
		argNum++
	}

	if counterType != nil {
		query += fmt.Sprintf(" AND type = $%d", argNum)
		args = append(args, *counterType)
	}

	var count int
	err := r.pool.QueryRow(ctx, query, args...).Scan(&count)
	return count, err
}

// GetBySubscriber retrieves all counters for a subscriber
func (r *Repository) GetBySubscriber(ctx context.Context, subscriberID int) ([]model.Counter, error) {
	query := `
		SELECT id, subscriber_id, serial_number, type, installation_date,
		       verification_date, is_active, initial_value, created_at, updated_at
		FROM counters
		WHERE subscriber_id = $1
		ORDER BY installation_date DESC
	`

	rows, err := r.pool.Query(ctx, query, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counters by subscriber: %w", err)
	}
	defer rows.Close()

	var counters []model.Counter
	for rows.Next() {
		var c model.Counter
		err := rows.Scan(
			&c.ID, &c.SubscriberID, &c.SerialNumber, &c.Type,
			&c.InstallationDate, &c.VerificationDate, &c.IsActive,
			&c.InitialValue, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan counter: %w", err)
		}
		counters = append(counters, c)
	}

	return counters, nil
}

// Update updates a counter
func (r *Repository) Update(ctx context.Context, id int, req *model.UpdateCounterRequest) (*model.Counter, error) {
	query := `
		UPDATE counters
		SET serial_number = COALESCE($1, serial_number),
		    type = COALESCE($2, type),
		    verification_date = COALESCE($3::DATE, verification_date),
		    is_active = COALESCE($4, is_active),
		    updated_at = NOW()
		WHERE id = $5
		RETURNING id, subscriber_id, serial_number, type, installation_date,
		          verification_date, is_active, initial_value, created_at, updated_at
	`

	var verificationDate *string
	if req.VerificationDate != nil {
		verificationDate = req.VerificationDate
	}

	var counter model.Counter
	err := r.pool.QueryRow(ctx, query,
		req.SerialNumber, req.Type, verificationDate, req.IsActive, id,
	).Scan(
		&counter.ID, &counter.SubscriberID, &counter.SerialNumber, &counter.Type,
		&counter.InstallationDate, &counter.VerificationDate, &counter.IsActive,
		&counter.InitialValue, &counter.CreatedAt, &counter.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update counter: %w", err)
	}

	return &counter, nil
}

// Delete deletes a counter
func (r *Repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM counters WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete counter: %w", err)
	}
	return nil
}

// Activate activates a counter
func (r *Repository) Activate(ctx context.Context, id int) error {
	query := `UPDATE counters SET is_active = true, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to activate counter: %w", err)
	}
	return nil
}

// Deactivate deactivates a counter
func (r *Repository) Deactivate(ctx context.Context, id int) error {
	query := `UPDATE counters SET is_active = false, updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate counter: %w", err)
	}
	return nil
}

// GetActiveCountersBySubscriber retrieves all active counters for a subscriber
func (r *Repository) GetActiveCountersBySubscriber(ctx context.Context, subscriberID int) ([]model.Counter, error) {
	query := `
		SELECT id, subscriber_id, serial_number, type, installation_date,
		       verification_date, is_active, initial_value, created_at, updated_at
		FROM counters
		WHERE subscriber_id = $1 AND is_active = true
		ORDER BY installation_date DESC
	`

	rows, err := r.pool.Query(ctx, query, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active counters: %w", err)
	}
	defer rows.Close()

	var counters []model.Counter
	for rows.Next() {
		var c model.Counter
		err := rows.Scan(
			&c.ID, &c.SubscriberID, &c.SerialNumber, &c.Type,
			&c.InstallationDate, &c.VerificationDate, &c.IsActive,
			&c.InitialValue, &c.CreatedAt, &c.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan counter: %w", err)
		}
		counters = append(counters, c)
	}

	return counters, nil
}

// GetDashboardStats retrieves dashboard statistics
func (r *Repository) GetDashboardStats(ctx context.Context) (*model.Dashboard, error) {
	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE is_active = true) as active,
			COUNT(*) FILTER (WHERE type = 'cold') as cold,
			COUNT(*) FILTER (WHERE type = 'hot') as hot
		FROM counters
	`

	var dashboard model.Dashboard
	err := r.pool.QueryRow(ctx, query).Scan(
		&dashboard.TotalCounters, &dashboard.ActiveCounters,
		&dashboard.ColdCounters, &dashboard.HotCounters,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard stats: %w", err)
	}

	dashboard.ByType = map[string]int{
		"cold": dashboard.ColdCounters,
		"hot":  dashboard.HotCounters,
	}
	dashboard.ByStatus = map[string]int{
		"active":   dashboard.ActiveCounters,
		"inactive": dashboard.TotalCounters - dashboard.ActiveCounters,
	}

	return &dashboard, nil
}

// CheckSerialNumberExists checks if a serial number already exists
func (r *Repository) CheckSerialNumberExists(ctx context.Context, serialNumber string, excludeID *int) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM counters WHERE serial_number = $1`
	args := []interface{}{serialNumber}
	argNum := 2

	if excludeID != nil {
		query += fmt.Sprintf(" AND id != $%d", argNum)
		args = append(args, *excludeID)
		argNum++
	}

	query += ")"

	var exists bool
	err := r.pool.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}
