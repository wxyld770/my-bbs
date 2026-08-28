package repository_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"
)

func TestUserRepository_UpdateAvatarUsesAtomicCooldown(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	ctx := context.Background()
	user := &model.User{Username: "avatar-repository", Password: "hash", Status: model.UserStatusNormal}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	firstAt := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	firstURL := "https://cdn.example.com/Avatar.png"
	if err := repo.UpdateAvatar(ctx, user.ID, firstURL, firstAt, firstAt.Add(-24*time.Hour)); err != nil {
		t.Fatalf("first update: %v", err)
	}

	tooSoonAt := firstAt.Add(24*time.Hour - time.Millisecond)
	err := repo.UpdateAvatar(ctx, user.ID, "https://cdn.example.com/too-soon.png", tooSoonAt, tooSoonAt.Add(-24*time.Hour))
	if !errors.Is(err, repository.ErrAvatarUpdateTooFrequent) {
		t.Fatalf("too-soon update error=%v, want cooldown", err)
	}

	// 即使已过冷却，相同 URL 也只是幂等成功，不能刷新 avatar_updated_at。
	if err := repo.UpdateAvatar(ctx, user.ID, firstURL, firstAt.Add(25*time.Hour), firstAt.Add(time.Hour)); err != nil {
		t.Fatalf("same URL after cooldown: %v", err)
	}
	stored, err := repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarUpdatedAt == nil || !stored.AvatarUpdatedAt.Equal(firstAt) {
		t.Fatalf("same URL refreshed cooldown: user=%+v err=%v", stored, err)
	}

	exactlyAllowedAt := firstAt.Add(24 * time.Hour)
	// URL 路径区分大小写，数据库不能把这次变更当成相同链接。
	secondURL := "https://cdn.example.com/avatar.png"
	if err := repo.UpdateAvatar(ctx, user.ID, secondURL, exactlyAllowedAt, exactlyAllowedAt.Add(-24*time.Hour)); err != nil {
		t.Fatalf("exact 24-hour update: %v", err)
	}
	stored, err = repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarURL != secondURL || stored.AvatarUpdatedAt == nil || !stored.AvatarUpdatedAt.Equal(exactlyAllowedAt) {
		t.Fatalf("unexpected second avatar state: user=%+v err=%v", stored, err)
	}

	clearAt := exactlyAllowedAt.Add(24 * time.Hour)
	if err := repo.UpdateAvatar(ctx, user.ID, "", clearAt, clearAt.Add(-24*time.Hour)); err != nil {
		t.Fatalf("clear avatar: %v", err)
	}
	stored, err = repo.FindUserByID(ctx, user.ID)
	if err != nil || stored == nil || stored.AvatarURL != "" || stored.AvatarUpdatedAt == nil || !stored.AvatarUpdatedAt.Equal(clearAt) {
		t.Fatalf("unexpected cleared avatar state: user=%+v err=%v", stored, err)
	}
}

func TestUserRepository_UpdateAvatarRejectsSoftDeletedUser(t *testing.T) {
	db := testutil.NewTestDB(t)
	repo := gormrepo.NewUserRepository(db)
	ctx := context.Background()
	user := &model.User{Username: "deleted-avatar", Password: "hash", Status: model.UserStatusNormal}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Delete(user).Error; err != nil {
		t.Fatalf("delete user: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Millisecond)
	err := repo.UpdateAvatar(ctx, user.ID, "https://cdn.example.com/avatar.png", now, now.Add(-24*time.Hour))
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("soft-deleted update error=%v, want not found", err)
	}
}

func TestUserRepository_UpdateAvatarAllowsOnlyOneConcurrentChange(t *testing.T) {
	db := testutil.NewTestDB(t)
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql db: %v", err)
	}
	// SQLite 测试库使用一条连接避免把数据库级写锁错误误当作限额结果。
	sqlDB.SetMaxOpenConns(1)
	repo := gormrepo.NewUserRepository(db)
	ctx := context.Background()
	user := &model.User{Username: "avatar-concurrent", Password: "hash", Status: model.UserStatusNormal}
	if err := repo.CreateUser(ctx, user); err != nil {
		t.Fatalf("create user: %v", err)
	}

	now := time.Date(2026, 8, 28, 8, 0, 0, 0, time.UTC)
	const requests = 16
	var successes atomic.Int32
	errorsCh := make(chan error, requests)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			err := repo.UpdateAvatar(
				ctx,
				user.ID,
				fmt.Sprintf("https://cdn.example.com/avatar-%d.png", index),
				now,
				now.Add(-24*time.Hour),
			)
			switch {
			case err == nil:
				successes.Add(1)
			case errors.Is(err, repository.ErrAvatarUpdateTooFrequent):
			default:
				errorsCh <- err
			}
		}(i)
	}
	close(start)
	wg.Wait()
	close(errorsCh)
	for err := range errorsCh {
		t.Errorf("unexpected concurrent update error: %v", err)
	}
	if got := successes.Load(); got != 1 {
		t.Fatalf("successful concurrent updates=%d, want 1", got)
	}
}
