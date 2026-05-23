package middleware

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"gapi-server/internal/config"
	"gapi-server/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	enTranslations "github.com/go-playground/validator/v10/translations/en"
	zhTranslations "github.com/go-playground/validator/v10/translations/zh"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

const (
	langCN = "cn"
	langEN = "en"

	defaultZhPath = "./pkg/locale/zh"
	defaultEnPath = "./pkg/locale/en"
)

func I18n(cfg *config.LocaleConfig) gin.HandlerFunc {
	bundle := i18n.NewBundle(language.Chinese)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	loadMessages(bundle, defaultZhPath, language.Chinese)
	loadMessages(bundle, defaultEnPath, language.English)

	cnLocalizer := i18n.NewLocalizer(bundle, language.Chinese.String())
	enLocalizer := i18n.NewLocalizer(bundle, language.English.String())

	localeKey := cfg.LocaleKey
	defaultLang := cfg.DefaultLang

	return func(c *gin.Context) {
		lang := c.GetHeader(localeKey)
		if lang != langCN && lang != langEN {
			lang = defaultLang
		}
		if lang == langEN {
			c.Set(response.I18nLocalizerKey, enLocalizer)
		} else {
			c.Set(response.I18nLocalizerKey, cnLocalizer)
		}
		c.Next()
	}
}

func Validator(cfg *config.LocaleConfig) gin.HandlerFunc {
	zhTrans := zh.New()
	enTrans := en.New()
	uni := ut.New(enTrans, enTrans, zhTrans)

	zhTranslator, _ := uni.GetTranslator("zh")
	enTranslator, _ := uni.GetTranslator("en")

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" {
				return ""
			}
			if name != "" {
				return name
			}
			return strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
		})
		if err := zhTranslations.RegisterDefaultTranslations(v, zhTranslator); err != nil {
			panic(err)
		}
		if err := enTranslations.RegisterDefaultTranslations(v, enTranslator); err != nil {
			panic(err)
		}
	}

	localeKey := cfg.LocaleKey
	defaultLang := cfg.DefaultLang

	return func(c *gin.Context) {
		lang := c.GetHeader(localeKey)
		if lang != langCN && lang != langEN {
			lang = defaultLang
		}
		if lang == langEN {
			c.Set(response.ValidatorTranslatorKey, enTranslator)
		} else {
			c.Set(response.ValidatorTranslatorKey, zhTranslator)
		}
		c.Next()
	}
}

func loadMessages(bundle *i18n.Bundle, dir string, tag language.Tag) {
	_ = filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(info.Name(), ".json") {
			return err
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			panic("failed to read locale file: " + path)
		}
		var msgs []*i18n.Message
		if e := json.Unmarshal(data, &msgs); e != nil {
			panic("failed to parse locale file: " + path)
		}
		if e := bundle.AddMessages(tag, msgs...); e != nil {
			panic("failed to add messages: " + path)
		}
		return nil
	})
}
