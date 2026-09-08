# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Go + Gin 后端服务骨架（PostgreSQL + Redis + etcd）。

@AGENTS.md
@internal/AGENTS.md

上面两份**随本文件自动导入**，是改代码必须遵守的规则，优先级高于本文件——
`AGENTS.md` 是通用约定，`internal/AGENTS.md` 覆盖 `internal/` 下的分层与请求链路。
若上下文里没看到它们的内容，手动读一次。

`README.md`（项目背景、怎么跑起来、配置、鉴权、部署）**不会自动加载**，需要时自己读。
四份文档的完整分工见 README 的「文档分工」一节。

本文件写命令、代码生成，以及**跨多个文件才能看清的架构脉络**，不重复上面三份的内容。

## 常用命令

```bash
make run            # 启动 server (go run ./cmd/server)
make run-cli        # 启动 CLI
make build          # 构建到 bin/ (注入 Version/BuildTime/AppEnv/GinMode 等 ldflags)
make build-cli
make docker         # APP_ENV / GIN_MODE 可覆盖
```

Makefile **没有** `test` / `fmt` / `lint` / `generate` 目标，这些直接用 go 命令：

```bash
go test ./...                              # 单元测试
go test ./internal/cronjob/ -run TestTraceSuite -v   # 跑单个测试
go test ./internal/cronjob/ -run 'TestTraceSuite/TestScheduledRunSharesTraceID' -v  # suite 里的单个用例
go test -race ./internal/cronjob/
go test ./... -count=1                     # 绕过缓存
go build ./... && go vet ./...
gofmt -l .                                 # 列出未格式化的文件
```

仓库无 `.golangci.yml`，也没有配置 linter，`go vet` 是唯一的静态检查。

### 集成测试

`internal/dal/postgres_integration_test.go` 带 `//go:build integration`，需要一个已建表并灌过 seed 的 PostgreSQL：

```bash
go test ./internal/dal/ -tags=integration -v      # DSN 默认连本地 gapi 库，可用 TEST_PG_DSN 覆盖
go vet -tags=integration ./internal/dal/          # 只检查能否编译，不需要 DB
```

它不会在缺少 DB 时 skip，而是直接失败。起依赖：`docker compose -f deploy/docker/docker-compose.yaml up -d`（PG / Redis / etcd；`--profile tools` 额外起 etcd-workbench）。`deploy/docker/` 下另有 `podman-compose.yaml`。

建表顺序有讲究，见 `.github/workflows/test.yml`：**001 → 002 → 手动插入 admin/user 角色 → 003**。003 的两条 `INSERT ... SELECT` 依赖 `sys_role` 里已有这两个角色，角色缺失时不报错、只静默插入 0 行。

## 代码生成

四类生成物，**都不要手改**：

| 生成物 | 命令 | 说明 |
| --- | --- | --- |
| `cmd/*/wire_gen.go` | `make wire` | 依赖注入 |
| `internal/dal/query/*.gen.go`、`internal/dal/model/*.gen.go` | `cd cmd/gen && go run .` | gorm/gen **从真实数据库反射生成**，跑之前库要是最新 schema |
| `*_string.go` | `go generate ./...` | stringer |
| `docs/` | `make swagger` | swag |

`wire` 的 `go:generate` 用的是 `go run -mod=mod`，不需要全局装 wire。但 `stringer` 和 `swag` 需要自己装（当前环境都没装）。

只有 `*_string.go` 有「生成物是否最新」的编译期守卫（stringer 生成的 `_()` 函数里有 `_ = x[ResourceTypeAPI-1]` 这类断言，常量值变动后不重新生成会编译报错）。`wire_gen.go` 与 `docs/` 漏跑不会被任何检查发现。

## 架构脉络

分层方向与依赖约束见 `internal/AGENTS.md`，本节只写跨文件的关系。

### 三个入口，共用一套 provider

`cmd/server`（HTTP）、`cmd/cli`（cobra 运维脚本）、`cmd/gen`（DAL 代码生成）各有独立的 wire 注入，共享 `internal/provider` 下的 provider set。`BaseInfraSet` 是公用部分，`InfraSet` 额外挂 server 专用的 `etcd.NewDynConfig` 与 `etcd.NewRegistry`，`CliInfraSet` 则不含这两个。依赖聚合体分别是 `app.App` / `app.Cli` / `app.Gen`，各自的 `Close()` 统一释放连接。

`cmd/gen` 自己也走 wire 拿 DB 连接——**它是代码生成器，却依赖运行期的数据库**，这个环形关系在 provider 里看不出来。

**改公用 provider set 会同时影响三个入口**，`make wire` 会一并重新生成三处 `wire_gen.go`，别只检查一处。

### 生命周期钩子

server 和 CLI 各有一套钩子接口，**别混用**：

- `server.IServerHook`：`OnStart` / `OnReady` / `OnStop`，嵌 `BaseServerHook` 可只实现关心的方法
- `app.ICliHook`：`OnInit` / `OnClose`，嵌 `BaseCliHook`

时序在 `internal/server/server.go`：`OnStart` 在监听前串行执行，失败即 `Fatal`；`OnReady` 在后台协程里轮询 TCP 拨号确认端口可连后才触发，10s 拨不通就跳过（不 Fatal）；`OnStop` **逆序**执行。Linux 上走 `gracehttp`（平滑重启），其他平台走标准 `ListenAndServe` + 信号监听。

`etcd.Discovery` / `etcd.DynConfig` 同时实现了两套接口（`OnStart`/`OnStop` 和 `OnInit`/`OnClose`），所以能同时挂在 server 和 CLI 上。新增钩子在 `internal/provider/hook.go` 注册。

### 配置的两阶段加载

加载分两步，这决定了新配置项能在多早被用到：

1. `BootstrapConfig` —— 本地文件即可解析出的部分，用于先把 etcd 客户端与 logger 拉起来
2. `NewConfig` —— 借 etcd 客户端 merge 远程配置，产出完整的 `Config`

想加"启动早期就要用"的配置项（影响 logger 或 etcd 建连本身的），得动第一步；否则只加进 `Config` 即可。分层规则与 `HotConfig` 的坑见 README。

### 定时任务运行时

`internal/cronjob` 是调度框架，`internal/jobs` 放具体任务。启动时把代码里的任务定义 upsert 进 `sys_cron_job`，并按库里的 `enabled` 决定是否注册——**`UpsertJob` 的 `DoUpdates` 只更新 interval / description / updated_at，不含 `enabled`**，那是运维在库里调的状态，不能被代码里的默认值覆盖回去。改 upsert 的更新列时别把它加进去。

每次执行前抢 etcd 分布式锁（`LockerAdapter`，固定 prefix `cron` 与 60s TTL），多实例只跑一个。执行记录落 `sys_cron_job_execution`：panic 记 `panic`，ctx 被取消记 `cancelled`。HTTP 手动触发是**异步**的，立即返回；`force=false` 时非并发模式下正在运行会返回 `ErrJobRunning` → `response.Busy`。

停止时先 `cron.Stop()` 不再调度，再 cancel 所有在跑的任务，最多等 `cron.shutdown_timeout` 秒。

### ctx 在两条任务触发路径上的差异

规则与写法见 `internal/AGENTS.md` 的「上下文」。这里只记**代码现状**：两条路径汇入同一个 `executeWithRecording`，但 ctx 来源不同，看单个函数发现不了。

| 触发方式 | ctx 来源 | trace_id | 为什么是这个 |
| --- | --- | --- | --- |
| 调度器（`wrapJob`） | `context.WithCancel(logger.WithTraceID(context.Background(), logger.GenerateTraceID()))` | **每次执行新生成一个** | 与任何请求无关，是链路起点；cancel 存进 `cancelMap` 供 `Stop()` 中止 |
| HTTP 手动触发（`TriggerManual`） | `context.WithoutCancel(ctx)` | 沿用触发它的那次请求的 | 脱开请求生命周期，但保留 trace_id |

于是两条路径都能按 trace_id 串起一次执行：`manager` / job 内部 / service 落库三处的日志挂的是同一个 id。`executeWithRecording` 里还兜了一层——ctx 里没有 trace_id 时补一个，保证任何调用方进来都有。

收尾落库另外派生了 `recordCtx := context.WithoutCancel(ctx)`：job 被取消时 `ctx` 已失效，用它写库会失败，最终状态就丢了。

`internal/cronjob/trace_test.go` 守着两个方向：一次调度内 trace_id 共享（且注册期日志不串上执行期 id）、手动触发的请求 trace_id 透传且不被误判为 cancelled。

其余 `context.Background()` 的用法都是**正确**的，别顺手改：`server.go`（生命周期钩子与关停）、`config.go` 与 `etcd.go`（启动期建连）、`app/*.go`（进程级释放）——这些都在请求链路之外，本就是链路起点。

> **一处剩余例外**：`cronjob.CronLogger`（适配 robfig/cron 的日志）**记不了 trace_id**，因为 `cron.Logger` 的方法签名里没有 ctx。所以 `cronjob.Logger` 接口在 `Ctx` 之外仍保留不带 ctx 的 Info/Debug/Warn/Error 四个方法，专供它使用。影响有限——它记的是调度器自身的事件（chain 跳过/延迟、panic 兜底），不属于任何一次具体执行。**但这也意味着 `m.logger.Info(...)` 这种漏写 `Ctx` 的调用能编译通过。**

### 事务穿透与两类 dal

`database.TransactionManager.Transaction(ctx, fn)` 把 tx 塞进 ctx，dal 层用 `getQuery(ctx)`（内部 `database.TxFromContext(ctx, d.DB)`）取出——在事务里拿到 tx，否则回落普通连接。所以 service 组合多个 dal 调用时只要传同一个 ctx 就在同一事务里。写法见 `internal/AGENTS.md`。

`internal/dal` 下混着两类 dal，**只有第一类涉及事务**：

- GORM 型（`cron_job.go` / `user.go` / `permission.go`）：有 `getQuery(ctx)`，走上面的穿透机制
- Redis 型（`captcha.go` / `token.go` / `activation_code.go`）：直接用 redis client，没有 `getQuery`，也不受 `Transaction` 影响

在同一个 `Transaction` 闭包里混用两类时要注意：GORM 那部分回滚了，Redis 那部分**不会**跟着回滚。

### 路由注册链

handler 实现 `router.Registrar`（一个 `Register(r *gin.RouterGroup)` 方法）→ 加进 `provider.HandlerSet` 的 registrar 列表 → 重新生成 wire。`V1Handlers` 遍历所有 registrar 完成注册，全部挂在 `/api/v1` 下。

**路由表是各 handler 的 `Register` 各自拼出来的，代码里没有集中的路由清单**——想知道现有路由只能逐个看 handler。也没有 `router` 层的测试，所以路由意外变化、漏挂鉴权都不会被任何检查发现。

鉴权是 handler 自己挂的（`AuthHandler` / `UserHandler` 结构体有 `JWTAuth gin.HandlerFunc` 字段，在 `Register` 里对子组 `Use(h.JWTAuth)`），不在 `router.NewEngine` 的全局链上。注意 provider 注入的是裸 `gin.HandlerFunc` 而非具名类型。

**`/api/v1/cron-jobs/*` 全部无鉴权**（`internal/handler/v1/cron_job.go` 有 TODO 注明待补），意味着任何人都能启停和触发定时任务。

### 错误信息外泄的那条链路

`sys_cron_job_execution.error` 是**存进数据库、又会经接口返回**的字段，等同于响应内容：

```
manager.go   jobErr
  → service.RecordEnd            errMsg = jobErr.Error()
  → dal.FinishExecution          写入 sys_cron_job_execution.error（TEXT）
  → resp.CronJobExecutionItem    Error *string `json:"error"`   ← GET /cron-jobs/:name/executions
```

规则见 `internal/AGENTS.md`，这里只记**代码现状**：以下两处不符合规则，**已知且当前接受，不要顺手"修"**。

**一、panic 堆栈会经执行历史接口外泄。** `manager.go` 的 recover 分支用 `errors.Errorf("panic: %v\n%s", r, stack)` 把完整堆栈拼进 `jobErr`，走完上面整条链路。实测一次 panic 落库约 1615 字节，含 `goroutine N [running]:`、`.go:` 行号与本机绝对路径。`error` 列是 `TEXT`，不会截断。

**二、`response.ParamsValidateFail` 的三条兜底路径返回 `err.Error()` 原文。** 第一条（`err` 不是 `validator.ValidationErrors`）在生产就能触发：`ShouldBindQuery` / `ShouldBindUri` 遇到**类型不匹配**时返回的是 `*strconv.NumError`。实测 `?page=abc` 的响应体是：

```json
{"code":10002,"data":null,"message":"strconv.ParseInt: parsing \"abc\": invalid syntax"}
```

`req` 包里的 `Force bool`、`Page int`、`PageSize int` 都是触发入口，16 个调用点共享这个行为。修它会变更现有接口的错误文案，所以维持现状。

对照之下 `response.FailWithError` 是安全的：实测喂它 GORM 原始错误、`errors.Wrap` 包装过的错误、以及含 `password=secret` 的伪造连接串，响应统一是 `{"code":10006,"data":null,"message":""}`。

这两处都**没有测试守着**（`pkg/response/locale_test.go` 只管响应码与词条，不碰这两条路径），
`internal/handler/v1/resp` 整个包无测试。

