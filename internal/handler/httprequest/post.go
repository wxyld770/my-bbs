package httprequest

import "my-bbs/internal/model"

type CreatePostRequest struct {
	Title   string `json:"title" binding:"required,max=255"`
	Content string `json:"content" binding:"required"`
}

type UpdatePostRequest struct {
	// Pointer fields distinguish an omitted field from an explicitly empty one.
	Title   *string `json:"title" binding:"omitempty,max=255"`
	Content *string `json:"content"`
}

type SetVisibleRequest struct {
	Visible *uint8 `json:"visible" binding:"required,oneof=0 1"`
}

type PinPostRequest struct {
	Duration model.PostPinDuration `json:"duration" binding:"required,oneof=day week month permanent"`
}
