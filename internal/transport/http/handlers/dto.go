package handlers

import (
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
)

type taskMutationDTO struct {
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Status      taskdomain.Status `json:"status"`
}

type taskDTO struct {
    ID            int64             `json:"id"`
    Title         string            `json:"title"`
    Description   string            `json:"description"`
    Status        taskdomain.Status `json:"status"`
    CreatedAt     time.Time         `json:"created_at"`
    UpdatedAt     time.Time         `json:"updated_at"`
    ParentTaskID  *int64            `json:"parent_task_id,omitempty"`
    ScheduledDate *time.Time        `json:"scheduled_date,omitempty"`
}

func newTaskDTO(task *taskdomain.Task) taskDTO {
    return taskDTO{
        ID:            task.ID,
        Title:         task.Title,
        Description:   task.Description,
        Status:        task.Status,
        CreatedAt:     task.CreatedAt,
        UpdatedAt:     task.UpdatedAt,
        ParentTaskID:  task.ParentTaskID,
        ScheduledDate: task.ScheduledDate,
    }
}
