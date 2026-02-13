package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vodokanal/subscribers/internal/model"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) List(ctx context.Context, page, limit int) ([]model.Subscriber, error) {
	offset := (page - 1) * limit
	query := `
		SELECT id, account_number, last_name, first_name, middle_name,
		       email, phone, address, created_at, updated_at
		FROM subscribers
		ORDER BY id
		LIMIT $1 OFFSET $2
	`

	rows, err := r.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []model.Subscriber
	for rows.Next() {
		var s model.Subscriber
		err := rows.Scan(
			&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
			&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscriber: %w", err)
		}
		subscribers = append(subscribers, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return subscribers, nil
}

func (r *Repository) Count(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM subscribers`
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

func (r *Repository) Get(ctx context.Context, id int) (*model.Subscriber, error) {
	query := `
		SELECT id, account_number, last_name, first_name, middle_name,
		       email, phone, address, created_at, updated_at
		FROM subscribers
		WHERE id = $1
	`

	var s model.Subscriber
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
		&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber: %w", err)
	}

	return &s, nil
}

func (r *Repository) GetByAccountNumber(ctx context.Context, accountNumber string) (*model.Subscriber, error) {
	query := `
		SELECT id, account_number, last_name, first_name, middle_name,
		       email, phone, address, created_at, updated_at
		FROM subscribers
		WHERE account_number = $1
	`

	var s model.Subscriber
	err := r.pool.QueryRow(ctx, query, accountNumber).Scan(
		&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
		&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber by account number: %w", err)
	}

	return &s, nil
}

func (r *Repository) GetByEmail(ctx context.Context, email string) (*model.Subscriber, error) {
	query := `
		SELECT id, account_number, last_name, first_name, middle_name,
		       email, phone, address, created_at, updated_at
		FROM subscribers
		WHERE email = $1
	`

	var s model.Subscriber
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
		&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber by email: %w", err)
	}

	return &s, nil
}

func (r *Repository) Create(ctx context.Context, req *model.CreateSubscriberRequest) (*model.Subscriber, error) {
	query := `
		INSERT INTO subscribers (account_number, last_name, first_name, middle_name, email, phone, address)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`

	var s model.Subscriber
	s.AccountNumber = req.AccountNumber
	s.LastName = req.LastName
	s.FirstName = req.FirstName
	s.Phone = req.Phone
	s.Address = req.Address
	s.Email = req.Email

	if req.MiddleName != "" {
		s.MiddleName = &req.MiddleName
	}

	err := r.pool.QueryRow(ctx, query,
		req.AccountNumber, req.LastName, req.FirstName,
		nullIfEmpty(req.MiddleName), req.Email, req.Phone, req.Address,
	).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	return &s, nil
}

func (r *Repository) Update(ctx context.Context, id int, req *model.UpdateSubscriberRequest) (*model.Subscriber, error) {
	query := `
		UPDATE subscribers
		SET last_name = COALESCE($1, last_name),
		    first_name = COALESCE($2, first_name),
		    middle_name = COALESCE($3, middle_name),
		    email = COALESCE($4, email),
		    phone = COALESCE($5, phone),
		    address = COALESCE($6, address),
		    updated_at = NOW()
		WHERE id = $7
		RETURNING id, account_number, last_name, first_name, middle_name,
		          email, phone, address, created_at, updated_at
	`

	var s model.Subscriber
	err := r.pool.QueryRow(ctx, query,
		req.LastName, req.FirstName, req.MiddleName,
		req.Email, req.Phone, req.Address, id,
	).Scan(
		&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
		&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to update subscriber: %w", err)
	}

	return &s, nil
}

func (r *Repository) Delete(ctx context.Context, id int) error {
	query := `DELETE FROM subscribers WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete subscriber: %w", err)
	}
	return nil
}

func (r *Repository) Search(ctx context.Context, term string, page, limit int) ([]model.Subscriber, error) {
	offset := (page - 1) * limit
	searchTerm := "%" + term + "%"
	query := `
		SELECT id, account_number, last_name, first_name, middle_name,
		       email, phone, address, created_at, updated_at
		FROM subscribers
		WHERE account_number ILIKE $1
		   OR last_name ILIKE $1
		   OR first_name ILIKE $1
		   OR email ILIKE $1
		   OR phone ILIKE $1
		ORDER BY id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, searchTerm, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search subscribers: %w", err)
	}
	defer rows.Close()

	var subscribers []model.Subscriber
	for rows.Next() {
		var s model.Subscriber
		err := rows.Scan(
			&s.ID, &s.AccountNumber, &s.LastName, &s.FirstName, &s.MiddleName,
			&s.Email, &s.Phone, &s.Address, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan subscriber: %w", err)
		}
		subscribers = append(subscribers, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows error: %w", err)
	}

	return subscribers, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
