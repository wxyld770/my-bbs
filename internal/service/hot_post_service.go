package service

import (
	"context"
	"sort"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
)

const (
	// TodayHotPostLimit 是今日热榜固定返回的最大帖子数。
	TodayHotPostLimit = 10

	hotCommentWeightMillis int64 = 600
	hotLikeWeightMillis    int64 = 400
)

// Clock 为涉及自然日边界的用例提供可替换时间源。
type Clock func() time.Time

// HotPostSummary 是热榜应用层输出。ScoreMillis 表示 score 的千分位整数，
// 从计算到排序都不经过浮点数。
type HotPostSummary struct {
	Post         model.Post
	LikeCount    int64
	CommentCount int64
	ScoreMillis  int64
}

type HotPostService struct {
	hotPostRepo repository.HotPostReader
	userRepo    repository.UserReader
	now         Clock
}

func NewHotPostService(
	hotPostRepo repository.HotPostReader,
	userRepo repository.UserReader,
	clocks ...Clock,
) *HotPostService {
	now := Clock(time.Now)
	if len(clocks) > 0 && clocks[0] != nil {
		now = clocks[0]
	}
	return &HotPostService{hotPostRepo: hotPostRepo, userRepo: userRepo, now: now}
}

// GetTodayHotPosts 返回昨日和今日发布的公开帖子热榜。
// 时间窗口按应用本地时区计算：[昨日 00:00，明日 00:00)。
func (s *HotPostService) GetTodayHotPosts(ctx context.Context) ([]HotPostSummary, error) {
	now := s.now().In(time.Local)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	publishedFrom := today.AddDate(0, 0, -1)
	publishedBefore := today.AddDate(0, 0, 1)

	engagements, err := s.hotPostRepo.FindPublicPostEngagements(ctx, publishedFrom, publishedBefore)
	if err != nil {
		return nil, err
	}

	ranked := make([]HotPostSummary, len(engagements))
	for i := range engagements {
		ranked[i] = HotPostSummary{
			Post:         engagements[i].Post,
			LikeCount:    engagements[i].LikeCount,
			CommentCount: engagements[i].CommentCount,
			ScoreMillis: hotScoreMillis(
				engagements[i].CommentCount,
				engagements[i].LikeCount,
			),
		}
	}

	sort.Slice(ranked, func(i, j int) bool {
		left, right := ranked[i], ranked[j]
		if left.ScoreMillis != right.ScoreMillis {
			return left.ScoreMillis > right.ScoreMillis
		}
		if left.CommentCount != right.CommentCount {
			return left.CommentCount > right.CommentCount
		}
		if !left.Post.CreateTime.Equal(right.Post.CreateTime) {
			return left.Post.CreateTime.After(right.Post.CreateTime)
		}
		return left.Post.ID > right.Post.ID
	})

	if len(ranked) > TodayHotPostLimit {
		ranked = ranked[:TodayHotPostLimit]
	}
	if ranked == nil {
		return []HotPostSummary{}, nil
	}

	postPtrs := make([]*model.Post, len(ranked))
	for i := range ranked {
		postPtrs[i] = &ranked[i].Post
	}
	if err := fillPostUsers(ctx, s.userRepo, postPtrs); err != nil {
		return nil, err
	}
	return ranked, nil
}

func hotScoreMillis(commentCount, likeCount int64) int64 {
	// count 为整数，权重用千分位整数表达，天然等价于计算后向零截断到
	// 三位小数：comments*0.600 + likes*0.400。
	return commentCount*hotCommentWeightMillis + likeCount*hotLikeWeightMillis
}
