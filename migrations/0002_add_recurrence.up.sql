CREATE TABLE IF NOT EXISTS recurrence_rules (
    id            BIGSERIAL PRIMARY KEY,
    task_id       BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    rule_type     TEXT NOT NULL,        -- daily | monthly | specific_dates | even_odd
    interval_days INT,                  -- for daily: every N days
    month_day     INT,                  -- for monthly: 1..30
    specific_dates DATE[],              -- for specific_dates
    day_parity    TEXT,                 -- for even_odd: 'even' | 'odd'
    start_date    DATE NOT NULL,
    end_date      DATE,                 -- NULL = open-ended
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_task_rule UNIQUE (task_id)
);

CREATE INDEX IF NOT EXISTS idx_recurrence_rules_task_id ON recurrence_rules (task_id);

CREATE TABLE IF NOT EXISTS recurrence_occurrences (
    id              BIGSERIAL PRIMARY KEY,
    rule_id         BIGINT NOT NULL REFERENCES recurrence_rules(id) ON DELETE CASCADE,
    scheduled_date  DATE NOT NULL,
    task_id         BIGINT NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    CONSTRAINT unique_occurrence UNIQUE (rule_id, scheduled_date)
);

CREATE INDEX IF NOT EXISTS idx_occurrences_rule_id ON recurrence_occurrences (rule_id);

ALTER TABLE tasks
    ADD COLUMN IF NOT EXISTS parent_task_id BIGINT REFERENCES tasks(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS scheduled_date DATE;

CREATE INDEX IF NOT EXISTS idx_tasks_parent_task_id ON tasks (parent_task_id);
CREATE INDEX IF NOT EXISTS idx_tasks_scheduled_date  ON tasks (scheduled_date);