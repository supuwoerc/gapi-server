# internal/AGENTS.md

`internal/` 下的分层规则与写法。**改 `internal/` 里的代码前读完本文件。**

全局约定（自检闭环、代码生成、并发与工具库、枚举、CLI、测试、风格）在根目录 `AGENTS.md`，
那些同样适用；
`internal/` 各包之间的架构脉络见 `CLAUDE.md`。本文件只写与分层、请求链路相关的部分。

三条最容易踩的规则，赶时间就先看这三条：

1. **ctx 一路传到底**，不在链路中段自造——`context.Background()` 只在链路起点用
2. **诊断细节（堆栈、SQL、连接串、配置）只进日志**，响应只给响应码
3. **接口定义在调用方**，service 不 import dal 本体

## 分层与依赖方向

```
handler/v1  →  service  →  dal  →  dal/query（生成的 DAO）
                  ↘        ↓
                   dal/model      ← 各层共享的类型包
```

- 业务流向单向：handler 不直接调 dal，dal 不依赖 service。
- `dal/model` 是**例外**：只放表模型与枚举，handler / service / dal / cronjob 都可以
  直接 import，不算跨层。handler 的 `resp` 包也从 model 转换出响应结构体。
- `dal/query` 是 `gorm/gen` 生成的 DAO，**只有 dal 层能碰**，service 不要直接用。
- `service` **不 import `dal` 本体**——它只依赖自己定义的仓储接口，实现由 `wire.Bind`
  在 provider 里接上。别为了省事在 service 里直接 new 一个 dal。

### 接口定义在调用方

**接口定义在调用方**，不在实现方。这是本项目的核心约定：

- `internal/handler/v1/cron_job.go` 定义 `CronJobService`，实现是 `service.CronJobService`
- `internal/service/cron_job.go` 定义 `CronJobRepository`，实现是 `dal.CronJobDal`
- 两者在 `internal/provider/cronjob.go` 用 `wire.Bind` 接起来

好处是上层只声明自己用得到的方法，测试时替一个 stub 就行。新增模块照此办理。

### model 包的文件组织

**当前是一类型一文件**（`permission_action.go` / `permission_effect.go` /
`permission_resource_type.go` / `cron_job_triggered_by.go`），加上全包共用的 `enum.go`
与 stringer 生成物。表模型是 `gorm/gen` 生成的 `*.gen.go`，不手写。

沿用这个结构即可，但注意两件事：

- **枚举的手写文件与生成的 `*.gen.go` 分开**，别把手写内容加进 `.gen.go`（会被覆盖）。
- 新增整型枚举时 `go:generate stringer` 指令紧贴类型声明，`-output` 用
  `snake_case` 对应类型名（`ResourceType` → `resource_type_string.go`）。

> 域文件涨大时可以像 stream-v2 那样按业务域合并（一个域一个文件 + 一条
> `stringer -type=A,B`），但**当前枚举数量少，不必提前重构**——一类型一文件在
> 4 个枚举时反而更好找。

## 上下文：一次请求一个 ctx，贯穿到底

**同一次请求链路上的所有方法调用与日志都必须用同一个 context**，从 handler 一路传到
service、dal。trace_id 就挂在 ctx 上，换了 ctx 就等于换了一次请求的身份——日志串不起来，
事务也会脱开。

trace_id 在**两个链路起点**注入，都用 `pkg/logger` 的 `GenerateTraceID` / `WithTraceID`：

| 起点 | 谁注入 | 说明 |
| --- | --- | --- |
| HTTP 请求 | `middleware.Trace` | 上游透传了 `X-Trace-ID` 就沿用，否则生成 |
| 定时任务调度 | `cronjob.JobManager.wrapJob` | 每次执行生成一个，串起这一次执行的全部日志 |

**别在别处生成 trace_id**，也别自己写读写 ctx 的辅助函数——`pkg/logger` 已有
`GenerateTraceID` / `WithTraceID` / `TraceIDFromContext` 三个，middleware 与 cronjob
都复用它们。

### 基本规则

- handler 里用 `c.Request.Context()` 取 ctx，**不要用 `c` 本身当 context**。
  `*gin.Context` 虽然实现了 `context.Context`，但它的值与请求 ctx 不是一回事，
  混用会让 `logger.Ctx` 取不到 trace_id。
- ctx 一律作为**第一个参数**显式往下传：`func (s *S) Do(ctx context.Context, id int64) error`。
- **禁止在请求链路中段自造 ctx**。`context.Background()` / `context.TODO()` 只允许出现在
  链路的**起点**：`main`、CLI 命令、生命周期钩子、调度器触发的任务、进程级释放。
  在 service 或 dal 里写 `context.Background()` 会同时丢掉 trace_id 与事务句柄。
- 需要超时或取消时，**从传入的 ctx 派生**，不要另起一个：

```go
// 对：派生，保留 trace_id 与上游取消信号
ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
defer cancel()

// 错：凭空造一个，trace_id 与事务全丢
ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
```

- 不要把 ctx 存进结构体字段，每次调用显式传参。dal 的 `getQuery(ctx)` 就是这个模式：
  ctx 决定用事务句柄还是普通连接（见下面「事务」）。
- 日志一律 `l.Ctx(ctx).Error(...)`——用的是哪个 ctx，日志就归属哪次请求。

### 例外：活得比请求久的协程

这是唯一允许换 ctx 的场景，但**换法有讲究**。请求 ctx 在 handler 返回的那一刻就被
HTTP 服务取消，所以直接把它交给一个后台协程，协程一启动就拿到个死 ctx：

```go
// 错：handler 返回后 ctx 立刻失效，协程里的 DB 操作全部失败
go doSomething(c.Request.Context())
```

但换成 `context.Background()` 又丢了 trace_id，日志追踪就断了。**两者都要**——
用 `context.WithoutCancel` 断开取消信号、保留 ctx 里的值：

```go
// 对：不受请求结束影响，但 trace_id 仍在，日志能与触发它的请求串起来
jobCtx := context.WithoutCancel(ctx)
go m.executeWithRecording(jobCtx, j, TriggerByManual)
```

`JobManager.TriggerManual` 就是这个写法。需要能主动中止时用
`context.WithCancel(context.WithoutCancel(ctx))` 并把 cancel 存起来
（`JobManager` 的 `cancelMap` 是这个思路）。

判断标准很简单：**这段逻辑会在响应返回后继续跑吗？** 会，才需要脱离请求 ctx；
不会，就一路沿用同一个。

上面这段裸 `go` 是 `manager.go` 的现状（它自己 recover 了 panic）。**新代码起并发用
`conc`**，见根 `AGENTS.md` 的「并发：用 conc」——ctx 规则不变，
`pool.New().WithContext(ctx)` 里的 ctx 仍要是上游传进来或由它派生的：

```go
// 对：ctx 从上游派生，并发交给 conc
p := pool.New().WithContext(ctx).WithMaxGoroutines(8)
for _, id := range ids {
	p.Go(func(ctx context.Context) error { return s.Repo.Sync(ctx, id) })
}
if err := p.Wait(); err != nil { ... }
```

**并发写同一个事务要当心**：`TransactionManager.Transaction` 的 tx 挂在 ctx 上，
把同一个带 tx 的 ctx 分给多个 goroutine 并发执行，等于并发复用一条
`*gorm.DB` 连接——`pgx` 不支持这样用。事务里要么串行，要么先并发算好数据、
再在事务内串行落库。

**收尾的落库同理**：任务被取消后仍要写最终状态，那次写库得用
`context.WithoutCancel(ctx)` 派生的 ctx，否则 ctx 已失效、状态写不进去
（`executeWithRecording` 里的 `recordCtx`）。

## 错误处理：详情进日志，响应只给码

一条铁律，分两半：

> **对外响应只暴露预定义的响应码与文案；一切诊断细节（堆栈、SQL、连接串、配置、
> 内部错误原文）只进日志。**

两边都不能省：响应里漏了细节是安全问题，日志里少了细节则线上无法排查。

### 响应侧：绝不能出现在 JSON 里的东西

| 禁止出现 | 典型来源 |
| --- | --- |
| 堆栈 / 文件路径 / 行号 | `debug.Stack()`、用 `%+v` 打印被 `errors.Wrap` 包装过的 error |
| SQL 语句、表名、列名、GORM 原始错误 | 直接把 dal 的 error 往响应里塞 |
| 数据库 / Redis / etcd 的地址、账号、密码 | 连接失败的 error 原文常带 `host:port` 与用户名 |
| 任何 `config.*` 里的值 | 把配置结构体或其字段写进 data |
| 内部错误原文（`err.Error()`） | `FailWithMessage(c, err.Error())` |
| 依赖库的内部错误 | `strconv.ParseBool: parsing "x": invalid syntax` 这类 |

**唯一例外是 validator 的字段校验信息**——`response.ParamsValidateFail` 把
`validator.ValidationErrors` 翻译成 field → message 的 map，这是给调用方看的、
用于修正入参的业务信息，可以返回。

### 正确写法

```go
// service 层：细节写日志，对外返回预定义响应码
func (s *XxxService) Do(ctx context.Context, id int64) error {
	row, err := s.Repo.FindByID(ctx, id)
	if err != nil {
		// 原始 error（可能含 SQL、连接信息）只进日志
		s.Logger.Ctx(ctx).Error("failed to find xxx",
			zap.Int64("id", id),
			zap.Error(err),          // 完整 error 链
		)
		return response.InternalError   // 对外只给码
	}
	...
}

// handler 层：交给 FailWithError，不要自己拼文案
if err := h.Service.Do(ctx, id); err != nil {
	response.FailWithError(c, err)
	return
}
```

- `response.FailWithError` 是安全的：`StatusCode` 原样透出，context 取消/超时单独区分，
  其余一律归为 `InternalError`，**不会带出 error 原文**。有内部错误时优先用它。
- `response.FailWithCode(c, response.Xxx)` 用于明确的业务失败。
- **`response.FailWithMessage` 只能传写死的文案**，绝不能传 `err.Error()` 或任何拼接了
  error 的字符串。需要让调用方知道"为什么失败"就加一个响应码，不要透原文。
  （当前全仓库零调用，保持这样最好。）
- 不要自己拼 `c.JSON`——绕开 `pkg/response` 就绕开了这层约束。

> **现状说明**：service 层目前普遍是「记日志 + `return err`」把 GORM 原始 error 往上抛，
> 而非返回 `response.InternalError`。因为 `FailWithError` 会把认不出的 error 统一归为
> `InternalError`，**对外表现一致、没有泄露**，所以这是风格偏离而非安全问题。
> 新代码按上面的写法来，碰到旧代码不必专门改。

### 日志侧

```go
l.Ctx(ctx).Error("消息",
	zap.Error(err),              // 完整 error 链
	zap.Int64("id", id),         // 定位所需的业务字段
)
```

- 必须用 `l.Ctx(ctx)` 而非 `l.Error(...)`——`Ctx` 带上 trace_id，少了它无法串联同一次
  请求，也就没法把这条日志和用户报的那次失败对上。
- 用 `zap.Error(err)` 传 error 本身，不要 `zap.String("err", err.Error())`（丢掉 error 链）。
- 字段用 `zap.String` / `zap.Int64` 等强类型 field，不要拼进 message 字符串。
- 包装 error 用 `github.com/pkg/errors` 的 `Wrap` / `Wrapf` 带上下文
  （`errors.Wrapf(err, "sync job %s", name)`），日志里就能看到完整链路。
  注意包装后的 error **仍然只能进日志**。

**堆栈不用手写**：`pkg/logger` 配了 `zap.AddStacktrace(zapcore.ErrorLevel)`，
error 及以上级别自动带 `stacktrace` 字段，warn / info 不带（实测确认）。
所以不需要每处写 `zap.StackSkip`——panic 场景例外，那时用
`zap.ByteString("stack", debug.Stack())` 记被 recover 掉的那个栈。

### 落库的错误信息同样对外可见

`sys_cron_job_execution.error` 这类**存进数据库、又会通过接口返回**的错误字段，
等同于响应内容——写入时就要脱敏，不能把堆栈原文存进去。判断标准是：
这个字段会不会出现在某个 `resp.*` 结构体里？会，就按响应侧的规则处理。

具体到定时任务：**`SystemJob.Handle` 返回的 error 会原样落库并经执行历史接口返回**，
所以它必须是干净的。需要记 SQL、连接信息这类细节就在 job 内部记日志，对外返回一个
概括性的 error：

```go
func (j *XxxJob) Handle(ctx context.Context) error {
	if err := j.repo.Do(ctx); err != nil {
		j.logger.Ctx(ctx).Error("xxx job failed", zap.Error(err))
		return errors.New("同步数据失败")   // 落库并对外可见，不含内部细节
	}
	return nil
}
```

> 这条链路当前有一个**已知且接受**的破口：panic 的堆栈会经此外泄。详见 `CLAUDE.md`
> 的「错误信息外泄的那条链路」。不要顺手改。

### 事务：靠 ctx 穿透，不传 *gorm.DB

`database.TransactionManager.Transaction` 把事务句柄塞进 ctx，dal 层每个方法通过
`getQuery(ctx)` → `database.TxFromContext` 取出，没有事务时回退到注入的 `DB`。

```go
// service 层：同一个 ctx 传给多个 dal 方法，它们就在同一事务里
err := s.TxManager.Transaction(ctx, func(ctx context.Context) error {
	if err := s.Repo.CreateA(ctx, a); err != nil {
		return err            // 返回 error 即回滚
	}
	return s.Repo.UpdateB(ctx, b)
})
```

- **不要层层传递 `*gorm.DB`**，事务信息走 ctx。
- dal 方法必须用传入的 ctx（`getQuery(ctx)`），**自己造 ctx 会静默脱离事务**——
  不报错，但那条语句不受回滚保护。这是本项目最容易踩的 ctx 陷阱。
- **Redis 型 dal 不在事务范围内**（`captcha.go` / `token.go` / `activation_code.go`
  没有 `getQuery`）。在同一个 `Transaction` 闭包里混用两类 dal 时，GORM 那部分回滚了，
  Redis 那部分不会跟着回滚，需要自己补偿。

## 常见改动的完整步骤

### 加接口

1. 在 `internal/handler/v1/` 写 handler，实现 `Register(*gin.RouterGroup)`
2. 需要鉴权就加 `JWTAuth gin.HandlerFunc` 字段，在 `Register` 里对子组
   `Use(h.JWTAuth)`（参考 `auth.go` 把公开与受保护路由分成两个组的写法）
3. 加进 `provider.HandlerSet` 的 registrar 列表
4. 写 swagger 注解（现有 handler 都有 `@Summary` / `@Tags` / `@Success` / `@Router`，
   照抄邻近接口即可；受保护接口建议补
   `@Param Authorization header string true "Bearer {token}"`——当前没有接口写了这一行）
5. `make wire && make swagger`

**没有路由测试**，所以漏挂鉴权、路由路径写错都不会被任何检查发现，第 2 步要自己核对。
现有 `/api/v1/cron-jobs/*` 就是全部无鉴权的（代码里有 TODO 注明）。

### 加定时任务

在 `internal/jobs/` 实现 `cronjob.SystemJob`（`Name` / `Interval` / `ExecutionMode` /
`Handle`），加进 `provider.ProvideSystemJobs`，`make wire`。任务定义启动时自动同步进
`sys_cron_job`，可通过接口启停与手动触发。

注意 `dal.UpsertJob` 的 `DoUpdates` 不含 `enabled` 列——那是运维在库里调的状态，
不能被代码里的默认值覆盖回去。改 upsert 的更新列时别把它加进去。

`Handle` 返回的 error 会落库并对外可见，写法见上面「落库的错误信息同样对外可见」。

`Handle` 收到的 ctx 里**已经带了 trace_id**（调度触发是新生成的，手动触发是那次 HTTP
请求的），所以 job 内部一律 `j.logger.Ctx(ctx).Xxx(...)`——这样 job 自己的日志能和
`JobManager` 记的"开始/完成"、service 层的落库日志按同一个 trace_id 串起来。
用 `j.logger.Info(...)` 就断了，而且**能编译通过**（`cronjob.Logger` 接口保留了
不带 ctx 的方法给 `CronLogger` 用），没有测试拦得住，靠自己。

`Interval` 是 6 位含秒的 cron 表达式（`cron.WithSeconds()`）。

### 加表

1. 写 `migrations/00X_*.sql`，约定见 `migrations/README.md`
2. 把脚本执行到库里
3. 在**仓库根目录**跑 `go run ./cmd/gen` 重新生成 DAL
4. 若新表带枚举列，在 `internal/dal/model/` 手写枚举（转发给 `enum.go`，见根 `AGENTS.md`）
5. 写 dal 时沿用 `getQuery(ctx)` 模式

第 3 步的顺序不能反：**`cmd/gen` 从真实数据库反射生成**，库没更新就会生成旧结构。

`migrations/README.md` 里有几条容易踩的约定，动表结构前先读：唯一索引一律写成
部分索引 `WHERE deleted_at = 0`（软删除的行仍在表里，普通唯一索引会让被删的
username / code 永久不可复用）、针对部分索引做 upsert 时 `clause.OnConflict` 必须补
`TargetWhere`、枚举列用 `CHECK` 而非原生 `ENUM`。

> 本项目**有本地 RBAC**：`sys_user` / `sys_role` / `sys_permission` /
> `sys_user_role` / `sys_role_permission` 五张表，JWT 自签（`pkg/jwt`），
> 与 stream-v2 依赖外部 SSO 的模式不同。
