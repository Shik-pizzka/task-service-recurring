package postgres

import (
    "context"
    "errors"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/jackc/pgx/v5/pgxpool"

    recurrencedomain "example.com/taskservice/internal/domain/recurrence"
)

type RuleRepository struct{ pool *pgxpool.Pool }

func NewRuleRepository(pool *pgxpool.Pool) *RuleRepository { return &RuleRepository{pool: pool} }

func (r *RuleRepository) Create(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error) {
    const q = `
        INSERT INTO recurrence_rules
            (task_id,rule_type,interval_days,month_day,specific_dates,day_parity,start_date,end_date,created_at,updated_at)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
        RETURNING id,task_id,rule_type,interval_days,month_day,specific_dates,day_parity,start_date,end_date,created_at,updated_at`
    row := r.pool.QueryRow(ctx, q,
        rule.TaskID, string(rule.RuleType), rule.IntervalDays, rule.MonthDay,
        rule.SpecificDates, dpStr(rule.DayParity), rule.StartDate, rule.EndDate,
        rule.CreatedAt, rule.UpdatedAt)
    return scanRule(row)
}

func (r *RuleRepository) GetByTaskID(ctx context.Context, taskID int64) (*recurrencedomain.Rule, error) {
    const q = `SELECT id,task_id,rule_type,interval_days,month_day,specific_dates,day_parity,start_date,end_date,created_at,updated_at
               FROM recurrence_rules WHERE task_id=$1`
    rule, err := scanRule(r.pool.QueryRow(ctx, q, taskID))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, recurrencedomain.ErrNotFound
        }
        return nil, err
    }
    return rule, nil
}

func (r *RuleRepository) Update(ctx context.Context, rule *recurrencedomain.Rule) (*recurrencedomain.Rule, error) {
    const q = `UPDATE recurrence_rules
               SET rule_type=$1,interval_days=$2,month_day=$3,specific_dates=$4,day_parity=$5,start_date=$6,end_date=$7,updated_at=$8
               WHERE id=$9
               RETURNING id,task_id,rule_type,interval_days,month_day,specific_dates,day_parity,start_date,end_date,created_at,updated_at`
    updated, err := scanRule(r.pool.QueryRow(ctx, q,
        string(rule.RuleType), rule.IntervalDays, rule.MonthDay,
        rule.SpecificDates, dpStr(rule.DayParity), rule.StartDate, rule.EndDate, rule.UpdatedAt, rule.ID))
    if err != nil {
        if errors.Is(err, pgx.ErrNoRows) {
            return nil, recurrencedomain.ErrNotFound
        }
        return nil, err
    }
    return updated, nil
}

func (r *RuleRepository) Delete(ctx context.Context, taskID int64) error {
    result, err := r.pool.Exec(ctx, `DELETE FROM recurrence_rules WHERE task_id=$1`, taskID)
    if err != nil {
        return err
    }
    if result.RowsAffected() == 0 {
        return recurrencedomain.ErrNotFound
    }
    return nil
}

func (r *RuleRepository) ListAll(ctx context.Context) ([]*recurrencedomain.Rule, error) {
    const q = `SELECT id,task_id,rule_type,interval_days,month_day,specific_dates,day_parity,start_date,end_date,created_at,updated_at
               FROM recurrence_rules ORDER BY id`
    rows, err := r.pool.Query(ctx, q)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    var out []*recurrencedomain.Rule
    for rows.Next() {
        rule, err := scanRule(rows)
        if err != nil {
            return nil, err
        }
        out = append(out, rule)
    }
    return out, rows.Err()
}

type scanner interface{ Scan(dest ...any) error }

func scanRule(s scanner) (*recurrencedomain.Rule, error) {
    var (
        rule          recurrencedomain.Rule
        ruleType      string
        specificDates []time.Time
        dayParity     *string
        endDate       *time.Time
    )
    if err := s.Scan(&rule.ID, &rule.TaskID, &ruleType, &rule.IntervalDays, &rule.MonthDay,
        &specificDates, &dayParity, &rule.StartDate, &endDate, &rule.CreatedAt, &rule.UpdatedAt); err != nil {
        return nil, err
    }
    rule.RuleType = recurrencedomain.RuleType(ruleType)
    rule.SpecificDates = specificDates
    rule.EndDate = endDate
    if dayParity != nil {
        dp := recurrencedomain.DayParity(*dayParity)
        rule.DayParity = &dp
    }
    return &rule, nil
}

func dpStr(dp *recurrencedomain.DayParity) *string {
    if dp == nil {
        return nil
    }
    s := string(*dp)
    return &s
}