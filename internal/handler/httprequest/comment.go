package httprequest

type CreateCommentRequest struct {
	Content string `json:"content" binding:"required"`
}
