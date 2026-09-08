package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 字符串族枚举: 常量值直接落库(VARCHAR + CHECK IN (...)), 改拼写会让存量行不再匹配。
// 这里把值钉死, 同时校验 migrations 的 CHECK 集合与常量集合一一对应。
func TestStringEnumValuesArePinned(t *testing.T) {
	t.Run("TriggeredBy", func(t *testing.T) {
		want := map[TriggeredBy]string{
			TriggeredByScheduler: "scheduler",
			TriggeredByManual:    "manual",
		}
		for e, s := range want {
			assert.Equal(t, s, string(e), "常量值变了: 库里的 %q 不再匹配", s)
			assert.Equal(t, s, e.Text())
		}
		// ck_execution_triggered_by 里只有这两个值
		assert.Len(t, want, 2, "新增触发方式需同步 migrations 的 CHECK IN")
	})

	t.Run("PermissionAction", func(t *testing.T) {
		want := map[PermissionAction]string{
			PermissionActionCreate: "create",
			PermissionActionRead:   "read",
			PermissionActionUpdate: "update",
			PermissionActionDelete: "delete",
		}
		for e, s := range want {
			assert.Equal(t, s, string(e), "常量值变了: 库里的 %q 不再匹配", s)
			assert.Equal(t, s, e.Text())
		}
		assert.Len(t, want, 4, "新增操作需同步 migrations 的 CHECK IN")
	})

	t.Run("PermissionEffect", func(t *testing.T) {
		want := map[PermissionEffect]string{
			PermissionEffectAllow: "allow",
			PermissionEffectDeny:  "deny",
		}
		for e, s := range want {
			assert.Equal(t, s, string(e), "常量值变了: 库里的 %q 不再匹配", s)
			assert.Equal(t, s, e.Text())
		}
		assert.Len(t, want, 2, "新增效果需同步 migrations 的 CHECK IN")
	})
}

// 整型族: 常量值直接落库(SMALLINT + CHECK BETWEEN 1 AND 5), 只能末尾追加。
// 重排不会有编译错误, 但存量行的含义会被静默改写, 所以钉死值与文案。
func TestResourceTypeValuesArePinned(t *testing.T) {
	want := map[ResourceType]struct {
		value int
		text  string
	}{
		ResourceTypeAPI:            {1, "api"},
		ResourceTypeFrontendMenu:   {2, "frontend-menu"},
		ResourceTypeFrontendRoute:  {3, "frontend-route"},
		ResourceTypeFrontendButton: {4, "frontend-button"},
		ResourceTypeData:           {5, "data"},
	}

	for rt, exp := range want {
		assert.Equal(t, exp.value, int(rt),
			"常量值变了: 库里存量行的 %d 会被重新解释成别的类型", exp.value)
		assert.Equal(t, exp.text, rt.String(),
			"文案变了: 漏跑 go generate, 或改了行尾注释")
		assert.Equal(t, exp.text, rt.Text())
	}

	// ck_permission_resource_type 的上界是 5
	assert.Len(t, want, 5, "新增类型需同步放宽 migrations 的 CHECK (BETWEEN 1 AND N)")
}

// 驱动递送同一个值可能有多种 Go 类型(pgx 的 SMALLINT 走 int16, 预处理/换驱动
// 可能给 int64 或文本), Scan 都要认, 且结果一致。
func TestScanAcceptsDriverRepresentations(t *testing.T) {
	t.Run("int enum", func(t *testing.T) {
		srcs := []any{
			int64(2), int32(2), int16(2), int8(2), int(2), "2", []byte("2"),
		}
		for _, src := range srcs {
			var got ResourceType
			require.NoErrorf(t, got.Scan(src), "scan %T", src)
			assert.Equalf(t, ResourceTypeFrontendMenu, got, "scan %T(%v)", src, src)
		}
	})

	t.Run("string enum", func(t *testing.T) {
		srcs := []any{"manual", []byte("manual")}
		for _, src := range srcs {
			var got TriggeredBy
			require.NoErrorf(t, got.Scan(src), "scan %T", src)
			assert.Equalf(t, TriggeredByManual, got, "scan %T(%v)", src, src)
		}
	})
}

// 认不出的类型必须报错, 不能静默置零: 这些列都是 NOT NULL, 且整型枚举从 1 起算,
// 零值不是合法值——静默置零会让坏数据看起来像合法的"未设置"。
func TestScanRejectsUnknownTypes(t *testing.T) {
	t.Run("int enum", func(t *testing.T) {
		var rt ResourceType
		require.Error(t, rt.Scan(nil))
		require.Error(t, rt.Scan(1.5))
		require.Error(t, rt.Scan("not-a-number"))
		err := rt.Scan(struct{}{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ResourceType", "错误信息应带上具体类型名")
	})

	t.Run("string enum", func(t *testing.T) {
		var by TriggeredBy
		require.Error(t, by.Scan(nil))
		err := by.Scan(123)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "TriggeredBy", "错误信息应带上具体类型名")
	})
}

// Value 的落库形式要和列类型对得上: 整型族给 int64, 字符串族给 string。
func TestValueUsesColumnRepresentation(t *testing.T) {
	v, err := ResourceTypeData.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(5), v, "SMALLINT 列应落 int64")

	v, err = TriggeredByManual.Value()
	require.NoError(t, err)
	assert.Equal(t, "manual", v, "VARCHAR 列应落 string")

	v, err = PermissionEffectDeny.Value()
	require.NoError(t, err)
	assert.Equal(t, "deny", v)
}

// Scan -> Value 往返后值不变(经驱动读回再写回去不应漂移)。
func TestRoundTrip(t *testing.T) {
	var rt ResourceType
	require.NoError(t, rt.Scan(int16(3)))
	v, err := rt.Value()
	require.NoError(t, err)
	assert.Equal(t, int64(3), v)

	var by TriggeredBy
	require.NoError(t, by.Scan([]byte("scheduler")))
	v, err = by.Value()
	require.NoError(t, err)
	assert.Equal(t, "scheduler", v)
}

// 整型族的 Text() 要兜底: stringer 对未定义值返回 "ResourceType(9)" 这类内部
// 表示, 会把 Go 类型名暴露给接口调用方。
func TestIntEnumTextFallsBackToUnknown(t *testing.T) {
	assert.Equal(t, "unknown", ResourceType(9).Text())
	assert.Equal(t, "unknown", ResourceType(0).Text())
	assert.Equal(t, "unknown", ResourceType(-1).Text())

	// String() 保留内部表示, 便于内部日志排查
	assert.Contains(t, ResourceType(9).String(), "ResourceType")
}

// 字符串族的 Text() 不做 unknown 伪装: 脏值原样透出便于发现问题。
func TestStringEnumTextPassesThrough(t *testing.T) {
	assert.Equal(t, "weird", TriggeredBy("weird").Text())
	assert.Empty(t, TriggeredBy("").Text())
}
