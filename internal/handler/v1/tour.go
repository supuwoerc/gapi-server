package v1

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/req"
	"github.com/supuwoerc/gapi-server/internal/handler/v1/resp"
	"github.com/supuwoerc/gapi-server/internal/middleware"
	"github.com/supuwoerc/gapi-server/pkg/response"
)

type TourServiceInterface interface {
	UpdateCompletedTours(ctx context.Context, userID uint64, tours []string) ([]string, error)
}

type TourHandler struct {
	Service TourServiceInterface
	JWTAuth gin.HandlerFunc
}

func (h *TourHandler) Register(r *gin.RouterGroup) {
	tour := r.Group("/tour")
	tour.Use(h.JWTAuth)
	{
		tour.PATCH("", h.Update)
	}
}

func (h *TourHandler) Update(c *gin.Context) {
	userID, ok := middleware.CurrentUserID(c)
	if !ok {
		response.FailWithCode(c, response.InvalidToken)
		return
	}
	var r req.UpdateToursRequest
	if err := c.ShouldBindJSON(&r); err != nil {
		response.ParamsValidateFail(c, err)
		return
	}
	tours, err := h.Service.UpdateCompletedTours(c.Request.Context(), userID, r.CompletedTours)
	if err != nil {
		response.FailWithError(c, err)
		return
	}
	response.SuccessWithData(c, resp.UpdateToursResponse{
		CompletedTours: tours,
	})
}
