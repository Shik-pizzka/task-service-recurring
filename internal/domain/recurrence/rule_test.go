package recurrence_test

import (
	"testing"
	"time"

	"example.com/taskservice/internal/domain/recurrence"
)

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func intPtr(v int) *int { return &v }

func dpPtr(v recurrence.DayParity) *recurrence.DayParity { return &v }

//Validate

func TestRule_Validate_Daily_OK(t *testing.T) {
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(2),
		StartDate:    date(2026, 1, 1),
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRule_Validate_Daily_MissingInterval(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeDaily,
		StartDate: date(2026, 1, 1),
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for missing interval_days")
	}
}

func TestRule_Validate_Monthly_OK(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeMonthly,
		MonthDay:  intPtr(15),
		StartDate: date(2026, 1, 1),
	}
	if err := r.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRule_Validate_Monthly_DayOutOfRange(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeMonthly,
		MonthDay:  intPtr(31),
		StartDate: date(2026, 1, 1),
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for month_day=31")
	}
}

func TestRule_Validate_EndBeforeStart(t *testing.T) {
	end := date(2025, 12, 31)
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(1),
		StartDate:    date(2026, 1, 1),
		EndDate:      &end,
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error when end_date is before start_date")
	}
}

func TestRule_Validate_InvalidRuleType(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  "weekly",
		StartDate: date(2026, 1, 1),
	}
	if err := r.Validate(); err == nil {
		t.Fatal("expected error for unknown rule_type")
	}
}

//Daily

func TestRule_Daily_EveryDay(t *testing.T) {
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(1),
		StartDate:    date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 4, 1), date(2026, 4, 5))
	want := []time.Time{
		date(2026, 4, 1),
		date(2026, 4, 2),
		date(2026, 4, 3),
		date(2026, 4, 4),
		date(2026, 4, 5),
	}
	assertDates(t, want, got)
}

func TestRule_Daily_EveryTwoDays(t *testing.T) {
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(2),
		StartDate:    date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 4, 1), date(2026, 4, 7))
	want := []time.Time{
		date(2026, 4, 1),
		date(2026, 4, 3),
		date(2026, 4, 5),
		date(2026, 4, 7),
	}
	assertDates(t, want, got)
}

func TestRule_Daily_FromAfterStart(t *testing.T) {
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(3),
		StartDate:    date(2026, 4, 1),
	}
	// Sequence: 1, 4, 7, 10, 13 — from day 5: 7, 10, 13
	got := r.Occurrences(date(2026, 4, 5), date(2026, 4, 14))
	want := []time.Time{
		date(2026, 4, 7),
		date(2026, 4, 10),
		date(2026, 4, 13),
	}
	assertDates(t, want, got)
}

//Monthly

func TestRule_Monthly_Normal(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeMonthly,
		MonthDay:  intPtr(15),
		StartDate: date(2026, 1, 1),
	}
	got := r.Occurrences(date(2026, 1, 1), date(2026, 4, 30))
	want := []time.Time{
		date(2026, 1, 15),
		date(2026, 2, 15),
		date(2026, 3, 15),
		date(2026, 4, 15),
	}
	assertDates(t, want, got)
}

func TestRule_Monthly_Feb30Skipped(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeMonthly,
		MonthDay:  intPtr(30),
		StartDate: date(2026, 1, 1),
	}
	got := r.Occurrences(date(2026, 1, 1), date(2026, 3, 31))
	want := []time.Time{
		date(2026, 1, 30),
		// Feb 30 skipped — does not exist
		date(2026, 3, 30),
	}
	assertDates(t, want, got)
}

//Specific dates

func TestRule_SpecificDates(t *testing.T) {
	r := recurrence.Rule{
		RuleType: recurrence.RuleTypeSpecificDates,
		SpecificDates: []time.Time{
			date(2026, 5, 1),
			date(2026, 5, 10),
			date(2026, 5, 25),
		},
		StartDate: date(2026, 5, 1),
	}
	got := r.Occurrences(date(2026, 5, 1), date(2026, 5, 31))
	want := []time.Time{
		date(2026, 5, 1),
		date(2026, 5, 10),
		date(2026, 5, 25),
	}
	assertDates(t, want, got)
}

func TestRule_SpecificDates_OutOfRange(t *testing.T) {
	r := recurrence.Rule{
		RuleType: recurrence.RuleTypeSpecificDates,
		SpecificDates: []time.Time{
			date(2026, 4, 1),
			date(2026, 5, 10),
		},
		StartDate: date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 5, 1), date(2026, 5, 31))
	want := []time.Time{date(2026, 5, 10)}
	assertDates(t, want, got)
}

//Even/odd

func TestRule_EvenDays(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeEvenOdd,
		DayParity: dpPtr(recurrence.DayParityEven),
		StartDate: date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 4, 1), date(2026, 4, 6))
	want := []time.Time{
		date(2026, 4, 2),
		date(2026, 4, 4),
		date(2026, 4, 6),
	}
	assertDates(t, want, got)
}

func TestRule_OddDays(t *testing.T) {
	r := recurrence.Rule{
		RuleType:  recurrence.RuleTypeEvenOdd,
		DayParity: dpPtr(recurrence.DayParityOdd),
		StartDate: date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 4, 1), date(2026, 4, 6))
	want := []time.Time{
		date(2026, 4, 1),
		date(2026, 4, 3),
		date(2026, 4, 5),
	}
	assertDates(t, want, got)
}

//Edge cases

func TestRule_EndDate_Clamps(t *testing.T) {
	end := date(2026, 4, 3)
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(1),
		StartDate:    date(2026, 4, 1),
		EndDate:      &end,
	}
	got := r.Occurrences(date(2026, 4, 1), date(2026, 4, 10))
	want := []time.Time{
		date(2026, 4, 1),
		date(2026, 4, 2),
		date(2026, 4, 3),
	}
	assertDates(t, want, got)
}

func TestRule_Occurrences_EmptyWhenFromAfterTo(t *testing.T) {
	r := recurrence.Rule{
		RuleType:     recurrence.RuleTypeDaily,
		IntervalDays: intPtr(1),
		StartDate:    date(2026, 4, 1),
	}
	got := r.Occurrences(date(2026, 4, 10), date(2026, 4, 1))
	if len(got) != 0 {
		t.Fatalf("expected empty slice, got %v", got)
	}
}

//helpers

func assertDates(t *testing.T, want, got []time.Time) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: want %d, got %d\nwant: %v\ngot:  %v", len(want), len(got), want, got)
	}
	for i := range want {
		if !want[i].Equal(got[i]) {
			t.Errorf("index %d: want %s, got %s", i, want[i].Format("2006-01-02"), got[i].Format("2006-01-02"))
		}
	}
}