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

type RecurrenceHandler struct {
    usecase recurrenceusecase.Usecase
}

func NewRecurrenceHandler(uc recurrenceusecase.Usecase) *RecurrenceHandler {
    return &RecurrenceHandler{usecase: uc}
}

// setRuleRequest is what the client sends.
type setRuleRequest struct {
    RuleType      string    `json:"rule_type"`
    IntervalDays  *int      `json:"interval_days,omitempty"`
    MonthDay      *int      `json:"month_day,omitempty"`
    SpecificDates []string  `json:"specific_dates,omitempty"` // "YYYY-MM-DD"
    DayParity     *string   `json:"day_parity,omitempty"`
    StartDate     string    `json:"start_date"`               // "YYYY-MM-DD"
    EndDate       *string   `json:"end_date,omitempty"`
}

type ruleDTO struct {
    ID            int64      `json:"id"`
    TaskID        int64      `json:"task_id"`
    RuleType      string     `json:"rule_type"`
    IntervalDays  *int       `json:"interval_days,omitempty"`
    MonthDay      *int       `json:"month_day,omitempty"`
    SpecificDates []string   `json:"specific_dates,omitempty"`
    DayParity     *string    `json:"day_parity,omitempty"`
    StartDate     string     `json:"start_date"`
    EndDate       *string    `json:"end_date,omitempty"`
    CreatedAt     time.Time  `json:"created_at"`
    UpdatedAt     time.Time  `json:"updated_at"`
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
        writeUsecaseRecurrenceError(w, err)
        return
    }

    writeJSON(w, http.StatusOK, toRuleDTO(rule))
}

func (h *RecurrenceHandler) GetRule(w http.ResponseWriter, r *http.Request) {
    taskID, err := getIDFromRequest(r)
    if err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    rule, err := h.usecase.GetRule(r.Context(), taskID)
    if err != nil {
        writeUsecaseRecurrenceError(w, err)
        return
    }
    writeJSON(w, http.StatusOK, toRuleDTO(rule))
}

func (h *RecurrenceHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
    taskID, err := getIDFromRequest(r)
    if err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    if err := h.usecase.DeleteRule(r.Context(), taskID); err != nil {
        writeUsecaseRecurrenceError(w, err)
        return
    }
    w.WriteHeader(http.StatusNoContent)
}

func (h *RecurrenceHandler) Generate(w http.ResponseWriter, r *http.Request) {
    taskID, err := getIDFromRequest(r)
    if err != nil {
        writeError(w, http.StatusBadRequest, err)
        return
    }
    horizon := time.Now().UTC().AddDate(0, 0, 30)
    tasks, err := h.usecase.GenerateOccurrences(r.Context(), taskID, horizon)
    if err != nil {
        writeUsecaseRecurrenceError(w, err)
        return
    }
    dtos := make([]taskDTO, 0, len(tasks))
    for _, t := range tasks {
        dtos = append(dtos, newTaskDTO(t))
    }
    writeJSON(w, http.StatusOK, dtos)
}

func toSetRuleInput(req setRuleRequest) (recurrenceusecase.SetRuleInput, error) {
    const layout = "2006-01-02"
    startDate, err := time.Parse(layout, req.StartDate)
    if err != nil {
        return recurrenceusecase.SetRuleInput{}, errors.New("start_date must be YYYY-MM-DD")
    }

    input := recurrenceusecase.SetRuleInput{
        RuleType:     recurrencedomain.RuleType(req.RuleType),
        IntervalDays: req.IntervalDays,
        MonthDay:     req.MonthDay,
        StartDate:    startDate,
    }

    if req.EndDate != nil {
        ed, err := time.Parse(layout, *req.EndDate)
        if err != nil {
            return recurrenceusecase.SetRuleInput{}, errors.New("end_date must be YYYY-MM-DD")
        }
        input.EndDate = &ed
    }

    if req.DayParity != nil {
        dp := recurrencedomain.DayParity(*req.DayParity)
        input.DayParity = &dp
    }

    for _, ds := range req.SpecificDates {
        d, err := time.Parse(layout, ds)
        if err != nil {
            return recurrenceusecase.SetRuleInput{}, errors.New("specific_dates must be YYYY-MM-DD")
        }
        input.SpecificDates = append(input.SpecificDates, d)
    }

    return input, nil
}

func writeUsecaseRecurrenceError(w http.ResponseWriter, err error) {
    switch {
    case errors.Is(err, recurrencedomain.ErrNotFound):
        writeError(w, http.StatusNotFound, err)
    case errors.Is(err, recurrenceusecase.ErrInvalidInput):
        writeError(w, http.StatusBadRequest, err)
    case errors.Is(err, taskdomain.ErrNotFound):
        writeError(w, http.StatusNotFound, err)
    default:
        writeError(w, http.StatusInternalServerError, err)
    }
}