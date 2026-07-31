-- 用户表
CREATE TABLE sys_user
(
    id               BIGSERIAL PRIMARY KEY,
    username         VARCHAR(64)  NOT NULL,
    password_hash    VARCHAR(256) NOT NULL,
    email            VARCHAR(128) NOT NULL DEFAULT '',
    phone            VARCHAR(32)  NOT NULL DEFAULT '',
    avatar           VARCHAR(512) NOT NULL DEFAULT '',
    bio              VARCHAR(256) NOT NULL DEFAULT '',
    enabled          BOOLEAN      NOT NULL DEFAULT TRUE,
    last_login_at    TIMESTAMPTZ           DEFAULT NULL,
    login_fail_count INT          NOT NULL DEFAULT 0,
    locked_until     TIMESTAMPTZ           DEFAULT NULL,
    completed_tours  JSONB                 DEFAULT NULL,
    created_at       TIMESTAMPTZ  NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL,
    deleted_at       BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_username ON sys_user (username);
CREATE UNIQUE INDEX idx_email ON sys_user (email);
CREATE INDEX idx_user_deleted_at ON sys_user (deleted_at);

COMMENT ON TABLE sys_user IS '用户表';
COMMENT ON COLUMN sys_user.username IS '登录名';
COMMENT ON COLUMN sys_user.password_hash IS '密码哈希';
COMMENT ON COLUMN sys_user.email IS '邮箱';
COMMENT ON COLUMN sys_user.phone IS '手机号';
COMMENT ON COLUMN sys_user.avatar IS '头像URL';
COMMENT ON COLUMN sys_user.bio IS '个人简介';
COMMENT ON COLUMN sys_user.enabled IS '是否启用';
COMMENT ON COLUMN sys_user.last_login_at IS '最近登录时间';
COMMENT ON COLUMN sys_user.login_fail_count IS '连续登录失败次数';
COMMENT ON COLUMN sys_user.locked_until IS '锁定截止时间';
COMMENT ON COLUMN sys_user.completed_tours IS '已完成的引导';

-- 角色表
CREATE TABLE sys_role
(
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(64)  NOT NULL,
    code        VARCHAR(64)  NOT NULL,
    parent_id   BIGINT                DEFAULT NULL,
    description VARCHAR(256) NOT NULL DEFAULT '',
    sort_order  INT          NOT NULL DEFAULT 0,
    enabled     BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ  NOT NULL,
    updated_at  TIMESTAMPTZ  NOT NULL,
    deleted_at  BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_code ON sys_role (code);
CREATE INDEX idx_parent_id ON sys_role (parent_id);
CREATE INDEX idx_role_deleted_at ON sys_role (deleted_at);

COMMENT ON TABLE sys_role IS '角色表';
COMMENT ON COLUMN sys_role.name IS '角色显示名称';
COMMENT ON COLUMN sys_role.code IS '角色唯一标识';
COMMENT ON COLUMN sys_role.parent_id IS '父角色ID';
COMMENT ON COLUMN sys_role.description IS '角色描述';
COMMENT ON COLUMN sys_role.sort_order IS '排序';
COMMENT ON COLUMN sys_role.enabled IS '是否启用';

-- 权限表
CREATE TABLE sys_permission
(
    id            BIGSERIAL PRIMARY KEY,
    code          VARCHAR(128) NOT NULL,
    name          VARCHAR(128) NOT NULL,
    resource_type SMALLINT     NOT NULL,
    module        VARCHAR(64)  NOT NULL DEFAULT '',
    resource_path VARCHAR(256) NOT NULL DEFAULT '',
    action        VARCHAR(32)  NOT NULL DEFAULT '',
    description   VARCHAR(256) NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL,
    updated_at    TIMESTAMPTZ  NOT NULL,
    deleted_at    BIGINT       NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX idx_permission_code ON sys_permission (code);
CREATE INDEX idx_module_resource_type ON sys_permission (module, resource_type);
CREATE INDEX idx_permission_deleted_at ON sys_permission (deleted_at);

COMMENT ON TABLE sys_permission IS '权限表';
COMMENT ON COLUMN sys_permission.code IS '权限标识 eg:user:create';
COMMENT ON COLUMN sys_permission.name IS '权限显示名称';
COMMENT ON COLUMN sys_permission.resource_type IS '资源类型 1=api 2=frontend-menu 3=frontend-route 4=frontend-button 5=data';
COMMENT ON COLUMN sys_permission.module IS '所属模块';
COMMENT ON COLUMN sys_permission.resource_path IS '资源路径';
COMMENT ON COLUMN sys_permission.action IS '操作 create/read/update/delete';
COMMENT ON COLUMN sys_permission.description IS '权限描述';

-- 用户角色关联表
CREATE TABLE sys_user_role
(
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT      NOT NULL,
    role_id    BIGINT      NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL,
    deleted_at BIGINT      NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX uk_user_role ON sys_user_role (user_id, role_id, deleted_at);
CREATE INDEX idx_role_id ON sys_user_role (role_id);
CREATE INDEX idx_user_role_deleted_at ON sys_user_role (deleted_at);

COMMENT ON TABLE sys_user_role IS '用户角色关联表';
COMMENT ON COLUMN sys_user_role.user_id IS '用户ID';
COMMENT ON COLUMN sys_user_role.role_id IS '角色ID';

-- 角色权限关联表
CREATE TABLE sys_role_permission
(
    id            BIGSERIAL PRIMARY KEY,
    role_id       BIGINT      NOT NULL,
    permission_id BIGINT      NOT NULL,
    effect        VARCHAR(8)  NOT NULL DEFAULT 'allow',
    created_at    TIMESTAMPTZ NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL,
    deleted_at    BIGINT      NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX uk_role_perm ON sys_role_permission (role_id, permission_id, deleted_at);
CREATE INDEX idx_permission_id ON sys_role_permission (permission_id);
CREATE INDEX idx_role_permission_deleted_at ON sys_role_permission (deleted_at);

COMMENT ON TABLE sys_role_permission IS '角色权限关联表';
COMMENT ON COLUMN sys_role_permission.role_id IS '角色ID';
COMMENT ON COLUMN sys_role_permission.permission_id IS '权限ID';
COMMENT ON COLUMN sys_role_permission.effect IS '效果 allow/deny';