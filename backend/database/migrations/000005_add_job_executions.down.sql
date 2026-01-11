-- Drop job_executions table
DROP INDEX IF EXISTS idx_job_executions_job_name;
DROP INDEX IF EXISTS idx_job_executions_deleted_at;
DROP TABLE IF EXISTS job_executions;
