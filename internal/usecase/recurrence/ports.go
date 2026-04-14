package recurrence

import (
    "context"
    "time"

    recurrencedomain "example.com/taskservice/internal/domain/recurrence"
    taskdomain "example.com/taskservice/internal/domain/task"
)

type RuleRepository interface {
    Create(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error)
    GetByTaskID(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error)
    Update(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error)
    Delete(ctx context.Context, taskID int64) error
    ListAll(ctx context.Context) ([]*recurrencedomain.Rule, error)
}

type OccurrenceRepository interface {
    ExistingDates(ctx context.Context, ruleID int64) (map[time.Time]struct{}, error)
    CreateBatch(ctx context.Context, ruleID int64, parent *taskdomain.Task, dates []time.Time) ([]*taskdomain.Task, error)
}

type TaskRepository interface {
    GetByID(ctx context.Context, id int64) (*taskdomain.Task, error)
}

type Usecase interface {
    SetRule(ctx context.Context, taskID int64, input SetRuleInput) (*recurrencedomain.Rule, error)
    GetRule(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error)
    DeleteRule(ctx context.Context, taskID int64) error
    GenerateOccurrences(ctx context.Context, taskID int64, horizon time.Time) ([]*taskdomain.Task, error)
    GenerateAll(ctx context.Context, horizon time.Time) error
}

type SetRuleInput struct {
    RuleType      recurrencedomain.RuleType
    IntervalDays  *int
    MonthDay      *int
    SpecificDates []time.Time
    DayParity     *recurrencedomain.DayParity
    StartDate     time.Time
    EndDate       *time.Time
}