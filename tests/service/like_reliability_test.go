package service_test

import (
	"context"
	"errors"
	"testing"

	postcache "my-bbs/internal/cache"
	"my-bbs/internal/model"
	"my-bbs/internal/service"
	"my-bbs/tests/testutil"
)

func TestLikeService_InvalidatesCachedCountAfterMutationWhenCountingFails(t *testing.T) {
	const (
		postID = 9
		userID = 7
	)
	countErr := errors.New("count unavailable")

	tests := []struct {
		name string
		call func(*service.LikeService) error
	}{
		{
			name: "like",
			call: func(svc *service.LikeService) error {
				_, err := svc.Like(context.Background(), postID, userID)
				return err
			},
		},
		{
			name: "unlike",
			call: func(svc *service.LikeService) error {
				_, err := svc.Unlike(context.Background(), postID, userID)
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			countCache := postcache.NewPostCountCache(testutil.NewTestRedis(t))
			countCache.SetLikeCounts(ctx, map[uint]int64{postID: 99})
			svc := service.NewLikeServiceWithCountCache(
				&countFailingLikeRepository{err: countErr},
				&publicPostReader{post: &model.Post{
					BaseModel: model.BaseModel{ID: postID},
					Visible:   model.VisiblePublic,
				}},
				&activeUserReader{},
				countCache,
			)

			if err := tt.call(svc); !errors.Is(err, countErr) {
				t.Fatalf("service error=%v, want count error", err)
			}
			if got := countCache.GetLikeCounts(ctx, []uint{postID}); len(got) != 0 {
				t.Fatalf("like count cache after committed mutation=%v, want invalidated", got)
			}
		})
	}
}

type countFailingLikeRepository struct {
	err error
}

func (r *countFailingLikeRepository) CountByPostID(context.Context, uint) (int64, error) {
	return 0, r.err
}

func (*countFailingLikeRepository) CountByPostIDs(context.Context, []uint) (map[uint]int64, error) {
	return map[uint]int64{}, nil
}

func (*countFailingLikeRepository) ExistsByUserAndPost(context.Context, uint, uint) (bool, error) {
	return false, nil
}

func (*countFailingLikeRepository) Create(context.Context, *model.PostLike) error {
	return nil
}

func (*countFailingLikeRepository) DeleteByUserAndPost(context.Context, uint, uint) error {
	return nil
}
