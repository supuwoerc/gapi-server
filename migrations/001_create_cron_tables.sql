-- 定时任务注册表
CREATE TABLE sys_cron_job
(
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(128) NOT NULL,
    description  VARCHAR(512) NOT NULL DEFAULT '',
    "interval"   VARCHAR(64)  NOT NULL,
    enabled      BOOLEAN      NOT NULL DEFAULT TRUE,
    last_run_at  TIMESTAMPTZ           DEFAULT NULL,
    last_status  VARCHAR(16)  NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    deleted_at   BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_name ON sys_cron_job (name);
CREATE INDEX idx_cron_job_deleted_at ON sys_cron_job (deleted_at);

COMMENT ON TABLE sys_cron_job IS '定时任务注册表';
COMMENT ON COLUMN sys_cron_job.name IS '任务唯一标识';
COMMENT ON COLUMN sys_cron_job.description IS '任务描述';
COMMENT ON COLUMN sys_cron_job."interval" IS 'cron 表达式（6位含秒）';
COMMENT ON COLUMN sys_cron_job.enabled IS '是否启用';
COMMENT ON COLUMN sys_cron_job.last_run_at IS '最近一次执行时间';
COMMENT ON COLUMN sys_cron_job.last_status IS '最近一次执行状态';

-- 执行日志表
CREATE TABLE sys_cron_job_execution
(
    id           BIGSERIAL PRIMARY KEY,
    job_name     VARCHAR(128) NOT NULL,
    status       VARCHAR(16)  NOT NULL,
    started_at   TIMESTAMPTZ  NOT NULL,
    ended_at     TIMESTAMPTZ           DEFAULT NULL,
    duration     BIGINT                DEFAULT NULL,
    error        TEXT         NOT NULL,
    triggered_by VARCHAR(32)  NOT NULL DEFAULT 'scheduler',
    created_at   TIMESTAMPTZ  NOT NULL,
    updated_at   TIMESTAMPTZ  NOT NULL,
    deleted_at   BIGINT       NOT NULL DEFAULT 0
);

CREATE INDEX idx_job_name ON sys_cron_job_execution (job_name);
CREATE INDEX idx_cron_job_execution_deleted_at ON sys_cron_job_execution (deleted_at);

COMMENT ON TABLE sys_cron_job_execution IS '定时任务执行日志';
COMMENT ON COLUMN sys_cron_job_execution.job_name IS '任务名称';
COMMENT ON COLUMN sys_cron_job_execution.status IS '执行状态(running/success/failed/cancelled/panic)';
COMMENT ON COLUMN sys_cron_job_execution.started_at IS '开始时间';
COMMENT ON COLUMN sys_cron_job_execution.ended_at IS '结束时间';
COMMENT ON COLUMN sys_cron_job_execution.duration IS '耗时（毫秒）';
COMMENT ON COLUMN sys_cron_job_execution.error IS '错误信息';
COMMENT ON COLUMN sys_cron_job_execution.triggered_by IS '触发方式(scheduler/manual)';