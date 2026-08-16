package httpresponse

import "my-bbs/internal/service"

type LikeToggleResponse struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"like_count"`
}

func NewLikeToggleResponse(result *service.LikeToggleResult) LikeToggleResponse {
	if result == nil {
		return LikeToggleResponse{}
	}
	return LikeToggleResponse{
		Liked:     result.Liked,
		LikeCount: result.LikeCount,
	}
}
