# MySQL 迁移到 PostgreSQL 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 gapi-server 的持久层从 MySQL 8.0 彻底切换到 PostgreSQL 16，不保留 MySQL 兼容路径。

**Architecture:** 项目的数据库耦合极浅 —— 业务层 12k 行 Go 代码中没有任何原生 SQL，全部通过 GORM + gorm.io/gen 生成的类型安全查询访问数据库。真正的 MySQL 绑定只在 `pkg/database/database.go` 的 driver 导入和 DSN 拼接两处。因此迁移的重心不在应用代码，而在 (1) 重写 `migrations/` 下的 DDL，(2) 处理 PostgreSQL 没有 unsigned 整型所引发的 `uint64` → `int64` 连锁改动。因为 gen 通过 `g.UseDB()` 从活库反向生成模型，schema 一改，7 张表的 14 个 `*.gen.go` 会重新生成为 `int64`，进而波及 76 处业务引用。

**Tech Stack:** Go 1.26.1, GORM 1.31.2, gorm.io/driver/postgres 1.6.0, gorm.io/gen 0.3.28, jackc/pgx v5.10.0, PostgreSQL 16+, Docker Compose, google/wire, testify

## Global Constraints

- **依赖版本（已在本机实测：`go build ./...` 与 `go test ./...` 全绿）**：

  | 模块 | 目标版本 | 当前版本 |
  |---|---|---|
  | `gorm.io/gorm` | v1.31.2 | v1.31.1 |
  | `gorm.io/driver/postgres` | v1.6.0（最新） | 新增 |
  | `gorm.io/gen` | v0.3.28 | v0.3.27 |
  | `github.com/jackc/pgx/v5` | v5.10.0 | 无（go.sum 有 v5.5.5 陈旧条目） |
  | `gorm.io/datatypes` | v1.2.7（已最新） | v1.2.7 |
  | `gorm.io/plugin/soft_delete` | v1.2.1（已最新） | v1.2.1 |
  | `gorm.io/plugin/dbresolver` | v1.6.2（已最新） | v1.6.2 |

- **pgx 必须显式升级。** `go get gorm.io/driver/postgres@latest` 只会拉到 pgx **v5.6.0** —— Go 的最小版本选择（MVS）取驱动声明的下界，而非 pgx 的最新版。要拿到 v5.10.0 必须单独 `go get github.com/jackc/pgx/v5@latest`。
- **`gorm.io/driver/mysql` 无法从 go.mod 移除。** `gorm.io/gen` 对它有传递依赖（`gorm.io/gen@v0.3.28 → gorm.io/driver/mysql@v1.5.7`），`go mod tidy` 会把它作为 `// indirect` 重新写回。目标是让它从 `require` 直接块降级为 indirect，而不是消失。`github.com/go-sql-driver/mysql v1.8.1 // indirect` 同理。
- PostgreSQL 服务端版本：`postgres:16-alpine`（见 Task 2 Step 1 的版本说明）
- 表名前缀 `sys_` 与单数表名策略（`SingularTable: true`）保持不变
- 所有主键与外键：Go 侧 `int64`，PG 侧 `BIGINT` / `BIGSERIAL`
- 软删除：保持 `soft_delete.DeletedAt` + `softDelete:milli`（bigint 存毫秒时间戳），语义不变
- `deleted_at` 列保持 `NOT NULL DEFAULT 0`，参与联合唯一索引
- 时区：容器与连接统一 `Asia/Shanghai`
- 不做数据搬迁（开发阶段可重建库），不需要回滚脚本
- 保留 `TranslateError: true`，使 `gorm.ErrDuplicatedKey` / `ErrRecordNotFound` 在 PG 下继续生效
- 每个任务结束时必须 `go build ./...` 通过
- **版本控制：不执行任何 git 操作。** 不要 `git add`、`git commit`、切分支或改动暂存区。每个任务末尾只列出变更文件清单，由用户自行审阅并手动提交。执行下一个任务前无需等待提交完成。

---

## File Structure

**修改：**
- `pkg/database/database.go` — driver 导入与 DSN 构造（唯一的驱动耦合点）
- `internal/config/config.go:87-99` — `DatabaseConfig` 增加 `SSLMode`、`TimeZone` 字段
- `configs/default.yaml:8-18` — 端口 3306 → 5432，用户名，新增 sslmode/timezone
- `internal/config/config_test.go:15,32` — 端口断言
- `deploy/docker/docker-compose.yaml` — mysql 服务 → postgres 服务
- `cmd/gen/main.go` — 移除 `FieldSignable`（PG 下无 unsigned 可推断）
- `internal/dal/{user,permission,cron_job,token}.go` — `uint64` → `int64`
- `internal/service/{auth,user,cron_job}.go` — 接口签名 `uint64` → `int64`
- `internal/handler/v1/{auth,user}.go`、`internal/handler/v1/resp/cron_job.go` — 同上
- `internal/middleware/auth.go:20-27` — `CurrentUserID` 返回 `int64`
- `internal/cronjob/manager.go:20-21`、`manager_test.go:36,39` — 同上
- `pkg/jwt/jwt.go:19,45,69`、`pkg/jwt/jwt_test.go:20,40` — JWT claims `UserID int64`
- `internal/dal/model/*.gen.go`、`internal/dal/query/*.gen.go` — 由 gen 重新生成（14 个文件，不手改）

**重写：**
- `migrations/001_create_cron_tables.sql`
- `migrations/002_create_user_permission_tables.sql`
- `migrations/003_seed_frontend_permissions.sql`

**新建：**
- `migrations/README.md` — 手工执行顺序说明（当前无 migration runner）

**不改动：**
- `internal/dal/model/{permission_action,permission_effect,cron_job_triggered_by}.go` — `Scan` 已同时处理 `string` 与 `[]byte`，跨方言安全
- `pkg/database/tx.go`、`pkg/database/logger.go` — 与方言无关
- `internal/dal/captcha.go` — 纯 Redis

---

### Task 1: 切换数据库驱动与 DSN

把唯一的驱动耦合点换成 PostgreSQL。此时 migrations 还是 MySQL 语法，服务连不上真实库，但编译必须通过。

**Files:**
- Modify: `pkg/database/database.go:11,20-26,38`
- Modify: `internal/config/config.go:87-99`
- Modify: `configs/default.yaml:8-18`
- Test: `internal/config/config_test.go:15,32`

**Interfaces:**
- Consumes: 无（首个任务）
- Produces: `config.DatabaseConfig` 新增两个字段 `SSLMode string`（mapstructure `sslmode`）、`TimeZone string`（mapstructure `timezone`）；`database.NewConnection(cfg *config.DatabaseConfig, l *logger.Logger) (*gorm.DB, error)` 签名不变

- [ ] **Step 1: 升级并添加依赖**

先只加依赖、不动 `go.mod` 的 mysql 条目 —— 此刻 `pkg/database/database.go` 仍在 import mysql driver，提前 droprequire 会让 `go mod tidy` 立刻把它加回来。清理留到 Step 9，代码改完之后做。

第二条命令是必须的：单靠 `driver/postgres@latest` 只能拉到 pgx v5.6.0（MVS 取驱动声明的下界），显式指定才能到 v5.10.0。

```bash
go get gorm.io/gorm@latest gorm.io/driver/postgres@latest gorm.io/gen@latest
go get github.com/jackc/pgx/v5@latest
```

- [ ] **Step 2: 核对解析出的版本**

```bash
go list -m gorm.io/gorm gorm.io/driver/postgres gorm.io/gen github.com/jackc/pgx/v5
```

Expected:

```
gorm.io/gorm v1.31.2
gorm.io/driver/postgres v1.6.0
gorm.io/gen v0.3.28
github.com/jackc/pgx/v5 v5.10.0
```

若 pgx 显示 v5.6.0，说明 Step 1 的第二条命令没执行，补跑一次。

- [ ] **Step 3: 写失败测试 —— 配置能读取 sslmode 与 timezone**

修改 `internal/config/config_test.go`，把 `v.Set("database.port", 3306)` 改为 `5432`，并在其后追加两行 Set；同时把第 32 行断言 `3306` 改为 `5432`，然后在该断言后追加两行：

```go
	v.Set("database.port", 5432)
	v.Set("database.sslmode", "disable")
	v.Set("database.timezone", "Asia/Shanghai")
```

```go
	assert.Equal(t, 5432, cfg.Database.Port)
	assert.Equal(t, "disable", cfg.Database.SSLMode)
	assert.Equal(t, "Asia/Shanghai", cfg.Database.TimeZone)
```

- [ ] **Step 4: 运行测试确认失败**

Run: `go test ./internal/config/ -run TestNewConfig -v`
Expected: 编译失败，`cfg.Database.SSLMode undefined (type config.DatabaseConfig has no field or method SSLMode)`

- [ ] **Step 5: 给 DatabaseConfig 加字段**

把 `internal/config/config.go:87-99` 的注释与结构体替换为（注释里的 MySQL 字样一并改掉）：

```go
// DatabaseConfig holds PostgreSQL connection and pool settings.
type DatabaseConfig struct {
	Host            string `mapstructure:"host"`              // 数据库主机地址
	Port            int    `mapstructure:"port"`              // 数据库端口
	User            string `mapstructure:"user"`              // 数据库用户名
	Password        string `mapstructure:"password"`          // 数据库密码
	DBName          string `mapstructure:"dbname"`            // 数据库名称
	SSLMode         string `mapstructure:"sslmode"`           // SSL 模式 (disable/require/verify-ca/verify-full)
	TimeZone        string `mapstructure:"timezone"`          // 连接时区 (如 Asia/Shanghai)
	MaxIdleConns    int    `mapstructure:"max_idle_conns"`    // 连接池最大空闲连接数
	MaxOpenConns    int    `mapstructure:"max_open_conns"`    // 连接池最大打开连接数
	ConnMaxLifetime int    `mapstructure:"conn_max_lifetime"` // 连接最大存活时间 (单位: 秒)
	SlowThreshold   int    `mapstructure:"slow_threshold"`    // 慢查询阈值 (单位: 毫秒)
	LogLevel        int    `mapstructure:"log_level"`         // GORM 日志级别 (1=Silent 2=Error 3=Warn 4=Info)
}
```

- [ ] **Step 6: 运行测试确认通过**

Run: `go test ./internal/config/ -run TestNewConfig -v`
Expected: PASS

- [ ] **Step 7: 替换 driver 导入**

`pkg/database/database.go:11`，把 `"gorm.io/driver/mysql"` 改为：

```go
	"gorm.io/driver/postgres"
```

- [ ] **Step 8: 改写 DSN 构造**

替换 `pkg/database/database.go:20-26`。MySQL 的 `charset=utf8mb4&parseTime=True&loc=Local` 三个参数是 MySQL 专属，PG 用 `sslmode` 与 `TimeZone` 代替（PG 的编码由数据库自身的 `ENCODING` 决定，无需连接参数）：

```go
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	timeZone := cfg.TimeZone
	if timeZone == "" {
		timeZone = "Asia/Shanghai"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		sslMode,
		timeZone,
	)
```

- [ ] **Step 9: 改 gorm.Open 的方言**

`pkg/database/database.go:38`，把 `mysql.Open(dsn)` 改为 `postgres.Open(dsn)`。保留 `NamingStrategy`、`Logger`、`TranslateError: true` 三项不动：

```go
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
```

- [ ] **Step 10: 更新默认配置**

替换 `configs/default.yaml:8-18` 的 database 段：

```yaml
database:
  host: "127.0.0.1"           # 数据库主机地址
  port: 5432                  # 数据库端口
  user: "postgres"            # 数据库用户名
  password: "password"        # 数据库密码
  dbname: "gapi"              # 数据库名称
  sslmode: "disable"          # SSL 模式 (disable/require/verify-ca/verify-full)
  timezone: "Asia/Shanghai"   # 连接时区
  max_idle_conns: 10          # 连接池最大空闲连接数
  max_open_conns: 100         # 连接池最大打开连接数
  conn_max_lifetime: 3600            # 连接最大存活时间 (单位: 秒, 3600 = 1h)
  slow_threshold: 200               # 慢查询阈值 (单位: 毫秒, 200 = 200ms)
  log_level: 2                # GORM 日志级别 (1=Silent 2=Error 3=Warn 4=Info)
```

- [ ] **Step 11: 整理 go.mod 并确认 mysql 已降级为 indirect**

代码已不再 import mysql driver，现在 tidy 才能把它移出 `require` 直接块：

```bash
go mod tidy
```

然后确认：

```bash
grep -nE "gorm\.io|jackc/pgx" go.mod
```

Expected: `gorm.io/driver/postgres v1.6.0` 出现在直接依赖块中；`gorm.io/driver/mysql v1.5.7 // indirect` 与 `github.com/go-sql-driver/mysql v1.8.1 // indirect` 仍在，但都带 `// indirect` 标记。

这两个 mysql 条目**不能也不该删掉** —— `gorm.io/gen` 传递依赖它们，手工 droprequire 后下次 tidy 会原样加回。它们不会被编译进二进制，因为没有任何代码 import。

- [ ] **Step 12: 编译并跑全量测试**

Run: `go build ./... && go test ./...`
Expected: 全部 PASS（现有 15 个测试文件都不连数据库，不受影响）

**Task 1 完成。** 变更文件：`go.mod`、`go.sum`、`pkg/database/database.go`、`internal/config/config.go`、`internal/config/config_test.go`、`configs/default.yaml`。交由用户手动提交，不要执行任何 git 命令。

---

### Task 2: 替换本地开发环境为 PostgreSQL

先把库跑起来，后面的 migration 和 gen 重新生成都依赖一个可连接的 PG 实例。

**Files:**
- Modify: `deploy/docker/docker-compose.yaml:2-16`

**Interfaces:**
- Consumes: Task 1 的 `configs/default.yaml`（port 5432、user postgres、dbname gapi）
- Produces: 本机 `127.0.0.1:5432` 上的 PostgreSQL 16，库名 `gapi`，用户 `postgres`，密码 `password`

- [ ] **Step 1: 替换 mysql 服务定义**

**关于镜像版本：** 下面用 `postgres:16-alpine`。写计划时本机无法访问 Docker Hub，因此没有核实 PostgreSQL 服务端的当前最新大版本（17 和 18 可能已发布）。执行到这一步时先确认一次：

```bash
docker run --rm postgres:18-alpine postgres --version 2>/dev/null \
  || docker run --rm postgres:17-alpine postgres --version 2>/dev/null \
  || docker run --rm postgres:16-alpine postgres --version
```

取能拉到的最高版本，把下面 yaml 里的 tag 与 Global Constraints 的版本记录一并改掉。本计划的 DDL 只用了 `BIGSERIAL`、`JSONB`、`TIMESTAMPTZ`、`COMMENT ON`、`CROSS JOIN`、`ON CONFLICT` 这些特性，PG 12 起就全部支持，**任何 ≥16 的版本都不需要改 SQL**。

把 `deploy/docker/docker-compose.yaml` 开头的 `mysql:` 服务块（第 2-16 行，到 `command:` 行结束）整段替换为下面内容。`redis`、`etcd` 两个服务与 `volumes:` 段保持不动：

```yaml
  postgres:
    image: postgres:16-alpine
    container_name: gapi-postgres
    restart: unless-stopped
    ports:
      - "5432:5432"
    environment:
      POSTGRES_PASSWORD: password
      POSTGRES_USER: postgres
      POSTGRES_DB: gapi
      TZ: Asia/Shanghai
      PGTZ: Asia/Shanghai
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres -d gapi"]
      interval: 5s
      timeout: 5s
      retries: 10
```

- [ ] **Step 2: 更新 volumes 声明**

文件末尾的 `volumes:` 段里，把 `mysql_data:` 改为 `postgres_data:`，`redis_data:` 与 `etcd_data:` 不动：

```yaml
volumes:
  postgres_data:
  redis_data:
  etcd_data:
```

- [ ] **Step 3: 启动并确认健康**

```bash
docker compose -f deploy/docker/docker-compose.yaml up -d postgres
docker compose -f deploy/docker/docker-compose.yaml ps postgres
```

Expected: 状态显示 `healthy`

- [ ] **Step 4: 确认能连上且库存在**

Run: `docker exec gapi-postgres psql -U postgres -d gapi -c "SELECT version();"`
Expected: 输出 `PostgreSQL 16.x ... ` 一行

**Task 2 完成。** 变更文件：`deploy/docker/docker-compose.yaml`。交由用户手动提交，不要执行任何 git 命令。

---

### Task 3: 重写 cron 表 DDL

从最小的一组表开始，先验证 DDL 转换的模式，再套用到更复杂的权限表。

**Files:**
- Modify: `migrations/001_create_cron_tables.sql`（整体重写）

**Interfaces:**
- Consumes: Task 2 的 PG 实例
- Produces: `sys_cron_job`、`sys_cron_job_execution` 两张表。列名与语义与原 MySQL 版本完全一致，供 Task 6 的 gen 反向生成

**转换要点（本任务及 Task 4 共用）：**
- `BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY` → `BIGSERIAL PRIMARY KEY`
- `BIGINT UNSIGNED`（外键/deleted_at）→ `BIGINT`
- `TINYINT(1)` → `BOOLEAN`，默认值 `1` → `TRUE`
- `TINYINT` → `SMALLINT`
- `DATETIME` → `TIMESTAMPTZ`
- `JSON` → `JSONB`
- `ENGINE=InnoDB DEFAULT CHARSET=utf8mb4` → 整段删除
- 列内联 `COMMENT '...'` → PG 不支持，改为独立 `COMMENT ON COLUMN` 语句
- 表内联 `UNIQUE INDEX x (col)` / `INDEX x (col)` → PG 不支持写在 CREATE TABLE 里，拆为表外 `CREATE [UNIQUE] INDEX`
- 反引号 `` `interval` ``（MySQL 保留字转义）→ 双引号 `"interval"`（interval 在 PG 里是类型名，仍需引号）

- [ ] **Step 1: 重写 001**

把 `migrations/001_create_cron_tables.sql` 全文替换为：

```sql
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
```

注意：原 MySQL 版本两张表都有一个叫 `idx_deleted_at` 的索引。MySQL 的索引名只在表内唯一，PG 的索引名在整个 schema 内唯一，所以这里必须改名为 `idx_cron_job_deleted_at` 和 `idx_cron_job_execution_deleted_at`。

- [ ] **Step 2: 执行并确认无错**

```bash
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/001_create_cron_tables.sql
```

Expected: 输出若干 `CREATE TABLE` / `CREATE INDEX` / `COMMENT`，无 ERROR

- [ ] **Step 3: 验证表结构**

Run: `docker exec gapi-postgres psql -U postgres -d gapi -c "\d sys_cron_job"`
Expected: `id` 为 `bigint`、default `nextval('sys_cron_job_id_seq'::regclass)`；`enabled` 为 `boolean` default true；`"interval"` 列存在；索引含 `idx_name`（UNIQUE）

**Task 3 完成。** 变更文件：`migrations/001_create_cron_tables.sql`。交由用户手动提交，不要执行任何 git 命令。

---

### Task 4: 重写用户权限表 DDL 与 seed 数据

**Files:**
- Modify: `migrations/002_create_user_permission_tables.sql`（整体重写）
- Modify: `migrations/003_seed_frontend_permissions.sql`
- Create: `migrations/README.md`

**Interfaces:**
- Consumes: Task 3 的转换要点与已建好的 cron 表
- Produces: `sys_user`、`sys_role`、`sys_permission`、`sys_user_role`、`sys_role_permission` 五张表 + 前端权限 seed 数据

- [ ] **Step 1: 重写 002 的用户表与角色表**

把 `migrations/002_create_user_permission_tables.sql` 全文替换。先写前两张表：

```sql
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
```

- [ ] **Step 2: 追加权限表**

接着在同一文件末尾追加。注意 `resource_type` 从 `TINYINT` 变为 `SMALLINT`，`idx_code` 已被 `sys_role` 占用，权限表的必须改名为 `idx_permission_code`：

```sql
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
```

- [ ] **Step 3: 追加两张关联表**

继续在文件末尾追加。两个联合唯一索引都包含 `deleted_at`，这是软删除后允许重新插入同一组合的关键，必须保留：

```sql
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
```

- [ ] **Step 4: 执行 002 并验证**

```bash
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/002_create_user_permission_tables.sql
docker exec gapi-postgres psql -U postgres -d gapi -c "\dt sys_*"
```

Expected: 列出 7 张表（cron 2 张 + 本次 5 张），无 ERROR

- [ ] **Step 5: 调整 003 seed 脚本**

`migrations/003_seed_frontend_permissions.sql` 的两段 `INSERT ... VALUES` 在 PG 下完全可用，`NOW()` 也是标准函数，无需改动。只有末尾两条 `INSERT ... SELECT` 用了 MySQL 风格的隐式笛卡尔积 `FROM sys_role r, sys_permission p` —— 这在 PG 下同样合法，但显式 `CROSS JOIN` 更清晰。把最后两条语句替换为：

```sql
-- 给 admin 角色分配全部前端权限 (menu + route)
INSERT INTO sys_role_permission (role_id, permission_id, effect, created_at, updated_at, deleted_at)
SELECT r.id, p.id, 'allow', NOW(), NOW(), 0
FROM sys_role r
         CROSS JOIN sys_permission p
WHERE r.code = 'admin'
  AND p.resource_type IN (2, 3);

-- 给 user 角色分配基础前端权限（不含管理后台）
INSERT INTO sys_role_permission (role_id, permission_id, effect, created_at, updated_at, deleted_at)
SELECT r.id, p.id, 'allow', NOW(), NOW(), 0
FROM sys_role r
         CROSS JOIN sys_permission p
WHERE r.code = 'user'
  AND p.resource_type IN (2, 3)
  AND p.code NOT LIKE 'admin%'
  AND p.code NOT LIKE 'route:admin%';
```

- [ ] **Step 6: 执行 003 并验证权限已插入**

003 依赖 `sys_role` 里存在 `admin` / `user` 两个角色。开发库是空的，所以先插入这两个角色，再跑 seed：

```bash
docker exec gapi-postgres psql -U postgres -d gapi -c \
  "INSERT INTO sys_role (name, code, description, sort_order, enabled, created_at, updated_at, deleted_at) VALUES ('管理员','admin','',0,TRUE,NOW(),NOW(),0), ('普通用户','user','',1,TRUE,NOW(),NOW(),0);"
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/003_seed_frontend_permissions.sql
docker exec gapi-postgres psql -U postgres -d gapi -c \
  "SELECT r.code, count(*) FROM sys_role_permission rp JOIN sys_role r ON r.id = rp.role_id GROUP BY r.code ORDER BY r.code;"
```

Expected: `admin` 20 条（10 menu + 10 route），`user` 14 条（排除 3 个 admin menu 与 3 个 admin route，即 20 - 6）

- [ ] **Step 7: 写 migrations 执行说明**

项目当前没有 migration runner，也没有 `AutoMigrate` 调用，SQL 靠手工执行。新建 `migrations/README.md`，内容如下（注意这里外层用四个反引号包裹，实际文件内容里是三个）：

````markdown
# Migrations

目标数据库：PostgreSQL 16。当前没有自动化 migration runner，按序号手工执行。

## 首次初始化

```bash
docker compose -f deploy/docker/docker-compose.yaml up -d postgres

for f in migrations/0*.sql; do
  docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < "$f"
done
```

`003_seed_frontend_permissions.sql` 依赖 `sys_role` 中已存在 `admin` 与 `user`
两个角色，否则两条 `INSERT ... SELECT` 会静默插入 0 行。

## 约定

- 索引名在 PostgreSQL 中是 schema 级唯一的（MySQL 中仅表级唯一），
  新增索引时带上表名前缀避免冲突。
- 时间列统一 `TIMESTAMPTZ`；`deleted_at` 是 `BIGINT` 毫秒时间戳，
  由 `gorm.io/plugin/soft_delete` 维护，0 表示未删除。
- 改动表结构后必须重新生成 DAL：`go generate ./cmd/gen`。
````

**Task 4 完成。** 变更文件：`migrations/002_create_user_permission_tables.sql`、`migrations/003_seed_frontend_permissions.sql`、`migrations/README.md`（新建）。交由用户手动提交，不要执行任何 git 命令。

---

### Task 5: 重新生成 DAL 并把 uint64 改为 int64

这是工作量最大、最容易低估的一步。`cmd/gen/main.go:35` 的 `g.UseDB()` 从活库反向生成模型，PG 的 `BIGSERIAL` 会生成 `int64`，导致 14 个 `*.gen.go` 的 ID 类型全变，进而波及 76 处业务引用。生成代码与手写代码必须在同一个任务里改完，否则中间态无法编译。

**Files:**
- Modify: `cmd/gen/main.go:20-29`
- Modify: `internal/dal/model/*.gen.go`、`internal/dal/query/*.gen.go`（由工具生成，不手改）
- Modify: `internal/dal/{user,permission,cron_job,token}.go`
- Modify: `internal/service/{auth,user,cron_job}.go`
- Modify: `internal/handler/v1/{auth,user}.go`、`internal/handler/v1/resp/cron_job.go`
- Modify: `internal/middleware/auth.go:20-27`
- Modify: `internal/cronjob/manager.go:20-21`
- Modify: `pkg/jwt/jwt.go:19,45,69`
- Test: `pkg/jwt/jwt_test.go:20,40`、`internal/cronjob/manager_test.go:36,39`

**Interfaces:**
- Consumes: Task 4 建好的 7 张 PG 表
- Produces: 全项目主键/外键统一 `int64`。关键签名：
  - `jwt.Claims.UserID int64`
  - `jwt.Manager.GenerateTokenPair(userID int64, username string) (*TokenPair, error)`
  - `middleware.CurrentUserID(c *gin.Context) (int64, bool)`
  - `model.User.ID int64`、`model.Role.ID int64`、`model.Role.ParentID *int64`
  - `dal.UserDal.FindByID(ctx context.Context, id int64) (*model.User, error)`
  - `dal.PermissionDal.FindCodesByRoleIDsAndResourceType(ctx context.Context, roleIDs []int64, resourceType model.ResourceType) ([]string, error)`
  - `service.CronJobService.RecordStart(ctx context.Context, jobName string, triggeredBy model.TriggeredBy) (int64, error)`

- [ ] **Step 1: 关掉 FieldSignable**

`FieldSignable: true` 的作用是让 gen 按数据库的 unsigned 标记生成无符号 Go 类型。PG 没有 unsigned，这个开关已无意义，留着会造成误解。修改 `cmd/gen/main.go:20-29`：

```go
	g := gen.NewGenerator(gen.Config{
		OutPath:           filepath.Join(root, "internal/dal/query"),
		ModelPkgPath:      filepath.Join(root, "internal/dal/model"),
		Mode:              gen.WithQueryInterface,
		FieldNullable:     true,
		FieldCoverable:    false,
		FieldWithIndexTag: true,
		FieldWithTypeTag:  true,
	})
```

- [ ] **Step 2: 重新生成 DAL**

```bash
go generate ./cmd/gen
```

Expected: `internal/dal/model/` 与 `internal/dal/query/` 下 14 个 `.gen.go` 被重写

- [ ] **Step 3: 确认生成结果的类型变化**

```bash
grep -n "ID " internal/dal/model/sys_user.gen.go | head -3
grep -rn "CompletedTours\|Enabled " internal/dal/model/sys_user.gen.go
```

Expected: `ID int64`，gorm tag 里 `type:bigint`；`CompletedTours datatypes.JSONSlice[string]` 且 `type:jsonb`；`Enabled bool`

- [ ] **Step 4: 确认编译失败，看到全部待改点**

Run: `go build ./... 2>&1 | head -40`
Expected: 大量 `cannot use ... (variable of type uint64) as int64 value` 类型错误，集中在 dal / service / handler / middleware

- [ ] **Step 5: 批量替换业务代码里的 uint64**

这些文件里 `uint64` 全部表示实体 ID，可安全整体替换。`pkg/etcd/balancer.go:25` 的 `uint64(len(instances))` 是哈希取模，与数据库无关，**不要改**：

```bash
for f in \
  internal/dal/user.go \
  internal/dal/permission.go \
  internal/dal/cron_job.go \
  internal/dal/token.go \
  internal/service/auth.go \
  internal/service/user.go \
  internal/service/cron_job.go \
  internal/handler/v1/auth.go \
  internal/handler/v1/user.go \
  internal/handler/v1/resp/cron_job.go \
  internal/middleware/auth.go \
  internal/cronjob/manager.go \
  internal/cronjob/manager_test.go \
  pkg/jwt/jwt.go \
  pkg/jwt/jwt_test.go
do
  sed -i '' 's/\buint64\b/int64/g' "$f"
done
```

- [ ] **Step 6: 确认 etcd 的 uint64 未被误改**

Run: `grep -n "uint64" pkg/etcd/balancer.go`
Expected: 第 25 行 `return instances[idx%uint64(len(instances))], nil` 仍在（这是哈希取模，与 DB 无关）

- [ ] **Step 7: 编译**

Run: `go build ./...`
Expected: 编译通过。若仍报错，按提示逐个修正剩余 `uint64`（注意 `resp/cron_job.go` 的两处 `ID` 是 API 响应字段）

- [ ] **Step 8: 跑全量测试**

Run: `go test ./...`
Expected: 全部 PASS。`pkg/jwt/jwt_test.go` 的 `assert.Equal(t, int64(1), claims.UserID)` 与 `internal/cronjob/manager_test.go` 的 mock 签名都已被 Step 5 同步改掉

**Task 5 完成。** 变更文件：`cmd/gen/main.go`、`internal/dal/model/*.gen.go` 与 `internal/dal/query/*.gen.go`（14 个生成文件）、`internal/dal/{user,permission,cron_job,token}.go`、`internal/service/{auth,user,cron_job}.go`、`internal/handler/v1/{auth,user}.go`、`internal/handler/v1/resp/cron_job.go`、`internal/middleware/auth.go`、`internal/cronjob/{manager,manager_test}.go`、`pkg/jwt/{jwt,jwt_test}.go`。交由用户手动提交，不要执行任何 git 命令。

这个任务的改动面最大且必须整体生效 —— 生成代码与手写代码的类型不一致时无法编译，所以提交时这批文件应作为一个整体。

---

### Task 6: 端到端验证关键路径

编译通过不等于跑得通。这一步用真实 PG 验证三个最可能出问题的地方：`ON CONFLICT` upsert、JSONB 读写、软删除联合唯一索引。

**Files:**
- Test: `internal/dal/postgres_integration_test.go`（新建）

**Interfaces:**
- Consumes: Task 5 生成的 `query.Use(db)` 与 `model.*`；Task 4 建好的表
- Produces: 一个可重复运行的集成测试，`go test ./internal/dal/ -tags=integration`

**背景：** 项目现有 15 个测试文件都不连数据库（`internal/dal/captcha_test.go` 用的是 miniredis），DAL 层没有任何集成测试覆盖。这个任务补上最小验证，不追求完整覆盖。

- [ ] **Step 1: 写集成测试**

新建 `internal/dal/postgres_integration_test.go`。用构建标签隔离，避免污染 `go test ./...`：

```go
//go:build integration

package dal

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=postgres password=password dbname=gapi sslmode=disable TimeZone=Asia/Shanghai"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{TablePrefix: "sys_", SingularTable: true},
		TranslateError: true,
	})
	require.NoError(t, err)
	return db
}

// UpsertJob 依赖 clause.OnConflict，PG 下编译为 ON CONFLICT，
// 必须有匹配的唯一索引 idx_name 才生效。
func TestUpsertJobOnConflict(t *testing.T) {
	db := testDB(t)
	d := &CronJobDal{DB: db}
	ctx := context.Background()
	name := fmt.Sprintf("test_job_%d", time.Now().UnixNano())
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_cron_job WHERE name = ?", name)
	})

	job := &model.CronJob{
		Name:        name,
		Description: "first",
		Interval:    "0 0 * * * *",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, d.UpsertJob(ctx, job))

	job2 := &model.CronJob{
		Name:        name,
		Description: "second",
		Interval:    "0 30 * * * *",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	require.NoError(t, d.UpsertJob(ctx, job2))

	got, err := d.FindByName(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, "second", got.Description)
	assert.Equal(t, "0 30 * * * *", got.Interval)

	var count int64
	db.Model(&model.CronJob{}).Where("name = ?", name).Count(&count)
	assert.Equal(t, int64(1), count, "upsert 不应产生第二行")
}

// completed_tours 在 PG 中是 JSONB，验证 datatypes.JSONSlice 读写往返。
func TestCompletedToursJSONB(t *testing.T) {
	db := testDB(t)
	d := &UserDal{DB: db}
	ctx := context.Background()
	email := fmt.Sprintf("jsonb_%d@test.dev", time.Now().UnixNano())
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user WHERE email = ?", email)
	})

	u := &model.User{
		Username:     email,
		PasswordHash: "x",
		Email:        email,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	require.NoError(t, d.Create(ctx, u))
	require.NotZero(t, u.ID, "BIGSERIAL 应回填主键")

	require.NoError(t, d.UpdateCompletedTours(ctx, u.ID, []string{"welcome", "dashboard"}))

	got, err := d.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"welcome", "dashboard"}, []string(got.CompletedTours))
}

// uk_user_role 含 deleted_at，软删除后应能重新插入同一组合。
func TestSoftDeleteAllowsReinsert(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	q := query.Use(db)

	var roleID int64
	require.NoError(t, db.Raw("SELECT id FROM sys_role WHERE code = 'user'").Scan(&roleID).Error)
	require.NotZero(t, roleID, "seed 数据缺失，先执行 migrations/003")

	email := fmt.Sprintf("softdel_%d@test.dev", time.Now().UnixNano())
	u := &model.User{
		Username: email, PasswordHash: "x", Email: email,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	}
	require.NoError(t, q.User.WithContext(ctx).Create(u))
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user_role WHERE user_id = ?", u.ID)
		db.Exec("DELETE FROM sys_user WHERE id = ?", u.ID)
	})

	ur := &model.UserRole{UserID: u.ID, RoleID: roleID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	require.NoError(t, q.UserRole.WithContext(ctx).Create(ur))

	_, err := q.UserRole.WithContext(ctx).Where(q.UserRole.ID.Eq(ur.ID)).Delete()
	require.NoError(t, err)

	ur2 := &model.UserRole{UserID: u.ID, RoleID: roleID, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	assert.NoError(t, q.UserRole.WithContext(ctx).Create(ur2), "软删除后应允许重新插入")
}
```

- [ ] **Step 2: 运行集成测试**

Run: `go test ./internal/dal/ -tags=integration -v -run 'TestUpsertJobOnConflict|TestCompletedToursJSONB|TestSoftDeleteAllowsReinsert'`
Expected: 3 个测试全部 PASS

如果 `TestUpsertJobOnConflict` 报 `there is no unique or exclusion constraint matching the ON CONFLICT specification`，说明 `sys_cron_job` 的 `idx_name` 唯一索引没建上，回到 Task 3 检查。

- [ ] **Step 3: 确认默认测试套件未被影响**

Run: `go test ./...`
Expected: 全部 PASS，且 `internal/dal` 不尝试连接 PG（integration 标签未启用）

- [ ] **Step 4: 启动服务做一次真实冒烟**

```bash
docker compose -f deploy/docker/docker-compose.yaml up -d
go run ./cmd/server
```

Expected: 启动日志无数据库错误。cron 模块在启动时会 `UpsertJob` 注册任务，能起来就说明 PG 写入通路正常。确认后 Ctrl-C 退出。

**Task 6 完成。** 变更文件：`internal/dal/postgres_integration_test.go`（新建）。交由用户手动提交，不要执行任何 git 命令。

---

### Task 7: 清理残留的 MySQL 痕迹

**Files:**
- Modify: `.github/workflows/test.yml`
- Modify: 任何仍含 mysql 字样的文件（由 Step 1 的检索结果决定）

**Interfaces:**
- Consumes: 前六个任务的全部改动
- Produces: 仓库中不再有 MySQL 引用（除本计划文档本身）

- [ ] **Step 1: 全仓检索残留**

注意检索范围**不含 `go.mod` / `go.sum`** —— 那里的两条 mysql `// indirect` 条目是 `gorm.io/gen` 的传递依赖，属于预期状态，见 Global Constraints。

```bash
grep -rn -i "mysql\|utf8mb4\|innodb\|3306" \
  --include="*.go" --include="*.yaml" --include="*.yml" \
  --include="*.sql" --include="*.md" --include="Dockerfile" \
  --include="Makefile" . \
  | grep -v docs/superpowers/plans
```

Expected: 无输出。若有命中，逐个改掉（注释、文档措辞等）。

- [ ] **Step 2: 确认 mysql 未被编译进二进制**

上一步只查了源码文本，这一步验证实际依赖图 —— 确认没有任何包 import mysql driver：

```bash
go list -deps ./... | grep -i mysql || echo "✓ 依赖图中无 mysql"
```

Expected: `✓ 依赖图中无 mysql`。若有输出，说明还有代码在 import mysql driver，回到 Task 1 Step 7 检查。

- [ ] **Step 3: 给 CI 加 postgres 服务**

当前 `.github/workflows/test.yml` 只跑 `go test ./...` 和 `go build ./...`，不连数据库，所以默认套件不需要 PG。但加上 service 容器后可以顺带跑集成测试，防止 DDL 与代码漂移。把 `jobs.test.steps` 之前插入 `services`，并追加一个 step：

```yaml
jobs:
  test:
    runs-on: ubuntu-latest

    services:
      postgres:
        # 保持与 deploy/docker/docker-compose.yaml 中的 tag 一致（见 Task 2 Step 1）
        image: postgres:16-alpine
        env:
          POSTGRES_PASSWORD: password
          POSTGRES_USER: postgres
          POSTGRES_DB: gapi
        ports:
          - 5432:5432
        options: >-
          --health-cmd "pg_isready -U postgres -d gapi"
          --health-interval 5s
          --health-timeout 5s
          --health-retries 10

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Run tests
        run: go test ./...

      # 003 依赖 admin/user 角色已存在，因此拆成三步：建表 → 补角色 → 灌权限
      - name: Apply schema migrations
        env:
          PGPASSWORD: password
        run: |
          psql -h 127.0.0.1 -U postgres -d gapi -v ON_ERROR_STOP=1 \
            -f migrations/001_create_cron_tables.sql
          psql -h 127.0.0.1 -U postgres -d gapi -v ON_ERROR_STOP=1 \
            -f migrations/002_create_user_permission_tables.sql

      - name: Seed base roles
        env:
          PGPASSWORD: password
        run: |
          psql -h 127.0.0.1 -U postgres -d gapi -v ON_ERROR_STOP=1 -c \
            "INSERT INTO sys_role (name, code, description, sort_order, enabled, created_at, updated_at, deleted_at) VALUES ('管理员','admin','',0,TRUE,NOW(),NOW(),0), ('普通用户','user','',1,TRUE,NOW(),NOW(),0);"

      - name: Seed frontend permissions
        env:
          PGPASSWORD: password
        run: |
          psql -h 127.0.0.1 -U postgres -d gapi -v ON_ERROR_STOP=1 \
            -f migrations/003_seed_frontend_permissions.sql

      - name: Run integration tests
        run: go test ./internal/dal/ -tags=integration -v

      - name: Build
        run: go build ./...
```

三个 seed 步骤的顺序是强制的：003 的两条 `INSERT ... SELECT` 依赖 `sys_role` 中已有 `admin` 与 `user`，角色缺失时不会报错，只会静默插入 0 行，随后 `TestSoftDeleteAllowsReinsert` 会因取不到 roleID 而失败。

- [ ] **Step 4: 本地校验 workflow 语法**

Run: `python3 -c "import yaml,sys; yaml.safe_load(open('.github/workflows/test.yml')); print('yaml ok')"`
Expected: `yaml ok`

- [ ] **Step 5: 确认最终状态**

```bash
go build ./... && go test ./...
go test ./internal/dal/ -tags=integration
```

Expected: 全部 PASS

**Task 7 完成。** 变更文件：`.github/workflows/test.yml`，以及 Step 1 检索出的任何残留文件。交由用户手动提交，不要执行任何 git 命令。

---

## 验证清单

全部任务完成后逐项确认：

- [ ] `go build ./...` 通过
- [ ] `go test ./...` 全绿
- [ ] `go test ./internal/dal/ -tags=integration` 全绿
- [ ] `go list -m gorm.io/gorm gorm.io/driver/postgres gorm.io/gen github.com/jackc/pgx/v5` 输出 v1.31.2 / v1.6.0 / v0.3.28 / v5.10.0
- [ ] `go list -deps ./... | grep -i mysql` 无输出（依赖图中无 mysql）
- [ ] `go.mod` 中两条 mysql 条目带 `// indirect`（预期状态，非遗漏）
- [ ] `grep -ri mysql --include="*.go" .` 无输出
- [ ] `go run ./cmd/server` 能启动，cron 任务注册成功
- [ ] 从空库执行 `migrations/0*.sql` 能完整建起 7 张表 + seed
- [ ] `pkg/etcd/balancer.go:25` 的 `uint64` 取模逻辑未被误改

## 未包含的范围

- **数据搬迁**：按决策不做，开发库直接重建。若后续需要迁移存量数据，另需 pgloader 方案 + `uint64` 溢出校验（原 MySQL 的 `BIGINT UNSIGNED` 上界是 PG `BIGINT` 的两倍，理论上存在超界行）。
- **API 契约变更通知**：主键从 `uint64` 变 `int64` 不改变 JSON 数字表示，但如果前端有 TypeScript 强类型定义或依赖超过 2^63 的 ID，需要同步确认。
- **DAL 层完整测试覆盖**：Task 6 只覆盖三个高风险点。项目原本就没有 DAL 集成测试，补齐完整覆盖是独立工作。