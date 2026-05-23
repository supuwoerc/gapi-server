package response

import (
	"context"
	"errors"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	"github.com/nicksnyder/go-i18n/v2/i18n"
)

const (
	I18nLocalizerKey       = "i18n_localizer"
	ValidatorTranslatorKey = "validator_translator"
)

var indexRegexp = regexp.MustCompile(`\[\d+]`)

type BasicResponse[T any] struct {
	Code    int    `json:"code"`
	Data    T      `json:"data"`
	Message string `json:"message"`
}

func HttpResponse[T any](ctx *gin.Context, code StatusCode, data T, message *string) {
	var msg string
	if message != nil {
		msg = *message
	} else if localizer, ok := ctx.Value(I18nLocalizerKey).(*i18n.Localizer); ok && localizer != nil {
		msg = localizer.MustLocalize(&i18n.LocalizeConfig{MessageID: code.String()})
	}
	ctx.AbortWithStatusJSON(http.StatusOK, BasicResponse[T]{
		Code:    int(code),
		Data:    data,
		Message: msg,
	})
}

func Success(ctx *gin.Context) {
	HttpResponse[any](ctx, Ok, nil, nil)
}

func SuccessWithData[T any](ctx *gin.Context, data T) {
	HttpResponse[T](ctx, Ok, data, nil)
}

func SuccessWithMessage(ctx *gin.Context, message string) {
	HttpResponse[any](ctx, Ok, nil, &message)
}

func FailWithMessage(ctx *gin.Context, message string) {
	HttpResponse[any](ctx, Error, nil, &message)
}

func FailWithCode(ctx *gin.Context, code StatusCode) {
	HttpResponse[any](ctx, code, nil, nil)
}

func FailWithError(ctx *gin.Context, err error) {
	if code, ok := errors.AsType[StatusCode](err); ok {
		FailWithCode(ctx, code)
		return
	}
	switch {
	case errors.Is(err, context.Canceled):
		FailWithCode(ctx, CancelRequest)
	case errors.Is(err, context.DeadlineExceeded):
		FailWithCode(ctx, TimeoutErr)
	default:
		FailWithMessage(ctx, err.Error())
	}
}

func ParamsValidateFail(ctx *gin.Context, err error) {
	var errs validator.ValidationErrors
	if !errors.As(err, &errs) {
		msg := err.Error()
		HttpResponse[any](ctx, InvalidParams, nil, &msg)
		return
	}
	translator, exists := ctx.Get(ValidatorTranslatorKey)
	if !exists {
		msg := err.Error()
		HttpResponse[any](ctx, InvalidParams, nil, &msg)
		return
	}
	trans, ok := translator.(ut.Translator)
	if !ok {
		msg := err.Error()
		HttpResponse[any](ctx, InvalidParams, nil, &msg)
		return
	}
	errMap := make(map[string]string, len(errs))
	for _, e := range errs {
		field := indexRegexp.ReplaceAllString(e.Field(), "")
		if strings.TrimSpace(field) == "" {
			field = e.StructField()
		}
		errMap[field] = e.Translate(trans)
	}
	HttpResponse[any](ctx, InvalidParams, errMap, nil)
}
