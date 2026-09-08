package response

import (
	"encoding/json"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// localeDirs 是 middleware.I18n 实际加载的两个目录（那边用的是相对仓库根的
// ./pkg/locale/{zh,en}，这里从 pkg/response 出发所以是 ../locale/...）。
var localeDirs = map[string]string{
	"zh": "../locale/zh",
	"en": "../locale/en",
}

// statusCodeLineComments 解析 code.go，取出每个 StatusCode 常量的行尾注释。
// 行尾注释就是 stringer -linecomment 生成的 String() 返回值，也就是 i18n 的 MessageID。
// 直接解析源码而不是手工维护清单，新增响应码不需要改本测试。
func statusCodeLineComments(t *testing.T) map[string]string {
	t.Helper()

	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "code.go", nil, parser.ParseComments)
	require.NoError(t, err, "解析 code.go 失败")

	result := make(map[string]string)
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST {
			continue
		}
		for _, spec := range gen.Specs {
			vs, ok := spec.(*ast.ValueSpec)
			if !ok || len(vs.Names) == 0 {
				continue
			}
			// 只认显式写了 StatusCode 类型的常量
			ident, ok := vs.Type.(*ast.Ident)
			if !ok || ident.Name != "StatusCode" {
				continue
			}
			name := vs.Names[0].Name
			if vs.Comment == nil || len(vs.Comment.List) == 0 {
				t.Errorf("响应码 %s 缺行尾注释：stringer 会把常量名当文案", name)
				continue
			}
			text := strings.TrimSpace(strings.TrimPrefix(vs.Comment.List[0].Text, "//"))
			require.NotEmpty(t, text, "响应码 %s 的行尾注释是空的", name)
			result[name] = text
		}
	}
	require.NotEmpty(t, result, "没从 code.go 里解析出任何 StatusCode 常量")
	return result
}

// localeMessageIDs 读取一个语言目录下全部 json 的 MessageID 集合，
// 加载方式与 middleware.loadMessages 一致（遍历目录、每个文件是 []*i18n.Message）。
func localeMessageIDs(t *testing.T, dir string) map[string]string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	require.NoErrorf(t, err, "读词条目录 %s 失败", dir)

	ids := make(map[string]string)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		path := filepath.Join(dir, e.Name())
		data, rerr := os.ReadFile(path)
		require.NoErrorf(t, rerr, "读 %s 失败", path)

		var msgs []struct {
			ID    string `json:"id"`
			Other string `json:"other"`
		}
		require.NoErrorf(t, json.Unmarshal(data, &msgs), "解析 %s 失败", path)

		for _, m := range msgs {
			require.NotEmptyf(t, m.ID, "%s 里有词条缺 id", path)
			_, dup := ids[m.ID]
			require.Falsef(t, dup, "%s 里 id %q 重复", path, m.ID)
			ids[m.ID] = m.Other
		}
	}
	return ids
}

// 每个响应码都必须在 zh 与 en 两边都有词条。
//
// HttpResponse 用的是 localizer.MustLocalize，缺词条时**直接 panic**，
// 而不是退化成空串——panic 虽被 Recovery 中间件兜住，但调用方拿到的会是
// recoveryError(10005) 而非本该返回的那个码，业务语义就丢了。
//
// 曾经 activationCodeInvalid 与 activationCodeSendTooFrequent 两个码漏补词条,
// POST /api/v1/auth/verify-email 传错验证码即可触发。本测试就是为拦住这类遗漏。
func TestEveryStatusCodeHasLocaleEntry(t *testing.T) {
	codes := statusCodeLineComments(t)

	for lang, dir := range localeDirs {
		ids := localeMessageIDs(t, dir)
		for name, msgID := range codes {
			text, ok := ids[msgID]
			require.Truef(t, ok,
				"响应码 %s（MessageID %q）缺 %s 词条：命中该码时 MustLocalize 会 panic，"+
					"请在 %s/system.json 补一条", name, msgID, lang, dir)
			require.NotEmptyf(t, strings.TrimSpace(text),
				"响应码 %s（MessageID %q）的 %s 词条是空文案", name, msgID, lang)
		}
	}
}

// zh 与 en 的词条集合要一致：只补一边会导致换语言时 panic。
func TestLocalesHaveSameMessageIDs(t *testing.T) {
	zh := localeMessageIDs(t, localeDirs["zh"])
	en := localeMessageIDs(t, localeDirs["en"])

	for id := range zh {
		_, ok := en[id]
		require.Truef(t, ok, "词条 %q 只有 zh 没有 en", id)
	}
	for id := range en {
		_, ok := zh[id]
		require.Truef(t, ok, "词条 %q 只有 en 没有 zh", id)
	}
}

// 生成物是否最新：String() 必须返回行尾注释，而非退化成 "StatusCode(20013)"。
// 漏跑 go generate 时本测试失败。
func TestGeneratedStringMatchesLineComment(t *testing.T) {
	codes := statusCodeLineComments(t)

	// 反查：把常量名映射到值需要运行期的常量，这里用已知的几个做抽样断言，
	// 其余靠"不含类型名前缀"这条通用规则覆盖全部。
	for _, c := range allStatusCodes() {
		got := c.String()
		require.Falsef(t, strings.HasPrefix(got, "StatusCode("),
			"响应码 %d 的 String() 退化成 %q，漏跑 go generate ./...", int(c), got)
		require.Containsf(t, valuesOf(codes), got,
			"响应码 %d 的 String() 是 %q，与 code.go 的行尾注释都对不上", int(c), got)
	}
}

func valuesOf(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// allStatusCodes 列出全部响应码。新增码时补进来——漏补不会导致误报，
// 只是少一层抽样覆盖（TestEveryStatusCodeHasLocaleEntry 仍会解析源码全量检查）。
func allStatusCodes() []StatusCode {
	return []StatusCode{
		Ok, Error, InvalidParams, InvalidToken, CancelRequest, RecoveryError,
		InternalError, TimeoutErr, Busy,
		UserAlreadyExists, InvalidCredential, UserDisabled, UserLocked,
		TokenExpired, RefreshTokenUsed,
		CaptchaGenFailed, CaptchaInvalid, CaptchaExpired, CaptchaTokenInvalid,
		TourIDInvalid, TourLimitExceeded,
		ActivationCodeInvalid, ActivationCodeSendTooFrequent,
	}
}
