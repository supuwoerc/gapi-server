package response

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BasicResponse[T any] struct {
	Code    int    `json:"code"`
	Data    T      `json:"data"`
	Message string `json:"message"`
}

func Success(ctx *gin.Context) {
	ctx.JSON(http.StatusOK, BasicResponse[any]{Code: int(Ok)})
}

func SuccessWithData[T any](ctx *gin.Context, data T) {
	ctx.JSON(http.StatusOK, BasicResponse[T]{Code: int(Ok), Data: data})
}

func SuccessWithMessage(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, BasicResponse[any]{Code: int(Ok), Message: message})
}

func FailWithMessage(ctx *gin.Context, message string) {
	ctx.JSON(http.StatusOK, BasicResponse[any]{Code: int(Error), Message: message})
}

func FailWithCode(ctx *gin.Context, code StatusCode) {
	ctx.JSON(http.StatusOK, BasicResponse[any]{Code: int(code)})
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
