//go:build integration

// PostgreSQL 集成测试。需要一个已按 migrations/ 建好表并执行过 003 seed 的库：
//
//	go test ./internal/dal/ -tags=integration -v
//
// DSN 默认连本地 docker/podman 起的 gapi 库，可用 TEST_PG_DSN 覆盖。
package dal

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/supuwoerc/gapi-server/internal/dal/model"
	"github.com/supuwoerc/gapi-server/internal/dal/query"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func testDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=postgres password=password dbname=gapi sslmode=disable TimeZone=Asia/Shanghai"
	}
	// 不设 NamingStrategy：gen 生成的 model 已通过 TableName() 固定表名，
	// 额外加 TablePrefix 反而会让手写的 db.Exec 与生成代码看到不同的表。
	// TranslateError 与 pkg/database/database.go 保持一致。
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	require.NoError(t, err, "连不上 PG，检查容器是否启动或 TEST_PG_DSN")
	return db
}

// rawDB 不开 TranslateError，用于断言 PG 原始错误里的约束名。
// driver v1.6.0 会把 23514 统一翻译成 gorm.ErrCheckConstraintViolated，
// 丢掉约束名，无法区分是哪条 CHECK 触发的。
func rawDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_PG_DSN")
	if dsn == "" {
		dsn = "host=127.0.0.1 port=5432 user=postgres password=password dbname=gapi sslmode=disable TimeZone=Asia/Shanghai"
	}
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func uniqueName(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}

func roleIDsOf(t *testing.T, db *gorm.DB, code string) []int64 {
	t.Helper()
	var ids []int64
	require.NoError(t, db.Raw("SELECT id FROM sys_role WHERE code = ?", code).Scan(&ids).Error)
	require.NotEmpty(t, ids, "角色 %s 不存在，先执行 migrations/README.md 里的补角色语句", code)
	return ids
}

// UpsertJob 依赖 clause.OnConflict。idx_cron_job_name 是部分唯一索引
// (WHERE deleted_at = 0)，PG 要求推断谓词与索引谓词匹配，
// 所以 UpsertJob 必须带上 TargetWhere，否则这里会报
// "no unique or exclusion constraint matching the ON CONFLICT specification"。
func TestUpsertJobOnConflict(t *testing.T) {
	db := testDB(t)
	d := &CronJobDal{DB: db}
	ctx := context.Background()
	name := uniqueName("test_job")
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_cron_job WHERE name = ?", name)
	})

	require.NoError(t, d.UpsertJob(ctx, &model.CronJob{
		Name: name, Description: "first", Interval: "0 0 * * * *",
	}))
	require.NoError(t, d.UpsertJob(ctx, &model.CronJob{
		Name: name, Description: "second", Interval: "0 30 * * * *",
	}))

	got, err := d.FindByName(ctx, name)
	require.NoError(t, err)
	assert.Equal(t, "second", got.Description)
	assert.Equal(t, "0 30 * * * *", got.Interval)
	require.NotZero(t, got.ID, "IDENTITY 应回填主键")

	var count int64
	db.Model(&model.CronJob{}).Where("name = ?", name).Count(&count)
	assert.Equal(t, int64(1), count, "upsert 不应产生第二行")
}

// last_status 可空，"从未执行" 用 NULL 而非空串表达。
func TestLastStatusNullable(t *testing.T) {
	db := testDB(t)
	d := &CronJobDal{DB: db}
	ctx := context.Background()
	name := uniqueName("test_status")
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_cron_job WHERE name = ?", name)
	})

	require.NoError(t, d.UpsertJob(ctx, &model.CronJob{Name: name, Interval: "* * * * * *"}))

	got, err := d.FindByName(ctx, name)
	require.NoError(t, err)
	assert.Nil(t, got.LastStatus, "从未执行的任务 last_status 应为 NULL")

	require.NoError(t, d.UpdateLastRun(ctx, name, "success"))

	got, err = d.FindByName(ctx, name)
	require.NoError(t, err)
	require.NotNil(t, got.LastStatus)
	assert.Equal(t, "success", *got.LastStatus)
	assert.NotNil(t, got.LastRunAt)
}

// completed_tours 在 PG 中是 JSONB。pgx 对 jsonb 的 ScanType 落 default 分支
// 返回 string，靠 cmd/gen 的 FieldType 覆盖成 datatypes.JSONSlice[string]，
// 这里验证读写往返。
func TestCompletedToursJSONB(t *testing.T) {
	db := testDB(t)
	d := &UserDal{DB: db}
	ctx := context.Background()
	name := uniqueName("jsonb")
	email := name + "@test.dev"
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user WHERE username = ?", name)
	})

	u := &model.User{Username: name, PasswordHash: "x", Email: email}
	require.NoError(t, d.Create(ctx, u))
	require.NotZero(t, u.ID, "IDENTITY 应回填主键")

	got, err := d.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.Empty(t, got.CompletedTours, "新用户 completed_tours 应为空")

	require.NoError(t, d.UpdateCompletedTours(ctx, u.ID, []string{"welcome", "dashboard"}))

	got, err = d.FindByID(ctx, u.ID)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"welcome", "dashboard"}, []string(got.CompletedTours))
}

// uk_user_role 是部分唯一索引 (user_id, role_id) WHERE deleted_at = 0，
// 软删除后应能重新插入同一组合。
func TestSoftDeleteAllowsReinsert(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	q := query.Use(db)

	roleID := roleIDsOf(t, db, "user")[0]
	name := uniqueName("softdel")

	u := &model.User{Username: name, PasswordHash: "x", Email: name + "@test.dev"}
	require.NoError(t, q.User.WithContext(ctx).Create(u))
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user_role WHERE user_id = ?", u.ID)
		db.Exec("DELETE FROM sys_user WHERE id = ?", u.ID)
	})

	ur := &model.UserRole{UserID: u.ID, RoleID: roleID}
	require.NoError(t, q.UserRole.WithContext(ctx).Create(ur))

	_, err := q.UserRole.WithContext(ctx).Where(q.UserRole.ID.Eq(ur.ID)).Delete()
	require.NoError(t, err)

	ur2 := &model.UserRole{UserID: u.ID, RoleID: roleID}
	assert.NoError(t, q.UserRole.WithContext(ctx).Create(ur2),
		"软删除后应允许重新插入同一 (user_id, role_id)")
}

// 部分唯一索引让被软删的 username 可以重新注册。
// 普通唯一索引下这里会报 duplicate key —— 这是本次 DDL 修掉的缺陷。
func TestSoftDeletedUsernameIsReusable(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	q := query.Use(db)
	name := uniqueName("reuse")

	u1 := &model.User{Username: name, PasswordHash: "x", Email: name + "@test.dev"}
	require.NoError(t, q.User.WithContext(ctx).Create(u1))
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user WHERE username = ?", name)
	})

	_, err := q.User.WithContext(ctx).Where(q.User.ID.Eq(u1.ID)).Delete()
	require.NoError(t, err)

	u2 := &model.User{Username: name, PasswordHash: "x", Email: name + "@test.dev"}
	assert.NoError(t, q.User.WithContext(ctx).Create(u2),
		"软删除后同名同邮箱应可重新注册")
}

// idx_user_email 排除空串，多个不填邮箱的用户应能共存。
func TestMultipleUsersWithEmptyEmail(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	q := query.Use(db)
	n1, n2 := uniqueName("noemail_a"), uniqueName("noemail_b")
	t.Cleanup(func() {
		db.Exec("DELETE FROM sys_user WHERE username IN (?, ?)", n1, n2)
	})

	require.NoError(t, q.User.WithContext(ctx).Create(&model.User{Username: n1, PasswordHash: "x"}))
	assert.NoError(t, q.User.WithContext(ctx).Create(&model.User{Username: n2, PasswordHash: "x"}),
		"email 为空串的第二个用户不应撞唯一约束")
}

// resource_type 是 SMALLINT，pgx 以 int16 递送 —— 验证 ResourceType.Scan
// 的 int16 分支。MySQL 的 TINYINT 走 int64，故此路径 PG 独有。
func TestResourceTypeScanFromSmallint(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	q := query.Use(db)

	perms, err := q.Permission.WithContext(ctx).
		Where(q.Permission.ResourceType.Eq(model.ResourceTypeFrontendMenu)).
		Limit(1).Find()
	require.NoError(t, err, "SMALLINT -> ResourceType 扫描失败，检查 Scan 的 int16 分支")
	require.NotEmpty(t, perms, "seed 数据缺失，先执行 migrations/003")

	assert.Equal(t, model.ResourceTypeFrontendMenu, perms[0].ResourceType)
	assert.NotEmpty(t, perms[0].Code)
	assert.Equal(t, model.PermissionActionRead, perms[0].Action, "003 seed 的 action 应为 read")
}

// 权限查询链路：JOIN + IN + 枚举比较，覆盖 int64 切片参数与 effect 解析。
func TestFindCodesByRoleIDsAndResourceType(t *testing.T) {
	db := testDB(t)
	ctx := context.Background()
	pd := &PermissionDal{DB: db}

	adminIDs := roleIDsOf(t, db, "admin")
	codes, err := pd.FindCodesByRoleIDsAndResourceType(ctx, adminIDs, model.ResourceTypeFrontendMenu)
	require.NoError(t, err)
	assert.Len(t, codes, 10, "admin 应有 10 个 menu 权限")
	assert.Contains(t, codes, "dashboard")
	assert.Contains(t, codes, "admin:users")

	userIDs := roleIDsOf(t, db, "user")
	userCodes, err := pd.FindCodesByRoleIDsAndResourceType(ctx, userIDs, model.ResourceTypeFrontendMenu)
	require.NoError(t, err)
	assert.Len(t, userCodes, 6, "user 应有 6 个 menu 权限（10 个中排除 4 个 admin）")
	assert.NotContains(t, userCodes, "admin:users")

	// 空切片走短路分支，不应发 SQL
	empty, err := pd.FindCodesByRoleIDsAndResourceType(ctx, nil, model.ResourceTypeFrontendMenu)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// CHECK 约束应拦住非法枚举值。这些是 DB 层的最后一道防线，
// Go 侧的常量约束不住直接写库的场景。
func TestCheckConstraintsRejectInvalidValues(t *testing.T) {
	db := testDB(t)

	cases := []struct {
		name       string
		sql        string
		args       []any
		constraint string
	}{
		{
			name:       "effect 非 allow/deny",
			sql:        "INSERT INTO sys_role_permission (role_id, permission_id, effect) VALUES (1, 1, ?)",
			args:       []any{"maybe"},
			constraint: "ck_role_permission_effect",
		},
		{
			name:       "resource_type 越界",
			sql:        "INSERT INTO sys_permission (code, name, resource_type, action) VALUES (?, 'x', 9, 'read')",
			args:       []any{uniqueName("ck")},
			constraint: "ck_permission_resource_type",
		},
		{
			name:       "action 非 CRUD",
			sql:        "INSERT INTO sys_permission (code, name, resource_type, action) VALUES (?, 'x', 1, 'frobnicate')",
			args:       []any{uniqueName("ck")},
			constraint: "ck_permission_action",
		},
		{
			name:       "last_status 非法状态",
			sql:        `INSERT INTO sys_cron_job (name, "interval", last_status) VALUES (?, '* * * * * *', 'bogus')`,
			args:       []any{uniqueName("ck")},
			constraint: "ck_cron_job_last_status",
		},
		{
			name:       "triggered_by 非 scheduler/manual",
			sql:        "INSERT INTO sys_cron_job_execution (job_name, status, started_at, error, triggered_by) VALUES (?, 'success', NOW(), '', 'cron')",
			args:       []any{uniqueName("ck")},
			constraint: "ck_execution_triggered_by",
		},
	}

	raw := rawDB(t)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// 走应用同款配置：断言 GORM 翻译后的哨兵错误
			err := db.Exec(tc.sql, tc.args...).Error
			require.Error(t, err, "非法值应被 CHECK 拒绝")
			assert.ErrorIs(t, err, gorm.ErrCheckConstraintViolated)

			// 走未翻译的连接：断言触发的是预期的那条约束
			rawErr := raw.Exec(tc.sql, tc.args...).Error
			require.Error(t, rawErr)
			var pgErr *pgconn.PgError
			require.ErrorAs(t, rawErr, &pgErr)
			assert.Equal(t, "23514", pgErr.Code)
			assert.Equal(t, tc.constraint, pgErr.ConstraintName)
		})
	}
}
