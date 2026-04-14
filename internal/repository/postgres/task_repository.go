package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

type Repository struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		INSERT INTO tasks (title, description, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, title, description, status, parent_task_id, scheduled_date, created_at, updated_at
	`
	row := r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status, task.CreatedAt, task.UpdatedAt,
	)
	return scanTask(row)
}

func (r *Repository) GetByID(ctx context.Context, id int64) (*taskdomain.Task, error) {
	const query = `
		SELECT id, title, description, status, parent_task_id, scheduled_date, created_at, updated_at
		FROM tasks
		WHERE id = $1
	`
	found, err := scanTask(r.pool.QueryRow(ctx, query, id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return found, nil
}

func (r *Repository) Update(ctx context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	const query = `
		UPDATE tasks
		SET title       = $1,
		    description = $2,
		    status      = $3,
		    updated_at  = $4
		WHERE id = $5
		RETURNING id, title, description, status, parent_task_id, scheduled_date, created_at, updated_at
	`
	updated, err := scanTask(r.pool.QueryRow(ctx, query,
		task.Title, task.Description, task.Status, task.UpdatedAt, task.ID,
	))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, taskdomain.ErrNotFound
		}
		return nil, err
	}
	return updated, nil
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM tasks WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return taskdomain.ErrNotFound
	}
	return nil
}

// List returns tasks filtered by the given filter.
func (r *Repository) List(ctx context.Context, filter taskusecase.ListFilter) ([]taskdomain.Task, error) {
	query := `
		SELECT id, title, description, status, parent_task_id, scheduled_date, created_at, updated_at
		FROM tasks
	`
	var (
		conditions []string
		args       []any
		argIdx     = 1
	)

	if filter.OnlyTemplates {
		conditions = append(conditions, "parent_task_id IS NULL")
	}
	if filter.ParentID != nil {
		conditions = append(conditions, fmt.Sprintf("parent_task_id = $%d", argIdx))
		args = append(args, *filter.ParentID)
		argIdx++
	}

	if len(conditions) > 0 {
		query += " WHERE " + joinConditions(conditions)
	}
	query += " ORDER BY id DESC"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := make([]taskdomain.Task, 0)
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, *task)
	}
	return tasks, rows.Err()
}

func joinConditions(conditions []string) string {
	result := conditions[0]
	for _, c := range conditions[1:] {
		result += " AND " + c
	}
	return result
}

type taskScanner interface {
	Scan(dest ...any) error
}

func scanTask(scanner taskScanner) (*taskdomain.Task, error) {
	var (
		task          taskdomain.Task
		status        string
		parentTaskID  *int64
		scheduledDate *time.Time
	)
	if err := scanner.Scan(
		&task.ID,
		&task.Title,
		&task.Description,
		&status,
		&parentTaskID,
		&scheduledDate,
		&task.CreatedAt,
		&task.UpdatedAt,
	); err != nil {
		return nil, err
	}
	task.Status = taskdomain.Status(status)
	task.ParentTaskID = parentTaskID
	task.ScheduledDate = scheduledDate
	return &task, nil
}