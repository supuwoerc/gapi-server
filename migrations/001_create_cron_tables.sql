-- 定时任务注册表
CREATE TABLE sys_cron_job (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(128) NOT NULL COMMENT '任务唯一标识',
    description VARCHAR(512) DEFAULT '' COMMENT '任务描述',
    `interval` VARCHAR(64) NOT NULL COMMENT 'cron 表达式（6位含秒）',
    enabled TINYINT(1) DEFAULT 1 COMMENT '是否启用',
    last_run_at DATETIME DEFAULT NULL COMMENT '最近一次执行时间',
    last_status VARCHAR(16) DEFAULT '' COMMENT '最近一次执行状态',
    created_at DATETIME DEFAULT NULL,
    updated_at DATETIME DEFAULT NULL,
    deleted_at BIGINT UNSIGNED DEFAULT 0,
    UNIQUE INDEX idx_name (name),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务注册表';

-- 执行日志表
CREATE TABLE sys_cron_job_execution (
    id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    job_name VARCHAR(128) NOT NULL COMMENT '任务名称',
    status VARCHAR(16) NOT NULL COMMENT '执行状态(running/success/failed/cancelled/panic)',
    started_at DATETIME NOT NULL COMMENT '开始时间',
    ended_at DATETIME DEFAULT NULL COMMENT '结束时间',
    duration BIGINT DEFAULT NULL COMMENT '耗时（毫秒）',
    error TEXT COMMENT '错误信息',
    triggered_by VARCHAR(32) DEFAULT 'scheduler' COMMENT '触发方式(scheduler/manual)',
    created_at DATETIME DEFAULT NULL,
    updated_at DATETIME DEFAULT NULL,
    deleted_at BIGINT UNSIGNED DEFAULT 0,
    INDEX idx_job_name (job_name),
    INDEX idx_deleted_at (deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='定时任务执行日志';
