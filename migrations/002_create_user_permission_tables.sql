-- 用户表
CREATE TABLE sys_user
(
    id               BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    username         VARCHAR(64)     NOT NULL COMMENT '登录名',
    password_hash    VARCHAR(256)    NOT NULL COMMENT '密码哈希',
    email            VARCHAR(128)    NOT NULL DEFAULT '' COMMENT '邮箱',
    phone            VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '手机号',
    avatar           VARCHAR(512)    NOT NULL DEFAULT '' COMMENT '头像URL',
    bio              VARCHAR(256)    NOT NULL DEFAULT '' COMMENT '个人简介',
    status           TINYINT         NOT NULL DEFAULT 1 COMMENT '状态 1=启用 0=禁用',
    last_login_at    DATETIME                 DEFAULT NULL COMMENT '最近登录时间',
    login_fail_count INT             NOT NULL DEFAULT 0 COMMENT '连续登录失败次数',
    locked_until     DATETIME                 DEFAULT NULL COMMENT '锁定截止时间',
    completed_tours  JSON                     DEFAULT NULL COMMENT '已完成的引导',
    created_at       DATETIME        NOT NULL,
    updated_at       DATETIME        NOT NULL,
    deleted_at       BIGINT UNSIGNED NOT NULL DEFAULT 0,
    UNIQUE INDEX idx_username (username),
    UNIQUE INDEX idx_email (email),
    INDEX idx_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户表';

-- 角色表
CREATE TABLE sys_role
(
    id          BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    name        VARCHAR(64)     NOT NULL COMMENT '角色显示名称',
    code        VARCHAR(64)     NOT NULL COMMENT '角色唯一标识',
    parent_id   BIGINT UNSIGNED          DEFAULT NULL COMMENT '父角色ID',
    description VARCHAR(256)    NOT NULL DEFAULT '' COMMENT '角色描述',
    sort_order  INT             NOT NULL DEFAULT 0 COMMENT '排序',
    status      TINYINT         NOT NULL DEFAULT 1 COMMENT '状态 1=启用 0=禁用',
    created_at  DATETIME        NOT NULL,
    updated_at  DATETIME        NOT NULL,
    deleted_at  BIGINT UNSIGNED NOT NULL DEFAULT 0,
    UNIQUE INDEX idx_code (code),
    INDEX idx_parent_id (parent_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='角色表';

-- 权限表
CREATE TABLE sys_permission
(
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    code          VARCHAR(128)    NOT NULL COMMENT '权限标识 eg:user:create',
    name          VARCHAR(128)    NOT NULL COMMENT '权限显示名称',
    resource_type TINYINT         NOT NULL COMMENT '资源类型 1=api 2=frontend-menu 3=frontend-route 4=frontend-button 5=data',
    module        VARCHAR(64)     NOT NULL DEFAULT '' COMMENT '所属模块',
    resource_path VARCHAR(256)    NOT NULL DEFAULT '' COMMENT '资源路径',
    action        VARCHAR(32)     NOT NULL DEFAULT '' COMMENT '操作 create/read/update/delete',
    description   VARCHAR(256)    NOT NULL DEFAULT '' COMMENT '权限描述',
    created_at    DATETIME        NOT NULL,
    updated_at    DATETIME        NOT NULL,
    deleted_at    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    UNIQUE INDEX idx_code (code),
    INDEX idx_module_resource_type (module, resource_type),
    INDEX idx_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='权限表';

-- 用户角色关联表
CREATE TABLE sys_user_role
(
    id         BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    user_id    BIGINT UNSIGNED NOT NULL COMMENT '用户ID',
    role_id    BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    created_at DATETIME        NOT NULL,
    updated_at DATETIME        NOT NULL,
    deleted_at BIGINT UNSIGNED NOT NULL DEFAULT 0,
    UNIQUE INDEX uk_user_role (user_id, role_id, deleted_at),
    INDEX idx_role_id (role_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='用户角色关联表';

-- 角色权限关联表
CREATE TABLE sys_role_permission
(
    id            BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
    role_id       BIGINT UNSIGNED NOT NULL COMMENT '角色ID',
    permission_id BIGINT UNSIGNED NOT NULL COMMENT '权限ID',
    effect        VARCHAR(8)      NOT NULL DEFAULT 'allow' COMMENT '效果 allow/deny',
    created_at    DATETIME        NOT NULL,
    updated_at    DATETIME        NOT NULL,
    deleted_at    BIGINT UNSIGNED NOT NULL DEFAULT 0,
    UNIQUE INDEX uk_role_perm (role_id, permission_id, deleted_at),
    INDEX idx_permission_id (permission_id),
    INDEX idx_deleted_at (deleted_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4 COMMENT ='角色权限关联表';