package recurrence

import (
    "context"
    "fmt"
    "time"

    recurrencedomain "example.com/taskservice/internal/domain/recurrence"
    taskdomain "example.com/taskservice/internal/domain/task"
)

const GenerationHorizonDays = 30

type Service struct {
    ruleRepo       RuleRepository
    occurrenceRepo OccurrenceRepository
    taskRepo       TaskRepository
    now            func() time.Time
}

func NewService(ruleRepo RuleRepository, occurrenceRepo OccurrenceRepository, taskRepo TaskRepository) *Service {
    return &Service{
        ruleRepo:       ruleRepo,
        occurrenceRepo: occurrenceRepo,
        taskRepo:       taskRepo,
        now:            func() time.Time { return time.Now().UTC() },
    }
}

func (s *Service) SetRule(ctx context.Context, taskID int64, input SetRuleInput) (*recurrencedomain.Rule, error) {
    if taskID <= 0 {
        return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
    }
    if _, err := s.taskRepo.GetByID(ctx, taskID); err != nil {
        return nil, err
    }
    rule := &recurrencedomain.Rule{
        TaskID:        taskID,
        RuleType:      input.RuleType,
        IntervalDays:  input.IntervalDays,
        MonthDay:      input.MonthDay,
        SpecificDates: input.SpecificDates,
        DayParity:     input.DayParity,
        StartDate:     input.StartDate,
        EndDate:       input.EndDate,
    }
    if err := rule.Validate(); err != nil {
        return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err)
    }
    existing, err := s.ruleRepo.GetByTaskID(ctx, taskID)
    if err != nil && err != recurrencedomain.ErrNotFound {
        return nil, err
    }
    now := s.now()
    if existing != nil {
        rule.ID = existing.ID
        rule.CreatedAt = existing.CreatedAt
        rule.UpdatedAt = now
        return s.ruleRepo.Update(ctx, rule)
    }
    rule.CreatedAt = now
    rule.UpdatedAt = now
    return s.ruleRepo.Create(ctx, rule)
}

func (s *Service) GetRule(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error) {
    if taskID <= 0 {
        return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
    }
    return s.ruleRepo.GetByTaskID(ctx, taskID)
}

func (s *Service) DeleteRule(ctx context.Context, taskID int64) error {
    if taskID <= 0 {
        return fmt.Errorf("%w: id must be positive", ErrInvalidInput)
    }
    return s.ruleRepo.Delete(ctx, taskID)
}

func (s *Service) GenerateOccurrences(ctx context.Context, taskID int64, horizon time.Time) ([]*taskdomain.Task, error) {
    if taskID <= 0 {
        return nil, fmt.Errorf("%w: id must be positive", ErrInvalidInput)
    }
    rule, err := s.ruleRepo.GetByTaskID(ctx, taskID)
    if err != nil {
        return nil, err
    }
    parent, err := s.taskRepo.GetByID(ctx, taskID)
    if err != nil {
        return nil, err
    }
    return s.generate(ctx, rule, parent, horizon)
}

func (s *Service) GenerateAll(ctx context.Context, horizon time.Time) error {
    rules, err := s.ruleRepo.ListAll(ctx)
    if err != nil {
        return err
    }
    for _, rule := range rules {
        parent, err := s.taskRepo.GetByID(ctx, rule.TaskID)
        if err != nil {
            continue // parent was deleted outside of cascade
        }
        _, _ = s.generate(ctx, rule, parent, horizon)
    }
    return nil
}

func (s *Service) generate(ctx context.Context, rule *recurrencedomain.Rule, parent *taskdomain.Task, horizon time.Time) ([]*taskdomain.Task, error) {
    now := trunc(s.now())
    horizon = trunc(horizon)
    if horizon.Before(now) {
        horizon = now
    }
    candidates := rule.Occurrences(now, horizon)
    if len(candidates) == 0 {
        return nil, nil
    }
    existing, err := s.occurrenceRepo.ExistingDates(ctx, rule.ID)
    if err != nil {
        return nil, err
    }
    var toCreate []time.Time
    for _, d := range candidates {
        if _, ok := existing[d]; !ok {
            toCreate = append(toCreate, d)
        }
    }
    if len(toCreate) == 0 {
        return nil, nil
    }
    return s.occurrenceRepo.CreateBatch(ctx, rule.ID, parent, toCreate)
}

func trunc(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}