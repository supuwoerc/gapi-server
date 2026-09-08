package model

import (
	"database/sql/driver"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

// 落库枚举的共用样板。本包所有枚举都靠这里的函数实现 Scan / Value / Text，
// 新增枚举时不用再抄一遍 switch（每个类型三行转发即可）。
//
// 分成两族是因为本包的枚举落库形式不统一，且都由 DDL 钉死：
//   - 字符串族（VARCHAR + CHECK IN (...)）：TriggeredBy、PermissionAction、
//     PermissionEffect。用 enumStringXxx。
//   - 整型族（SMALLINT + CHECK BETWEEN）：ResourceType。用 enumIntXxx。
//
// ⚠️ 两族的常量值都**直接落库**，改动等于改写存量数据的含义：字符串族不能改
// 拼写（要同步 migrations 里的 CHECK 与 seed SQL），整型族只能末尾追加、绝不
// 能重排或复用已废弃的值（重排不报错，但库里的 2 会被静默解释成别的东西，
// 且要同步 CHECK 的上界）。改之前先确认存量数据。

type enumString interface{ ~string }

type enumInt interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// enumStringIntoDB 字符串枚举的落库形式：string。
func enumStringIntoDB[T enumString](v T) (driver.Value, error) {
	return string(v), nil
}

// enumStringFromDB 从数据库值还原字符串枚举。
//
// pgx 读 VARCHAR 通常给 string，但预处理与不同驱动版本下也可能是 []byte，
// 两种都认，行为不随 GORM / 驱动版本变化。
//
// 不校验值是否为已定义的常量——helper 拿不到各枚举的合法集合，这层由列上的
// CHECK IN (...) 兜着。认不出的类型返回错误而非静默置零：这些列都是 NOT NULL，
// 静默置零会让一行坏数据看起来像合法的空值。
func enumStringFromDB[T enumString](dst *T, src any) error {
	switch v := src.(type) {
	case string:
		*dst = T(v)
	case []byte:
		*dst = T(v)
	default:
		return fmt.Errorf("cannot scan %T into %s", src, enumTypeName(*dst))
	}
	return nil
}

// enumStringText 对外文案。字符串枚举的常量值本身就是文案（"scheduler"、
// "create"），直接返回即可；真出现库里的脏值也原样透出，不伪装成 "unknown"。
func enumStringText[T enumString](v T) string {
	return string(v)
}

// enumIntIntoDB 整型枚举的落库形式：整型。
// driver 支持的整型只有 int64，这里统一收口。
func enumIntIntoDB[T enumInt](v T) (driver.Value, error) {
	return int64(v), nil
}

// enumIntFromDB 从数据库值还原整型枚举。
//
// 不依赖 GORM 对具名整型的反射转换：PostgreSQL 的 SMALLINT 经 pgx 以 int16
// 递送，但换驱动 / 走预处理时也可能是 int64、int32、int 或文本形式，这几种
// 都认，行为不随 GORM / 驱动版本变化。
//
// 认不出的类型返回错误而非静默置零——零值在本包的整型枚举里不是合法值
// （都从 1 起算），静默置零会让坏数据看起来像合法的"未设置"。
func enumIntFromDB[T enumInt](dst *T, src any) error {
	switch v := src.(type) {
	case int64:
		*dst = T(v)
	case int32:
		*dst = T(v)
	case int16:
		*dst = T(v)
	case int8:
		*dst = T(v)
	case int:
		*dst = T(v)
	case string:
		n, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("cannot scan %q into %s: %w", v, enumTypeName(*dst), err)
		}
		*dst = T(n)
	case []byte:
		n, err := strconv.ParseInt(string(v), 10, 64)
		if err != nil {
			return fmt.Errorf("cannot scan %q into %s: %w", v, enumTypeName(*dst), err)
		}
		*dst = T(n)
	default:
		return fmt.Errorf("cannot scan %T into %s", src, enumTypeName(*dst))
	}
	return nil
}

// enumIntText 对外文案，区间外的值兜底成 "unknown"。
//
// stringer 对未定义的值返回 "ResourceType(9)" 这类内部表示，而这些文案会经接口
// 吐给调用方（暴露 Go 类型名），故统一在这里兜底。
//
// **对外转换一律用 Text()，不要用 String()**。String() 保留内部表示，
// 内部日志里反而有助于排查。
func enumIntText[T enumInt](v T) string {
	s, ok := any(v).(fmt.Stringer)
	if !ok {
		// 没生成 String()：漏跑 go generate 了
		return "unknown"
	}
	name := s.String()
	if strings.HasPrefix(name, enumTypeName(v)+"(") {
		return "unknown"
	}
	return name
}

// enumTypeName 取具体类型名（如 "ResourceType"），用于拼 stringer 的兜底前缀
// 与错误信息。用反射而非让调用方手传字符串：手传的话传错了兜底会静默失效。
func enumTypeName[T any](v T) string {
	return reflect.TypeOf(v).Name()
}
