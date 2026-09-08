# gapi-server

Go + Gin 的后端服务骨架。要求 **Go 1.26+**（见 `go.mod`）。

## 技术栈

| 方向 | 选型 |
| --- | --- |
| Web 框架 | gin |
| 命令行 | spf13/cobra（运维脚本入口） |
| 依赖注入 | google/wire（编译期生成） |
| 配置 | viper，多环境 + etcd 远程配置热更新 |
| 日志 | zap + lumberjack 按天切割 |
| 数据库 | PostgreSQL 18 + GORM，DAL 由 `gorm/gen` 从库反射生成 |
| 缓存 | Redis 7 |
| 服务注册发现 / 配置中心 / 分布式锁 | etcd 3.5+ |
| 定时任务 | robfig/cron，含执行记录与分布式锁互斥 |
| 鉴权 | 自签 JWT（golang-jwt）+ 本地 RBAC |
| 验证码 | wenlng/go-captcha（滑块 / 点选 / 旋转） |
| 接口文档 | swaggo |

## 架构总览

请求从中间件链进入，经 handler → service → dal 三层向下，`dal/model` 为各层共享，
`dal/query` 是 `gorm/gen` 生成的 DAO（只有 dal 层能碰）。

三个入口共用 `internal/provider` 下的 provider set：`cmd/server`（HTTP 服务）、
`cmd/cli`（cobra 运维脚本）、`cmd/gen`（DAL 代码生成）。详见 `CLAUDE.md`。

## 目录结构

```
cmd/server/          HTTP 服务入口与 wire 注入
cmd/cli/             运维脚本入口（cobra），独立 wire 注入
  commands/          命令按组分目录：system/
cmd/gen/             DAL 代码生成器（从真实数据库反射）
configs/             default.yaml + 环境覆盖(dev/prod)
deploy/docker/       docker-compose / podman-compose
docs/                swag 生成的接口文档
migrations/          建表脚本，按序号手工执行（约定见 migrations/README.md）
internal/
  app/               进程级依赖聚合与释放（App=服务 / Cli=脚本 / Gen=代码生成）
  captcha/           验证码生成与校验
  config/            配置结构体与加载
  cronjob/           调度器、执行记录、分布式锁适配
  dal/               数据访问层
    model/           表模型（生成）与枚举（手写）
    query/           gorm/gen 生成的 DAO
  handler/v1/        HTTP 处理器
    req/ resp/       请求与响应结构体
  jobs/              具体定时任务实现
  middleware/        trace/i18n/validator/recovery/logger/cors/ratelimit + JWT 鉴权
  provider/          wire provider set
  router/            引擎组装与路由注册
  server/            HTTP 服务与生命周期钩子
pkg/
  database/          GORM 连接、日志、事务管理
  email/             SMTP 发信
  etcd/              客户端、注册、发现、负载均衡、动态配置、分布式锁
  jwt/               token 签发与解析
  locale/            i18n 词条（zh/en）
  logger/            zap 封装与 trace_id 辅助
  netutil/           网络工具
  redis/             客户端
  response/          统一响应与响应码
```

`pkg/` 下均为与业务无关的通用能力，只依赖 `internal/config`，不碰其他 internal 包。

## 文档分工

| 文件 | 回答什么问题 | 谁读 |
| --- | --- | --- |
| `README.md` | 这是什么？怎么跑起来？怎么配置和部署？ | 人 |
| `AGENTS.md` | 通用约定：自检、代码生成、枚举、CLI、测试、风格 | 人 + AI |
| `internal/AGENTS.md` | `internal/` 的分层方向、ctx 规范、错误处理、各类改动步骤 | 人 + AI |
| `CLAUDE.md` | 多个文件之间有哪些看不见的关系？有哪些已知缺陷？ | 主要给 AI |

四者内容不重叠，交叉引用而不复制。**改开发约定请动对应的 `AGENTS.md`，不要写回本文件**，
否则两处很快就会分叉。

`CLAUDE.md` 用 `@AGENTS.md` 与 `@internal/AGENTS.md` 把两份约定导入 AI 的上下文，
所以约定写在 `AGENTS.md` 里就会被自动遵守，写在 README 里则不会。

另有 `migrations/README.md` 专讲建表脚本的执行方式与 PostgreSQL 相关约定
（部分唯一索引、`CHECK` 约束、`OnConflict` 的 `TargetWhere` 等）。

## 运行前置依赖

| 依赖 | 版本 | 默认地址 | 是否必需 |
| --- | --- | --- | --- |
| PostgreSQL | 18 | `127.0.0.1:5432` | 必需 |
| Redis | 7 | `127.0.0.1:6379` | 必需（限流、验证码、token） |
| etcd | 3.5+ | `127.0.0.1:2379` | 必需（注册发现、配置中心、分布式锁） |

etcd 做四件事：**服务注册**、**服务发现**、**配置中心**（热更新 `HotConfig` 段）
与**分布式锁**（定时任务多实例互斥）。它是**硬依赖**：即便把 `etcd.dyn_config.enabled`
关掉，启动时仍会建连，连不上会直接报错退出。

一键起全部依赖：

```bash
docker compose -f deploy/docker/docker-compose.yaml up -d
docker compose -f deploy/docker/docker-compose.yaml --profile tools up -d   # 额外起 etcd-workbench
```

用 Podman 时把 compose 文件换成 `deploy/docker/podman-compose.yaml`。

开发环境 Redis 版本较低时不支持 `maint_notifications`，`configs/dev.yaml` 里已默认关闭。

## 快速开始

```bash
# 1. 起依赖
docker compose -f deploy/docker/docker-compose.yaml up -d

# 2. 建表 + 灌种子数据
#    注意 003 依赖 sys_role 里已有 admin/user 两个角色，所以拆成三步
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/001_create_cron_tables.sql
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/002_create_user_permission_tables.sql
docker exec gapi-postgres psql -U postgres -d gapi -c \
  "INSERT INTO sys_role (name, code, sort_order) VALUES ('管理员','admin',0), ('普通用户','user',1);"
docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < migrations/003_seed_frontend_permissions.sql

# 3. 启动
make run
```

`migrations/000_clean.sql` 会 **DROP 全部表**，只用于重置开发库，
不要对有数据的库执行。完整说明见 `migrations/README.md`。

服务默认监听 `:8080`，健康检查 `GET /api/v1/health`，
swagger UI 在非 prod 环境下位于 `/api/v1/swagger/index.html`。

七张表：`sys_cron_job` / `sys_cron_job_execution`（定时任务定义与执行记录）、
`sys_user` / `sys_role` / `sys_permission` / `sys_user_role` / `sys_role_permission`（RBAC）。

## 配置

三层叠加，后者覆盖前者：

| 层 | 来源 | 说明 |
| --- | --- | --- |
| 基础 | `configs/default.yaml` | 全部配置项与默认值，基础设施地址在此按部署环境改 |
| 环境 | `configs/{dev,prod}.yaml` | 只放**行为差异**，按 `APP_ENV` 选择，默认 `dev` |
| 远程 | etcd 的 `etcd.dyn_config.key` | 可选，运行期热更新 `HotConfig` 段 |

各环境的覆盖内容：

- `dev.yaml` — 关闭 `redis.maint_notifications`（本地 Redis 版本较低）
- `prod.yaml` — 日志不再输出到控制台

数据库账号密码、Redis / etcd 地址这类基础设施信息不放环境文件，统一改 `default.yaml`
（部署时随二进制上传的那份），或写进 etcd 远程配置。生产环境的 `jwt.secret` 留空，
**必须通过 etcd 注入**。

> **`APP_ENV=test` 会启动失败**：`config.DetermineEnvironment` 接受 `dev`/`test`/`prod`
> 三个值（其余回落 `dev`），但 `configs/` 下只有 `default`/`dev`/`prod`——
> 选到 `test` 时 merge 找不到同名文件会 panic。需要 test 环境就补一个 `configs/test.yaml`。

可热更新的只有 `HotConfig` 里的三段：`cors`、`rate_limit`、`tour`。
`DynConfig` watch 到 etcd 变更后整体替换这部分，其余配置改了要重启才生效。

> 远程配置的合并是 `viper.MergeConfig`，作用于**整份配置**而非仅 `HotConfig`，
> 也就是说 etcd 里的内容可以覆盖数据库账号密码。多环境共用同一套 etcd 集群时，
> 务必给各环境配不同的 `etcd.dyn_config.key`，否则会互相读到对方的配置。

## 鉴权

自签 JWT + 本地 RBAC，不依赖外部 SSO。请求头 `Authorization: Bearer <token>`，
`middleware.JWTAuth` 解析后把 user_id 与 username 写入 gin context
（用 `middleware.CurrentUserID` / `CurrentUsername` 取）。

access_token 默认 15 分钟，refresh_token 默认 7 天，`POST /api/v1/auth/refresh` 换新。
refresh token 存 Redis，用过即失效。

权限走 `sys_user_role` + `sys_role_permission` 两级关联，`GET /api/v1/auth/permissions`
返回当前用户的菜单与路由权限码。`ResourceType` 区分 api / frontend-menu /
frontend-route / frontend-button / data 五类。

鉴权是**各 handler 自己挂的**（结构体带 `JWTAuth gin.HandlerFunc` 字段，
在 `Register` 里对子组 `Use`），不在全局中间件链上。

> **`/api/v1/cron-jobs/*` 当前完全没有鉴权**（代码里有 TODO 注明待补），
> 任何人都能启停和手动触发定时任务。部署到公网前务必补上。

## 运维脚本 (CLI)

`cmd/cli` 是与 HTTP 服务平行的第二个入口，复用同一套配置与依赖注入。

```bash
make build-cli              # 产出 bin/gapi-server-cli
./bin/gapi-server-cli --help
./bin/gapi-server-cli version    # 打印版本信息
./bin/gapi-server-cli welcome    # 打印欢迎信息并探测基础设施
```

目前只有 `system` 组下的 `version` 与 `welcome` 两条命令，作为脚手架示例。

配置走相对路径 `./configs`，因此需在项目根目录（或二进制与 `configs/` 同级处）执行，
`APP_ENV` 同样生效。失败时命令以非零码退出，可供 crontab 或外层脚本判断。

CLI 不经过鉴权中间件（没有 HTTP 请求，也就没有 `Authorization` 头）。谁能执行脚本由
服务器登录权限控制——**能登服务器就能跑任何命令**，写有破坏性的脚本时自行加二次确认。

新增命令的步骤见 `AGENTS.md`。

## 常用命令

```bash
make run          # 启动 HTTP 服务
make run-cli      # 跑 CLI
make build        # 构建服务到 bin/
make build-cli    # 构建 CLI 到 bin/
make docker       # 构建镜像，APP_ENV / GIN_MODE 可覆盖
make clean        # 清掉 bin/
make wire         # 依赖变更后重新生成 wire_gen.go（server 与 cli）
make swagger      # 重新生成接口文档
```

Makefile **没有** `test` / `fmt` / `lint` / `check` 目标，自检四项手动跑
（`gofmt -l .` / `go vet ./...` / `go test ./...` / `go build ./...`），详见 `AGENTS.md`。

DAL 代码生成不在 Makefile 里：`go run ./cmd/gen`，且必须在仓库根目录执行。

## 部署

```bash
make build APP_ENV=prod GIN_MODE=release
```

产出静态二进制（`CGO_ENABLED=0`），连同 `configs/` 一起上传。`APP_ENV` 与 `GIN_MODE`
在构建时烧进 ldflags，运行时同名环境变量优先级更高。

工作目录需为二进制与 `configs/` 的父目录：配置用相对路径 `./configs` 读取，
i18n 词条用 `./pkg/locale`，日志默认写到 `./logs`。

> **`pkg/locale/` 必须一起上传**，否则**每个响应都会 panic**：词条目录缺失时
> `loadMessages` 静默跳过（`filepath.Walk` 吞掉错误），而 `HttpResponse` 用的是
> `MustLocalize`。panic 被 Recovery 兜成 `recoveryError`(10005)，症状是
> "接口全都返回 10005"，容易误判成业务问题。

Linux 下会自动走 grace 平滑重启（`SIGUSR2` 热重启不断连接），其他平台走标准
`ListenAndServe` + 信号监听。进程用 systemd 之类的守护工具托管。

## 错误响应

接口失败时只返回预定义的响应码与文案，**基本不含诊断信息**——没有堆栈、SQL、
连接信息或配置值。排查线上问题请到日志里按 trace_id 查，那边记录了完整的 error 链与堆栈。

每个响应都带 `X-Trace-ID` 头，值即该次请求的 trace_id；上游若已带这个头会被透传沿用，
便于跨服务串联。同一次请求的所有日志共享这个 id。

参数校验失败时 `data` 是 field → message 的映射，指出哪个字段不合法：

```json
{"code": 10002, "data": {"name": "name为必填字段"}, "message": "参数错误"}
{"code": 10006, "data": null, "message": "服务内部错误，请稍后再试"}
```

请求头 `Locale: cn|en`（key 由 `locale.locale_key` 配置，默认 `Locale`）可切文案语言，
默认 `cn`。

> 有两处**已知且当前接受**的偏差：定时任务的 panic 堆栈会经执行历史接口外泄、
> 参数类型不匹配时会返回 `strconv` 的错误原文。详见 `CLAUDE.md`。

对应的编码规范（什么不能进响应、日志该记什么）见 `internal/AGENTS.md`。

## 参与开发

改代码前请读 **`AGENTS.md`**（通用约定：自检、代码生成、枚举、测试、风格）；
动到 `internal/` 下的代码还要读 **`internal/AGENTS.md`**（分层方向、ctx 规范、
错误处理、加接口 / 定时任务 / 表的完整步骤）。跨文件的架构脉络与已知缺陷见 **`CLAUDE.md`**。

一句话版本：动手前 `AGENTS.md`，碰 `internal/` 加读 `internal/AGENTS.md`，
收工前 `gofmt -l . && go vet ./... && go test ./... && go build ./...`。
