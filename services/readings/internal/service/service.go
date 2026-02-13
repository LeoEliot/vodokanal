package service

import (
	"context"
	"fmt"
	"time"

	"github.com/vodokanal/readings/internal/model"
	"github.com/vodokanal/readings/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo      *repository.Repository
	validator *model.ReadingValidator
	logger    *zap.Logger
}

func New(repo *repository.Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: model.DefaultReadingValidator(),
		logger:    logger,
	}
}

func (s *Service) Create(ctx context.Context, req *model.CreateReadingRequest, submittedBy *int) (*model.Reading, error) {
	// Get latest reading for validation
	latest, err := s.repo.GetLatestByCounter(ctx, req.CounterID)
	previousValue := 0.0
	if err == nil && latest != nil {
		previousValue = latest.Value
	}

	// Validate request
	if err := s.validator.ValidateCreate(req, previousValue); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Create reading
	reading, err := s.repo.Create(ctx, req, submittedBy)
	if err != nil {
		return nil, fmt.Errorf("failed to create reading: %w", err)
	}

	s.logger.Info("Reading created",
		zap.Int("reading_id", reading.ID),
		zap.Int("subscriber_id", req.SubscriberID),
		zap.Int("counter_id", req.CounterID),
		zap.Float64("value", req.Value),
	)

	return reading, nil
}

func (s *Service) Get(ctx context.Context, id int) (*model.Reading, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) List(ctx context.Context, subscriberID, counterID *int, page, limit int) (*model.ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	readings, err := s.repo.List(ctx, subscriberID, counterID, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list readings: %w", err)
	}

	total, err := s.repo.Count(ctx, subscriberID, counterID)
	if err != nil {
		return nil, fmt.Errorf("failed to count readings: %w", err)
	}

	totalPages := (total + limit - 1) / limit

	return &model.ListResult{
		Data:       readings,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) GetLatestByCounter(ctx context.Context, counterID int) (*model.Reading, error) {
	return s.repo.GetLatestByCounter(ctx, counterID)
}

func (s *Service) GetLatestBySubscriber(ctx context.Context, subscriberID int) ([]model.Reading, error) {
	return s.repo.GetLatestBySubscriber(ctx, subscriberID)
}

func (s *Service) GetByDateRange(ctx context.Context, subscriberID int, startDate, endDate string) ([]model.Reading, error) {
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("invalid start date: %w", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("invalid end date: %w", err)
	}

	if end.Before(start) {
		return nil, fmt.Errorf("end date must be after start date")
	}

	return s.repo.GetByDateRange(ctx, subscriberID, start, end)
}

func (s *Service) Update(ctx context.Context, id int, req *model.UpdateReadingRequest) (*model.Reading, error) {
	// Validate update request
	if err := s.validator.ValidateUpdate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	reading, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update reading: %w", err)
	}

	s.logger.Info("Reading updated",
		zap.Int("reading_id", id),
	)

	return reading, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete reading: %w", err)
	}

	s.logger.Info("Reading deleted",
		zap.Int("reading_id", id),
	)

	return nil
}

func (s *Service) Verify(ctx context.Context, id int, verifiedBy int) (*model.Reading, error) {
	req := &model.UpdateReadingRequest{
		Verified:   boolPtr(true),
		VerifiedBy: &verifiedBy,
	}

	reading, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to verify reading: %w", err)
	}

	s.logger.Info("Reading verified",
		zap.Int("reading_id", id),
		zap.Int("verified_by", verifiedBy),
	)

	return reading, nil
}

func (s *Service) GetMonthlyReadings(ctx context.Context, subscriberID int, months int) ([]model.MonthlyReadings, error) {
	if months <= 0 {
		months = 12
	}
	if months > 24 {
		months = 24
	}

	return s.repo.GetMonthlyReadings(ctx, subscriberID, months)
}

func (s *Service) GetStatistics(ctx context.Context, subscriberID, counterID int) (*model.Statistics, error) {
	// Get current reading
	current, err := s.repo.GetLatestByCounter(ctx, counterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get current reading: %w", err)
	}

	// Get previous month reading
	startDate := time.Now().AddDate(0, -1, 0)
	endDate := time.Now()
	prevReadings, err := s.repo.GetByDateRange(ctx, subscriberID, startDate, endDate)

	previousValue := 0.0
	if len(prevReadings) > 0 {
		// Get the first (oldest) reading in the range
		previousValue = prevReadings[0].Value
	}

	consumption := current.Value - previousValue
	if consumption < 0 {
		consumption = 0
	}

	return &model.Statistics{
		SubscriberID:   subscriberID,
		CounterID:      counterID,
		CurrentValue:  current.Value,
		PreviousValue: previousValue,
		Consumption:    consumption,
		Period:         startDate.Format("2006-01"),
	}, nil
}

func boolPtr(b bool) *bool {
	return &b
}
