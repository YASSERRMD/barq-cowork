package service

import (
	"context"
	"time"

	"github.com/barq-cowork/barq-cowork/internal/domain"
	"github.com/google/uuid"
)

// ScheduleRepository is the minimal store interface for schedules.
type ScheduleRepository interface {
	Create(ctx context.Context, s *domain.Schedule) error
	GetByID(ctx context.Context, id string) (*domain.Schedule, error)
	ListByProject(ctx context.Context, projectID string) ([]*domain.Schedule, error)
	List(ctx context.Context) ([]*domain.Schedule, error)
	Update(ctx context.Context, s *domain.Schedule) error
	Delete(ctx context.Context, id string) error
}

// ScheduleService manages recurring task schedules.
type ScheduleService struct {
	repo        ScheduleRepository
	projectRepo interface {
		GetByID(ctx context.Context, id string) (*domain.Project, error)
	}
}

func NewScheduleService(repo ScheduleRepository, projectRepo interface {
	GetByID(ctx context.Context, id string) (*domain.Project, error)
}) *ScheduleService {
	return &ScheduleService{repo: repo, projectRepo: projectRepo}
}

func (s *ScheduleService) Create(ctx context.Context,
	projectID, name, description, cronExpr, taskTitle, taskDesc, providerID string,
	enabled bool,
) (*domain.Schedule, error) {
	if _, err := s.projectRepo.GetByID(ctx, projectID); err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	sched := &domain.Schedule{
		ID:          uuid.NewString(),
		ProjectID:   projectID,
		Name:        name,
		Description: description,
		CronExpr:    cronExpr,
		TaskTitle:   taskTitle,
		TaskDesc:    taskDesc,
		ProviderID:  providerID,
		Enabled:     enabled,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.Create(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *ScheduleService) Get(ctx context.Context, id string) (*domain.Schedule, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *ScheduleService) ListByProject(ctx context.Context, projectID string) ([]*domain.Schedule, error) {
	return s.repo.ListByProject(ctx, projectID)
}

func (s *ScheduleService) List(ctx context.Context) ([]*domain.Schedule, error) {
	return s.repo.List(ctx)
}

func (s *ScheduleService) Update(ctx context.Context,
	id, name, description, cronExpr, taskTitle, taskDesc, providerID string,
	enabled bool,
) (*domain.Schedule, error) {
	sched, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	sched.Name = name
	sched.Description = description
	sched.CronExpr = cronExpr
	sched.TaskTitle = taskTitle
	sched.TaskDesc = taskDesc
	sched.ProviderID = providerID
	sched.Enabled = enabled
	sched.UpdatedAt = time.Now().UTC()
	if err := s.repo.Update(ctx, sched); err != nil {
		return nil, err
	}
	return sched, nil
}

func (s *ScheduleService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}
