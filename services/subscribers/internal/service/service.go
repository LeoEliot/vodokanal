package service

import (
	"context"
	"fmt"

	"github.com/vodokanal/subscribers/internal/model"
	"github.com/vodokanal/subscribers/internal/repository"
)

type Service struct {
	repo *repository.Repository
}

func New(repo *repository.Repository) *Service {
	return &Service{repo: repo}
}

type ListResult struct {
	Data       []model.Subscriber `json:"data"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	Total      int                `json:"total"`
	TotalPages int                `json:"total_pages"`
}

type ListOptions struct {
	Page  int
	Limit int
	Query string
}

func (s *Service) List(ctx context.Context, opts ListOptions) (*ListResult, error) {
	if opts.Page <= 0 {
		opts.Page = 1
	}
	if opts.Limit <= 0 {
		opts.Limit = 20
	}
	if opts.Limit > 100 {
		opts.Limit = 100
	}

	var subscribers []model.Subscriber
	var err error

	if opts.Query != "" {
		subscribers, err = s.repo.Search(ctx, opts.Query, opts.Page, opts.Limit)
	} else {
		subscribers, err = s.repo.List(ctx, opts.Page, opts.Limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list subscribers: %w", err)
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count subscribers: %w", err)
	}

	totalPages := (total + opts.Limit - 1) / opts.Limit

	return &ListResult{
		Data:       subscribers,
		Page:       opts.Page,
		Limit:      opts.Limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) Get(ctx context.Context, id int) (*model.Subscriber, error) {
	subscriber, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber: %w", err)
	}
	return subscriber, nil
}

func (s *Service) GetByAccountNumber(ctx context.Context, accountNumber string) (*model.Subscriber, error) {
	subscriber, err := s.repo.GetByAccountNumber(ctx, accountNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber by account number: %w", err)
	}
	return subscriber, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*model.Subscriber, error) {
	subscriber, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscriber by email: %w", err)
	}
	return subscriber, nil
}

func (s *Service) Create(ctx context.Context, req *model.CreateSubscriberRequest) (*model.Subscriber, error) {
	// Validate account number is unique
	_, err := s.repo.GetByAccountNumber(ctx, req.AccountNumber)
	if err == nil {
		return nil, fmt.Errorf("subscriber with account number %s already exists", req.AccountNumber)
	}

	// Validate email is unique
	_, err = s.repo.GetByEmail(ctx, req.Email)
	if err == nil {
		return nil, fmt.Errorf("subscriber with email %s already exists", req.Email)
	}

	subscriber, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}

	return subscriber, nil
}

func (s *Service) Update(ctx context.Context, id int, req *model.UpdateSubscriberRequest) (*model.Subscriber, error) {
	// Check if subscriber exists
	_, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("subscriber not found: %w", err)
	}

	subscriber, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update subscriber: %w", err)
	}

	return subscriber, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	// Check if subscriber exists
	_, err := s.repo.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("subscriber not found: %w", err)
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete subscriber: %w", err)
	}

	return nil
}
