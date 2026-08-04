# Migrations

目标数据库：PostgreSQL 18。当前没有自动化 migration runner，按序号手工执行。

## 首次初始化

脚本按序号执行，整段流程等价于「重置」：`000_clean.sql` 会先 DROP 全部表，
随后 001/002 建表、003 补种子数据。**在已有数据的库上执行会清空全部数据**，
仅适用于空库或可丢弃的开发库，不要对存有数据的库执行。

此前"重跑会在 `CREATE TABLE` 处报错中断"的行为已不存在——000 会先清掉旧表，
整段重跑会静默成功。需要重置时直接整段重跑即可。

```bash
docker compose -f deploy/docker/docker-compose.yaml up -d postgres

for f in migrations/0*.sql; do
  docker exec -i gapi-postgres psql -U postgres -d gapi -v ON_ERROR_STOP=1 < "$f"
done
```

使用 Podman 时把 `docker compose` 换成 `podman compose`、compose 文件换成
`deploy/docker/podman-compose.yaml`，其余相同。

### 003 的前置依赖

`003_seed_frontend_permissions.sql` 末尾两条 `INSERT ... SELECT` 依赖 `sys_role`
中已存在 `admin` 与 `user` 两个角色，否则会**静默插入 0 行**（不报错）。空库需先补角色：

```bash
docker exec gapi-postgres psql -U postgres -d gapi -c \
  "INSERT INTO sys_role (name, code, sort_order) VALUES ('管理员','admin',0), ('普通用户','user',1);"
```

执行完 003 后应得到：`admin` 20 条角色权限（10 menu + 10 route），
`user` 12 条（20 条中排除 4 个 admin menu 与 4 个 admin route）。

## 约定

- 索引名在 PostgreSQL 中是 schema 级唯一的（MySQL 中仅表级唯一），
  统一带表名前缀避免冲突。
- 时间列统一 `TIMESTAMPTZ`，`created_at`/`updated_at` 带 `DEFAULT NOW()`
  （GORM 每次插入都会显式赋值，默认值只为手写 SQL 方便）。
- `deleted_at` 是 `BIGINT` 毫秒时间戳，由 `gorm.io/plugin/soft_delete` 维护，
  0 表示未删除。
- **唯一索引一律写成部分索引 `WHERE deleted_at = 0`。** 因为软删除只是把
  `deleted_at` 置为时间戳，行仍在表里，普通唯一索引会让被软删的 username /
  code 永久无法重新使用。关联表的联合唯一索引同理，用
  `(user_id, role_id) WHERE deleted_at = 0`，不要把 `deleted_at` 塞进索引列
  （那是 MySQL 没有部分索引时的 workaround，语义上拦不住同一组合软删多次）。
- 允许空串的列若要建唯一索引，需把空串一并排除（形如
  `WHERE deleted_at = 0 AND email <> ''`），否则空串之间会互相撞唯一约束。
  当前 `sys_user.email` 已是必填列，其唯一索引只需 `WHERE deleted_at = 0`。
- 枚举语义的列用 `CHECK` 约束而非原生 `ENUM` 类型：原生 enum 的值只能加不能删，
  且 gen 反向生成时会退化成 `string`；`CHECK` 则完全不影响 gen 的类型推断，
  改约束也不需要重新生成 DAL。取值来源见 `internal/dal/model/` 下的枚举定义
  与 `internal/cronjob/cronjob.go`。
- 主键用 `BIGINT GENERATED ALWAYS AS IDENTITY`（SQL 标准写法），不用 `BIGSERIAL`。
- 改动表结构后必须重新生成 DAL，注意要在**仓库根目录**执行
  （`internal/config/viper.go` 用相对路径 `./configs` 找配置）：

  ```bash
  go run ./cmd/gen
  ```

### 部分唯一索引与 GORM 的 OnConflict

PG 要求 `ON CONFLICT` 的推断谓词与部分索引的谓词匹配，否则报
`there is no unique or exclusion constraint matching the ON CONFLICT specification`。
所以针对部分唯一索引做 upsert 时，必须给 `clause.OnConflict` 补上 `TargetWhere`：

```go
clause.OnConflict{
    Columns: []clause.Column{{Name: "name"}},
    TargetWhere: clause.Where{Exprs: []clause.Expression{
        clause.Eq{Column: "deleted_at", Value: 0},
    }},
    DoUpdates: ...,
}
```

现有例子见 `internal/dal/cron_job.go` 的 `UpsertJob`。