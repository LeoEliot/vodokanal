package service

import (
	"context"
	"fmt"
	"time"

	"github.com/vodokanal/counters/internal/model"
	"github.com/vodokanal/counters/internal/repository"
	"go.uber.org/zap"
)

type Service struct {
	repo      *repository.Repository
	validator *model.CounterValidator
	logger    *zap.Logger
}

func New(repo *repository.Repository, logger *zap.Logger) *Service {
	return &Service{
		repo:      repo,
		validator: model.DefaultCounterValidator(),
		logger:    logger,
	}
}

func (s *Service) Create(ctx context.Context, req *model.CreateCounterRequest) (*model.Counter, error) {
	// Validate request
	if err := s.validator.ValidateCreate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if serial number already exists
	exists, err := s.repo.CheckSerialNumberExists(ctx, req.SerialNumber, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check serial number: %w", err)
	}
	if exists {
		return nil, model.ErrDuplicateSerialNumber
	}

	// Create counter
	counter, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create counter: %w", err)
	}

	s.logger.Info("Counter created",
		zap.Int("counter_id", counter.ID),
		zap.Int("subscriber_id", req.SubscriberID),
		zap.String("serial_number", req.SerialNumber),
	)

	return counter, nil
}

func (s *Service) Get(ctx context.Context, id int) (*model.Counter, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) GetBySerialNumber(ctx context.Context, serialNumber string) (*model.Counter, error) {
	return s.repo.GetBySerialNumber(ctx, serialNumber)
}

func (s *Service) List(ctx context.Context, subscriberID *int, counterType *string, page, limit int) (*model.ListResult, error) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	counters, err := s.repo.List(ctx, subscriberID, counterType, page, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list counters: %w", err)
	}

	total, err := s.repo.Count(ctx, subscriberID, counterType)
	if err != nil {
		return nil, fmt.Errorf("failed to count counters: %w", err)
	}

	totalPages := (total + limit - 1) / limit

	return &model.ListResult{
		Data:       counters,
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
	}, nil
}

func (s *Service) GetBySubscriber(ctx context.Context, subscriberID int) ([]model.Counter, error) {
	return s.repo.GetBySubscriber(ctx, subscriberID)
}

func (s *Service) GetActiveBySubscriber(ctx context.Context, subscriberID int) ([]model.Counter, error) {
	return s.repo.GetActiveCountersBySubscriber(ctx, subscriberID)
}

func (s *Service) Update(ctx context.Context, id int, req *model.UpdateCounterRequest) (*model.Counter, error) {
	// Validate request
	if err := s.validator.ValidateUpdate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Check if serial number already exists (if being updated)
	if req.SerialNumber != nil {
		exists, err := s.repo.CheckSerialNumberExists(ctx, *req.SerialNumber, &id)
		if err != nil {
			return nil, fmt.Errorf("failed to check serial number: %w", err)
		}
		if exists {
			return nil, model.ErrDuplicateSerialNumber
		}
	}

	counter, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, fmt.Errorf("failed to update counter: %w", err)
	}

	s.logger.Info("Counter updated",
		zap.Int("counter_id", id),
	)

	return counter, nil
}

func (s *Service) Delete(ctx context.Context, id int) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete counter: %w", err)
	}

	s.logger.Info("Counter deleted",
		zap.Int("counter_id", id),
	)

	return nil
}

func (s *Service) Activate(ctx context.Context, id int) error {
	if err := s.repo.Activate(ctx, id); err != nil {
		return fmt.Errorf("failed to activate counter: %w", err)
	}

	s.logger.Info("Counter activated",
		zap.Int("counter_id", id),
	)

	return nil
}

func (s *Service) Deactivate(ctx context.Context, id int) error {
	if err := s.repo.Deactivate(ctx, id); err != nil {
		return fmt.Errorf("failed to deactivate counter: %w", err)
	}

	s.logger.Info("Counter deactivated",
		zap.Int("counter_id", id),
	)

	return nil
}

func (s *Service) Verify(ctx context.Context, id int, verifiedBy int, req *model.VerificationRequest) (*model.Counter, error) {
	// Validate request
	if err := s.validator.ValidateVerification(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	verificationDate, _ := time.Parse("2006-01-02", req.VerificationDate)

	reqUpdate := &model.UpdateCounterRequest{
		VerificationDate: &req.VerificationDate,
	}

	counter, err := s.repo.Update(ctx, id, reqUpdate)
	if err != nil {
		return nil, fmt.Errorf("failed to verify counter: %w", err)
	}

	s.logger.Info("Counter verified",
		zap.Int("counter_id", id),
		zap.Int("verified_by", verifiedBy),
		zap.Time("verification_date", verificationDate),
	)

	return counter, nil
}

func (s *Service) GetStatus(ctx context.Context, id int) (*model.CounterStatus, error) {
	counter, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get counter: %w", err)
	}

	status := &model.CounterStatus{
		CounterID:         counter.ID,
		SerialNumber:      counter.SerialNumber,
		Type:             string(counter.Type),
		IsActive:         counter.IsActive,
		NeedsVerification: false,
	}

	// Calculate verification status
	var referenceDate time.Time
	if counter.VerificationDate != nil && !counter.VerificationDate.IsZero() {
		referenceDate = *counter.VerificationDate
	} else {
		referenceDate = counter.InstallationDate
	}

	nextVerification := model.CalculateNextVerification(referenceDate, 1460) // 4 years
	status.NextVerification = model.DaysUntilVerification(nextVerification)
	status.NeedsVerification = time.Now().After(nextVerification)

	return status, nil
}

func (s *Service) GetDashboard(ctx context.Context) (*model.Dashboard, error) {
	dashboard, err := s.repo.GetDashboardStats(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get dashboard: %w", err)
	}

	return dashboard, nil
}

func (s *Service) GetStatusesBySubscriber(ctx context.Context, subscriberID int) ([]model.CounterStatus, error) {
	counters, err := s.repo.GetBySubscriber(ctx, subscriberID)
	if err != nil {
		return nil, fmt.Errorf("failed to get counters: %w", err)
	}

	statuses := make([]model.CounterStatus, 0, len(counters))
	for _, counter := range counters {
		var referenceDate time.Time
		if counter.VerificationDate != nil && !counter.VerificationDate.IsZero() {
			referenceDate = *counter.VerificationDate
		} else {
			referenceDate = counter.InstallationDate
		}

		nextVerification := model.CalculateNextVerification(referenceDate, 1460)

		status := model.CounterStatus{
			CounterID:         counter.ID,
			SerialNumber:      counter.SerialNumber,
			Type:             string(counter.Type),
			IsActive:         counter.IsActive,
			NextVerification: model.DaysUntilVerification(nextVerification),
			NeedsVerification: time.Now().After(nextVerification),
		}

		statuses = append(statuses, status)
	}

	return statuses, nil
}
