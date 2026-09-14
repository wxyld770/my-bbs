package httpresponse

import "my-bbs/internal/service"

type LikeResponse struct {
	Liked     bool  `json:"liked"`
	LikeCount int64 `json:"like_count"`
}

func NewLikeResponse(result *service.LikeResult) LikeResponse {
	if result == nil {
		return LikeResponse{}
	}
	return LikeResponse{
		Liked:     result.Liked,
		LikeCount: result.LikeCount,
	}
}
