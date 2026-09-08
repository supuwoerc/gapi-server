# AGENTS.md

全项目通用的开发约定，面向 AI 助手与开发者。**动手前读完本文件。**

| 想知道什么 | 看哪里 |
| --- | --- |
| 自检闭环、代码生成、枚举、CLI 命令、测试、风格 | 本文件 |
| 分层方向、ctx 规范、错误处理、加接口/任务/表的步骤 | **`internal/AGENTS.md`** |
| 项目背景、怎么跑起来、配置与部署 | `README.md` |
| 跨文件的架构脉络、已知且接受的缺陷 | `CLAUDE.md` |

改 `internal/` 下的代码要连着 `internal/AGENTS.md` 一起看——请求链路上的硬性规则
（ctx 怎么传、什么不能进响应）都在那份里，本文件只留一份红线摘要。

## 每次改动的闭环

Makefile **没有** `check` / `test` / `fmt` / `lint` 目标，四项手动跑，全过才算完成：

```bash
gofmt -l .          # 输出为空才算过（这条只检查不修改）
go vet ./...
go test ./...
go build ./...
```

任何改动结束前跑一次，失败就修，不要把失败留给用户。仓库没有 `.golangci.yml`，
也没有配置任何 linter，`go vet` 是唯一的静态检查。

PostgreSQL / Redis / etcd 是运行期硬依赖，本地未必都起着，所以**涉及基础设施的改动无法靠
`make run` 验证**——以上面四项加单元测试为准，别把"跑不起来"当成改错了。

`gofmt` 这一步最容易被 CJK 注释绊住：gofmt 按字节宽度对齐行尾注释，中文注释的结构体
字段（如 `internal/dal/model/` 里的模型）手工对齐几乎必然对不上。交给工具，不要靠肉眼
调空格——但**只格式化自己碰过的文件**，全库 `gofmt -w .` 会把无关文件塞进 diff：

```bash
gofmt -w internal/dal/model/enum.go     # 单个文件，优先用这个
```

## 代码生成：改完必须重新生成

四处生成物**不要手写**，改了源头就要重新生成，否则编译过了但运行时行为不对。

| 生成物 | 何时重新生成 | 命令 |
| --- | --- | --- |
| `cmd/*/wire_gen.go` | 增删依赖、改 provider set、改结构体字段 | `make wire` |
| `internal/dal/query/*.gen.go`、`internal/dal/model/*.gen.go` | 改了表结构 | `go run ./cmd/gen` |
| `*_string.go` | 加/改枚举常量或其行尾注释 | `go generate ./...` |
| `docs/` | 改了 handler 的 swagger 注解 | `make swagger` |

两个要点：

- **`go run ./cmd/gen` 从真实数据库反射生成**，跑之前库必须已经是最新 schema，
  否则会用旧结构覆盖掉手写不了的生成物。且必须在**仓库根目录**执行
  （`internal/config/viper.go` 用相对路径 `./configs` 找配置）。
- `make wire` 会一并重新生成三处 `wire_gen.go`（server / cli / gen），别只检查一处。

`wire` 的 `go:generate` 用的是 `go run -mod=mod`，不需要全局安装。`stringer` 与 `swag`
需要自己装，**不要手改生成物绕过**：

```bash
go install golang.org/x/tools/cmd/stringer@latest
go install github.com/swaggo/swag/cmd/swag@latest
```

只有 `*_string.go` 有「生成物是否最新」的守卫——stringer 生成的 `_()` 函数里有
`_ = x[ResourceTypeAPI-1]` 这类断言，常量值变动后不重新生成会**编译报错**。
`wire_gen.go`、`docs/` 与 DAL 生成物都没有对应检查，漏跑不会被发现。

## pkg/ 的边界

`pkg/` 是与业务无关的通用能力，构造函数接收 `*config.XxxConfig`，因此
**只允许 import `internal/config`**。不要让 `pkg/` 碰 `internal/` 下的其他包。

`internal/` 内部的分层规则见 `internal/AGENTS.md`。

## 落库的枚举

`internal/dal/model/enum.go` 收口了 `Scan` / `Value` / `Text` 的样板，
**新增枚举不要重新手写这三个方法**，转发过去即可，每个类型三行：

```go
func (i *TriggeredBy) Scan(src any) error          { return enumStringFromDB(i, src) }
func (i TriggeredBy) Value() (driver.Value, error) { return enumStringIntoDB(i) }
func (i TriggeredBy) Text() string                 { return enumStringText(i) }
```

分两族，按**落库形式**选，不要混用：

| 族 | 落库形式 | helper | 现有成员 |
| --- | --- | --- | --- |
| 字符串 | `VARCHAR` + `CHECK IN (...)` | `enumString*` | `TriggeredBy`、`PermissionAction`、`PermissionEffect` |
| 整型 | `SMALLINT` + `CHECK BETWEEN` | `enumInt*` | `ResourceType` |

`enum.go` 的 helper 是全包共用的，一处出错会影响每个枚举，所以它的边界
（多种驱动递送形式、拒绝未知类型而非静默置零、两族 `Text()` 的不同兜底）在
`enum_test.go` 里单独覆盖了。动它之前先看那几条用例。

> 这里与 stream-v2 的约定不同：那边所有枚举都是整型 + stringer，本项目三个枚举是
> **字符串落库**（VARCHAR 列），改成整型需要写迁移脚本转换存量数据，故保留。
> 新增枚举优先用整型 + stringer；确实要用字符串时归入字符串族，别自己手写 `Scan`。

### 常量值不能乱动

两族的常量值都**直接落库**，改动等于改写存量数据的含义：

- 字符串族**不能改拼写**，要同步 `migrations/` 里的 `CHECK IN (...)` 与 seed SQL
- 整型族**只能末尾追加，绝不能重排或复用已废弃的值**——重排不报错，但库里的 `2`
  会被静默解释成别的东西。新增值还要放宽 `CHECK` 的上界

`enum_test.go` 把值与文案钉死做守卫，同时校验数量与 `CHECK` 集合对得上。
新增值时在表里补一行，若测试失败说明动了已有值，先确认存量数据。

### 对外一律用 Text()，不要用 String()

整型族的 `Text()` 对未定义值兜底成 `"unknown"`——stringer 对区间外的值返回
`"ResourceType(9)"` 这类内部表示，会把 Go 类型名暴露给调用方。`String()` 保留内部
表示，只在内部日志里用（那时反而有助于排查）。

字符串族的 `Text()` 直接返回字符串本身，不做 unknown 伪装：其常量值就是文案，
库里真出现脏值时原样透出便于发现问题。

### 加响应码

三步，缺一不可：

1. 在 `pkg/response/code.go` 加常量，**文案写在行尾注释里**
2. `go generate ./...`
3. 在 `pkg/locale/zh/system.json` 与 `en/system.json` **两边都补词条**，
   id 就是行尾注释的文案

第 3 步漏了会 panic 而非返回空串（`HttpResponse` 用的是 `MustLocalize`），只补一边则
换语言时 panic。三步都有守卫：`go test ./pkg/response/`。

## 加 CLI 命令

在 `cmd/cli/commands/` 下建命令组目录（或复用 `system`），写
`newXxxCmd(...) *cobra.Command`，在该组的 `Register` 里挂上；新建组则在
`cmd/cli/root.go` 的 `init` 里加一行注册。命令需要新依赖时，加进 `app.Cli` 结构体并
确保 CLI 的 provider set 能提供，然后 `make wire`。

现有只有 `system` 组下的 `version` 与 `welcome` 两条命令，作为脚手架示例。

两条硬性要求：

- 构造 `app.Cli` 会连 PostgreSQL/Redis/etcd，**不需要基础设施的命令别去构造它**——
  否则 `--help` 也会因连不上依赖而失败。构造了就 `defer cli.Close()`。
- 配置走相对路径 `./configs`，命令必须在项目根目录（或二进制与 `configs/` 同级处）
  执行。失败时 `RunE` 返回 error 即可，会打印并以非零码退出。

CLI 不经过鉴权中间件（没有 HTTP 请求，也就没有 `Authorization` 头）。
**能登服务器就能跑任何命令**，写有破坏性的脚本时自行加二次确认。

## 测试

PostgreSQL / Redis / etcd 是硬依赖，但**单元测试不连它们**。需要 logger 时用
`zap.NewDevelopment()` 或 `zaptest/observer`（`pkg/logger/logger_test.go` 与
`internal/cronjob/trace_test.go` 都是这个路子），需要断言日志内容时用 observer。

```bash
go test ./internal/cronjob/ -run TestTraceSuite -v                          # 跑单个测试
go test ./internal/cronjob/ -run 'TestTraceSuite/TestScheduledRunSharesTraceID' -v  # suite 里的单个用例
go test ./internal/cronjob/ -race                                           # 异步逻辑加 -race
go test ./... -count=1                                                      # 绕过缓存
```

`internal/cronjob` 与 `internal/dal/model` 用 testify suite，`pkg/` 下多是普通
测试函数，跟着所在包的风格写。

写异步逻辑的测试要**用 channel 卡住时序**，别靠 `time.Sleep` 撞运气——
`trace_test.go` 里手动触发那个用例就是反例改过来的：最初让 job 直接跑完再断言，
结果 job 在 `cancel()` 之前就结束了，测试假通过（把被测的修复还原掉它依然过）。
改成「等 job 进入 Handle → cancel → 放行 job」后才真正拦住回归。

### 集成测试

`internal/dal/postgres_integration_test.go` 带 `//go:build integration`，
**缺 DB 时不 skip 而是直接失败**：

```bash
go test ./internal/dal/ -tags=integration -v     # 需要已建表并灌过 seed 的库
go vet -tags=integration ./internal/dal/         # 只检查能否编译，不需要 DB
```

改了这个文件至少要跑一次 `go vet -tags=integration`——**普通 `go vet ./...` 不会
编译带 tag 的文件**，它曾经因为 DAL 重新生成后 `Description` 变成 `*string` 而
长期编译不过，没有任何检查发现。

起依赖与建表顺序见 `README.md`。

### 已有的守卫

| 守卫 | 拦什么 |
| --- | --- |
| `TestScheduledRunSharesTraceID` | 一次调度内 trace_id 不再共享；注册期日志串上执行期 id |
| `TestManualTriggerPropagatesTraceIDAndSurvivesRequestCancel` | 手动触发丢 trace_id，或请求 ctx 取消后被误判为 cancelled |
| `TestStringEnumValuesArePinned` / `TestResourceTypeValuesArePinned` | 改动落库枚举的值或文案 |
| `TestScanAcceptsDriverRepresentations` | 枚举 `Scan` 不再兼容某种驱动递送形式 |
| `TestScanRejectsUnknownTypes` | `Scan` 遇未知类型静默置零而非报错 |
| `TestIntEnumTextFallsBackToUnknown` | 整型枚举泄露 `ResourceType(9)` 这类内部表示 |
| `TestResourceTypeScanFromSmallint`（integration） | PG 的 SMALLINT 读回枚举 |
| `TestCheckConstraintsRejectInvalidValues`（integration） | 迁移脚本的 CHECK 约束失效 |
| `pkg/response/locale_test.go` 三条 | 响应码漏写行尾注释、漏跑 `go generate`、漏补 i18n 词条（或只补一边） |

**没有守卫的面**：路由注册（路由变化、漏挂鉴权都不会被发现）、
`wire_gen.go` / `docs/` / DAL 生成物是否最新、`internal/handler/v1/resp` 无测试
（`resp` 层把 model 转成响应结构体那部分，包括 `sys_cron_job_execution.error` 的透传）。
改这些地方靠自己检查。

## 风格

- 注释用中文，写**为什么**这么做，而不是复述代码在做什么。既有代码里
  `dal.UpsertJob` 的 `TargetWhere` 说明、`router.NewEngine` 的中间件顺序说明、
  `enum.go` 的两族划分说明是范例，照这个密度来。
- 中间件顺序有讲究：`Trace` 必须最前（后续都依赖它注入的 trace_id），
  `Recovery` 在业务中间件之前。改 `router.NewEngine` 时不要打乱。
- 往 `config.HotConfig` 加字段前先确认那个配置项**真的能热生效**——它是运行期被
  替换的，只在启动时读一次的组件拿不到新值。

## 不要做的事

这几条是硬红线，细节与正确写法见 `internal/AGENTS.md`：

- **不把堆栈、SQL、连接串、配置值或 `err.Error()` 放进 API 响应**（validator 字段信息除外）
- **不用 `l.Error(...)` 代替 `l.Ctx(ctx).Error(...)`**（少了 `Ctx` 就没有 trace_id）
- **不在请求链路中段自造 ctx**（`context.Background()` 只在链路起点用），
  也不把请求 ctx 直接交给活得比请求久的协程——用 `context.WithoutCancel(ctx)`
- 不手写 `wire_gen.go`、`*_string.go`、`*.gen.go`、`docs/`
- 不重新手写落库枚举的 `Scan`/`Value`/`Text`，转发给 `enum.go`
- **不重排整型落库枚举的常量值**，也不改字符串枚举的拼写
- 不因为无关改动而全库 `gofmt -w .`
- 不在 `pkg/` 里 import `internal/` 业务包
- 不在 handler 里直接调 dal，或让 dal 依赖 service
- 不把数据库账号密码写进 `configs/{dev,prod}.yaml`（统一放 `default.yaml`）
- 不对有数据的库执行 `migrations/000_clean.sql`（它会 DROP 全部表）
- 不提交 `logs/`、`bin/`（已在 `.gitignore`）
