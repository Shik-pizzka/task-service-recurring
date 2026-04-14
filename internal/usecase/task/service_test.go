package task_test

import (
	"context"
	"errors"
	"testing"
	"time"

	taskdomain "example.com/taskservice/internal/domain/task"
	taskusecase "example.com/taskservice/internal/usecase/task"
)

//mock repository

type mockRepo struct {
	tasks  map[int64]*taskdomain.Task
	nextID int64
}

func newMockRepo() *mockRepo {
	return &mockRepo{tasks: make(map[int64]*taskdomain.Task), nextID: 1}
}

func (m *mockRepo) Create(_ context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	task.ID = m.nextID
	m.nextID++
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockRepo) GetByID(_ context.Context, id int64) (*taskdomain.Task, error) {
	t, ok := m.tasks[id]
	if !ok {
		return nil, taskdomain.ErrNotFound
	}
	return t, nil
}

func (m *mockRepo) Update(_ context.Context, task *taskdomain.Task) (*taskdomain.Task, error) {
	if _, ok := m.tasks[task.ID]; !ok {
		return nil, taskdomain.ErrNotFound
	}
	m.tasks[task.ID] = task
	return task, nil
}

func (m *mockRepo) Delete(_ context.Context, id int64) error {
	if _, ok := m.tasks[id]; !ok {
		return taskdomain.ErrNotFound
	}
	delete(m.tasks, id)
	return nil
}

func (m *mockRepo) List(_ context.Context) ([]taskdomain.Task, error) {
	out := make([]taskdomain.Task, 0, len(m.tasks))
	for _, t := range m.tasks {
		out = append(out, *t)
	}
	return out, nil
}

//tests

func TestService_Create_OK(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	task, err := svc.Create(context.Background(), taskusecase.CreateInput{
		Title:       "Test task",
		Description: "desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if task.Status != taskdomain.StatusNew {
		t.Fatalf("expected status 'new', got %q", task.Status)
	}
}

func TestService_Create_EmptyTitle(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	_, err := svc.Create(context.Background(), taskusecase.CreateInput{Title: "  "})
	if err == nil {
		t.Fatal("expected error for empty title")
	}
	if !errors.Is(err, taskusecase.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Create_InvalidStatus(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	_, err := svc.Create(context.Background(), taskusecase.CreateInput{
		Title:  "Task",
		Status: "unknown",
	})
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
}

func TestService_GetByID_NotFound(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	_, err := svc.GetByID(context.Background(), 999)
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestService_GetByID_InvalidID(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	_, err := svc.GetByID(context.Background(), 0)
	if !errors.Is(err, taskusecase.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestService_Update_OK(t *testing.T) {
	repo := newMockRepo()
	svc := taskusecase.NewService(repo)
	created, _ := svc.Create(context.Background(), taskusecase.CreateInput{Title: "Original"})

	updated, err := svc.Update(context.Background(), created.ID, taskusecase.UpdateInput{
		Title:  "Updated",
		Status: taskdomain.StatusInProgress,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Updated" {
		t.Fatalf("expected title 'Updated', got %q", updated.Title)
	}
}

func TestService_Update_TitleWhitespaceTrimmed(t *testing.T) {
	repo := newMockRepo()
	svc := taskusecase.NewService(repo)
	created, _ := svc.Create(context.Background(), taskusecase.CreateInput{Title: "Task"})

	_, err := svc.Update(context.Background(), created.ID, taskusecase.UpdateInput{
		Title:  "   ",
		Status: taskdomain.StatusNew,
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only title")
	}
}

func TestService_Delete_OK(t *testing.T) {
	repo := newMockRepo()
	svc := taskusecase.NewService(repo)
	created, _ := svc.Create(context.Background(), taskusecase.CreateInput{Title: "Task"})

	if err := svc.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := svc.GetByID(context.Background(), created.ID)
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Fatal("expected task to be deleted")
	}
}

func TestService_Delete_NotFound(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	err := svc.Delete(context.Background(), 999)
	if !errors.Is(err, taskdomain.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestService_List_Empty(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	tasks, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tasks) != 0 {
		t.Fatalf("expected empty list, got %d tasks", len(tasks))
	}
}

func TestService_Create_SetsTimestamps(t *testing.T) {
	svc := taskusecase.NewService(newMockRepo())
	before := time.Now().UTC().Add(-time.Second)
	task, _ := svc.Create(context.Background(), taskusecase.CreateInput{Title: "Task"})
	after := time.Now().UTC().Add(time.Second)

	if task.CreatedAt.Before(before) || task.CreatedAt.After(after) {
		t.Fatalf("created_at out of expected range: %v", task.CreatedAt)
	}
	if !task.CreatedAt.Equal(task.UpdatedAt) {
		t.Fatal("expected created_at == updated_at on creation")
	}
}