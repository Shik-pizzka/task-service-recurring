package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	recurrencedomain "example.com/taskservice/internal/domain/recurrence"
	taskdomain "example.com/taskservice/internal/domain/task"
	recurrenceusecase "example.com/taskservice/internal/usecase/recurrence"
)

// RecurrenceHandler handles all /tasks/{id}/recurrence* endpoints.
type RecurrenceHandler struct {
	usecase recurrenceusecase.Usecase
}

func NewRecurrenceHandler(uc recurrenceusecase.Usecase) *RecurrenceHandler {
	return &RecurrenceHandler{usecase: uc}
}

type setRuleRequest struct {
	RuleType      string   `json:"rule_type"`
	IntervalDays  *int     `json:"interval_days,omitempty"`
	MonthDay      *int     `json:"month_day,omitempty"`
	SpecificDates []string `json:"specific_dates,omitempty"`
	DayParity     *string  `json:"day_parity,omitempty"`
	StartDate     string   `json:"start_date"`
	EndDate       *string  `json:"end_date,omitempty"`
}

type ruleDTO struct {
	ID            int64     `json:"id"`
	TaskID        int64     `json:"task_id"`
	RuleType      string    `json:"rule_type"`
	IntervalDays  *int      `json:"interval_days,omitempty"`
	MonthDay      *int      `json:"month_day,omitempty"`
	SpecificDates []string  `json:"specific_dates,omitempty"`
	DayParity     *string   `json:"day_parity,omitempty"`
	StartDate     string    `json:"start_date"`
	EndDate       *string   `json:"end_date,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func toRuleDTO(r *recurrencedomain.Rule) ruleDTO {
	dto := ruleDTO{
		ID:           r.ID,
		TaskID:       r.TaskID,
		RuleType:     string(r.RuleType),
		IntervalDays: r.IntervalDays,
		MonthDay:     r.MonthDay,
		StartDate:    r.StartDate.Format("2006-01-02"),
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
	if r.DayParity != nil {
		s := string(*r.DayParity)
		dto.DayParity = &s
	}
	if r.EndDate != nil {
		s := r.EndDate.Format("2006-01-02")
		dto.EndDate = &s
	}
	for _, d := range r.SpecificDates {
		dto.SpecificDates = append(dto.SpecificDates, d.Format("2006-01-02"))
	}
	return dto
}

// SetRule — PUT /api/v1/tasks/{id}/recurrence
func (h *RecurrenceHandler) SetRule(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	var req setRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	input, err := toSetRuleInput(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := h.usecase.SetRule(r.Context(), taskID, input)
	if err != nil {
		writeRecurrenceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toRuleDTO(rule))
}

// GetRule — GET /api/v1/tasks/{id}/recurrence
func (h *RecurrenceHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	rule, err := h.usecase.GetRule(r.Context(), taskID)
	if err != nil {
		writeRecurrenceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toRuleDTO(rule))
}

// DeleteRule — DELETE /api/v1/tasks/{id}/recurrence
func (h *RecurrenceHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := h.usecase.DeleteRule(r.Context(), taskID); err != nil {
		writeRecurrenceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Generate — POST /api/v1/tasks/{id}/recurrence/generate
// Generates child tasks for the next 30 days. Idempotent.
func (h *RecurrenceHandler) Generate(w http.ResponseWriter, r *http.Request) {
	taskID, err := getIDFromRequest(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	horizon := time.Now().UTC().AddDate(0, 0, recurrenceusecase.GenerationHorizonDays)
	tasks, err := h.usecase.GenerateOccurrences(r.Context(), taskID, horizon)
	if err != nil {
		writeRecurrenceError(w, err)
		return
	}
	dtos := make([]taskDTO, 0, len(tasks))
	for _, t := range tasks {
		dtos = append(dtos, newTaskDTO(t))
	}
	writeJSON(w, http.StatusOK, dtos)
}

const dateLayout = "2006-01-02"

func toSetRuleInput(req setRuleRequest) (recurrenceusecase.SetRuleInput, error) {
	startDate, err := time.Parse(dateLayout, req.StartDate)
	if err != nil {
		return recurrenceusecase.SetRuleInput{}, errors.New("start_date must be in YYYY-MM-DD format")
	}
	input := recurrenceusecase.SetRuleInput{
		RuleType:     recurrencedomain.RuleType(req.RuleType),
		IntervalDays: req.IntervalDays,
		MonthDay:     req.MonthDay,
		StartDate:    startDate,
	}
	if req.EndDate != nil {
		ed, err := time.Parse(dateLayout, *req.EndDate)
		if err != nil {
			return recurrenceusecase.SetRuleInput{}, errors.New("end_date must be in YYYY-MM-DD format")
		}
		input.EndDate = &ed
	}
	if req.DayParity != nil {
		dp := recurrencedomain.DayParity(*req.DayParity)
		input.DayParity = &dp
	}
	for _, ds := range req.SpecificDates {
		d, err := time.Parse(dateLayout, ds)
		if err != nil {
			return recurrenceusecase.SetRuleInput{}, errors.New("each date in specific_dates must be in YYYY-MM-DD format")
		}
		input.SpecificDates = append(input.SpecificDates, d)
	}
	return input, nil
}

func writeRecurrenceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, recurrencedomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, taskdomain.ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, recurrenceusecase.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err)
	default:
		writeError(w, http.StatusInternalServerError, err)
	}
}