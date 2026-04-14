package postgres

import (
    "context"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"

    taskdomain "example.com/taskservice/internal/domain/task"
)

type OccurrenceRepository struct{ pool *pgxpool.Pool }

func NewOccurrenceRepository(pool *pgxpool.Pool) *OccurrenceRepository {
    return &OccurrenceRepository{pool: pool}
}

func (r *OccurrenceRepository) ExistingDates(ctx context.Context, ruleID int64) (map[time.Time]struct{}, error) {
    rows, err := r.pool.Query(ctx, `SELECT scheduled_date FROM recurrence_occurrences WHERE rule_id=$1`, ruleID)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    result := make(map[time.Time]struct{})
    for rows.Next() {
        var d time.Time
        if err := rows.Scan(&d); err != nil {
            return nil, err
        }
        result[trunc(d)] = struct{}{}
    }
    return result, rows.Err()
}

func trunc(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (r *OccurrenceRepository) CreateBatch(ctx context.Context, ruleID int64, parent *taskdomain.Task, dates []time.Time) ([]*taskdomain.Task, error) {
    tx, err := r.pool.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    now := time.Now().UTC()
    var created []*taskdomain.Task

    for _, d := range dates {
        d = trunc(d)
        var task taskdomain.Task
        var status string
        var parentID *int64
        var sched *time.Time
        err := tx.QueryRow(ctx, `
            INSERT INTO tasks (title,description,status,parent_task_id,scheduled_date,created_at,updated_at)
            VALUES ($1,$2,$3,$4,$5,$6,$7)
            RETURNING id,title,description,status,parent_task_id,scheduled_date,created_at,updated_at`,
            parent.Title, parent.Description, taskdomain.StatusNew, parent.ID, d, now, now,
        ).Scan(&task.ID, &task.Title, &task.Description, &status, &parentID, &sched, &task.CreatedAt, &task.UpdatedAt)
        if err != nil {
            return nil, err
        }
        task.Status = taskdomain.Status(status)
        task.ParentTaskID = parentID
        task.ScheduledDate = sched
        created = append(created, &task)

        if _, err := tx.Exec(ctx, `
            INSERT INTO recurrence_occurrences (rule_id,scheduled_date,task_id)
            VALUES ($1,$2,$3) ON CONFLICT (rule_id,scheduled_date) DO NOTHING`,
            ruleID, d, task.ID); err != nil {
            return nil, err
        }
    }
    return created, tx.Commit(ctx)
}