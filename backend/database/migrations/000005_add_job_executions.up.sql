-- Create job_executions table for tracking scheduled job runs
-- This prevents duplicate job executions during rapid server restarts
CREATE TABLE IF NOT EXISTS job_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    job_name TEXT UNIQUE NOT NULL,
    last_run_at DATETIME NOT NULL,
    locked_at DATETIME,
    locked_by TEXT,
    created_at DATETIME,
    updated_at DATETIME,
    deleted_at DATETIME
);

CREATE INDEX IF NOT EXISTS idx_job_executions_deleted_at ON job_executions(deleted_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_job_executions_job_name ON job_executions(job_name);
