package service_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/service"
	"my-bbs/pkg/bizerr"
)

type memoryUserRepository struct {
	nextID     uint
	byID       map[uint]*model.User
	byUsername map[string]*model.User
	createErr  error
}

var _ repository.UserRepository = (*memoryUserRepository)(nil)

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		nextID:     1,
		byID:       make(map[uint]*model.User),
		byUsername: make(map[string]*model.User),
	}
}

func (r *memoryUserRepository) CreateUser(_ context.Context, user *model.User) error {
	if r.createErr != nil {
		return r.createErr
	}
	user.ID = r.nextID
	r.nextID++
	r.byID[user.ID] = user
	r.byUsername[user.Username] = user
	return nil
}

func TestUserService_MapsRepositoryConflict(t *testing.T) {
	repo := newMemoryUserRepository()
	repo.createErr = repository.ErrAlreadyExists
	svc := service.NewUserService(repo)

	err := svc.Register(context.Background(), "conflict_user", "password1", "Conflict")
	if !errors.Is(err, bizerr.ErrUsernameExists) {
		t.Fatalf("want ErrUsernameExists, got %v", err)
	}
}

func (r *memoryUserRepository) FindUserByUsername(_ context.Context, username string) (*model.User, error) {
	return r.byUsername[username], nil
}

func (r *memoryUserRepository) FindUserByID(_ context.Context, id uint) (*model.User, error) {
	return r.byID[id], nil
}

func (r *memoryUserRepository) UpdateProfile(_ context.Context, id uint, nickname, introduction string) error {
	user := r.byID[id]
	if user != nil {
		user.Nickname = nickname
		user.Introduction = introduction
	}
	return nil
}

func TestUserService_DependsOnRepositoryPort(t *testing.T) {
	ctx := context.Background()
	repo := newMemoryUserRepository()
	svc := service.NewUserService(repo)

	if err := svc.Register(ctx, "port_user", "password1", "Port User"); err != nil {
		t.Fatalf("register through repository port: %v", err)
	}
	stored := repo.byUsername["port_user"]
	if stored == nil || stored.ID == 0 {
		t.Fatalf("user was not persisted through port: %+v", stored)
	}
	if stored.Password == "password1" {
		t.Fatal("service should hash password before calling repository port")
	}
	if stored.Status != model.UserStatusNormal {
		t.Fatalf("status=%d, want normal", stored.Status)
	}

	profile, err := svc.GetMe(ctx, stored.ID)
	if err != nil || profile != stored {
		t.Fatalf("read through repository port: profile=%+v err=%v", profile, err)
	}
}
