package recurrence

import (
    "errors"
    "fmt"
    "time"
)

type RuleType string

const (
    RuleTypeDaily         RuleType = "daily"
    RuleTypeMonthly       RuleType = "monthly"
    RuleTypeSpecificDates RuleType = "specific_dates"
    RuleTypeEvenOdd       RuleType = "even_odd"
)

func (r RuleType) Valid() bool {
    switch r {
    case RuleTypeDaily, RuleTypeMonthly, RuleTypeSpecificDates, RuleTypeEvenOdd:
        return true
    }
    return false
}

type DayParity string

const (
    DayParityEven DayParity = "even"
    DayParityOdd  DayParity = "odd"
)

func (d DayParity) Valid() bool { return d == DayParityEven || d == DayParityOdd }

var ErrNotFound = errors.New("recurrence rule not found")

type Rule struct {
    ID            int64
    TaskID        int64
    RuleType      RuleType
    IntervalDays  *int
    MonthDay      *int
    SpecificDates []time.Time
    DayParity     *DayParity
    StartDate     time.Time
    EndDate       *time.Time
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

func (r *Rule) Validate() error {
    if !r.RuleType.Valid() {
        return fmt.Errorf("invalid rule_type %q", r.RuleType)
    }
    if r.StartDate.IsZero() {
        return errors.New("start_date is required")
    }
    if r.EndDate != nil && !r.EndDate.After(r.StartDate) {
        return errors.New("end_date must be after start_date")
    }
    switch r.RuleType {
    case RuleTypeDaily:
        if r.IntervalDays == nil || *r.IntervalDays < 1 {
            return errors.New("interval_days must be >= 1 for daily rules")
        }
    case RuleTypeMonthly:
        if r.MonthDay == nil || *r.MonthDay < 1 || *r.MonthDay > 30 {
            return errors.New("month_day must be between 1 and 30")
        }
    case RuleTypeSpecificDates:
        if len(r.SpecificDates) == 0 {
            return errors.New("specific_dates must not be empty")
        }
    case RuleTypeEvenOdd:
        if r.DayParity == nil || !r.DayParity.Valid() {
            return errors.New("day_parity must be 'even' or 'odd'")
        }
    }
    return nil
}

// Occurrences returns all dates in [from, to] that match the rule.
func (r *Rule) Occurrences(from, to time.Time) []time.Time {
    from, to = trunc(from), trunc(to)
    if s := trunc(r.StartDate); from.Before(s) {
        from = s
    }
    if r.EndDate != nil {
        if e := trunc(*r.EndDate); to.After(e) {
            to = e
        }
    }
    if from.After(to) {
        return nil
    }
    switch r.RuleType {
    case RuleTypeDaily:
        return r.daily(from, to)
    case RuleTypeMonthly:
        return r.monthly(from, to)
    case RuleTypeSpecificDates:
        return r.specific(from, to)
    case RuleTypeEvenOdd:
        return r.evenOdd(from, to)
    }
    return nil
}

func (r *Rule) daily(from, to time.Time) []time.Time {
    interval := time.Duration(*r.IntervalDays) * 24 * time.Hour
    start := trunc(r.StartDate)
    cur := start
    if cur.Before(from) {
        diff := from.Sub(cur)
        steps := int(diff / interval)
        cur = start.Add(time.Duration(steps) * interval)
        if cur.Before(from) {
            cur = cur.Add(interval)
        }
    }
    var out []time.Time
    for !cur.After(to) {
        out = append(out, cur)
        cur = cur.Add(interval)
    }
    return out
}

func (r *Rule) monthly(from, to time.Time) []time.Time {
    day := *r.MonthDay
    var out []time.Time
    cur := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
    end := time.Date(to.Year(), to.Month(), 1, 0, 0, 0, 0, time.UTC)
    for !cur.After(end) {
        t := time.Date(cur.Year(), cur.Month(), day, 0, 0, 0, 0, time.UTC)
        // If day overflows (e.g. Feb 30), Go normalises to next month — skip it.
        if t.Month() == cur.Month() && !t.Before(from) && !t.After(to) {
            out = append(out, t)
        }
        cur = cur.AddDate(0, 1, 0)
    }
    return out
}

func (r *Rule) specific(from, to time.Time) []time.Time {
    var out []time.Time
    for _, d := range r.SpecificDates {
        d = trunc(d)
        if !d.Before(from) && !d.After(to) {
            out = append(out, d)
        }
    }
    return out
}

func (r *Rule) evenOdd(from, to time.Time) []time.Time {
    var out []time.Time
    for cur := from; !cur.After(to); cur = cur.AddDate(0, 0, 1) {
        even := cur.Day()%2 == 0
        if (*r.DayParity == DayParityEven && even) || (*r.DayParity == DayParityOdd && !even) {
            out = append(out, cur)
        }
    }
    return out
}

func trunc(t time.Time) time.Time {
    return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}