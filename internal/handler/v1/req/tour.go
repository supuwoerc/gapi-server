package req

type UpdateToursRequest struct {
	CompletedTours []string `json:"completed_tours" binding:"required"`
}
