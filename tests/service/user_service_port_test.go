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

type memoryInvitationRepository struct {
	userRepo  *memoryUserRepository
	available map[string]bool
}

var _ repository.UserRepository = (*memoryUserRepository)(nil)
var _ repository.InvitationRepository = (*memoryInvitationRepository)(nil)

func newMemoryUserRepository() *memoryUserRepository {
	return &memoryUserRepository{
		nextID:     1,
		byID:       make(map[uint]*model.User),
		byUsername: make(map[string]*model.User),
	}
}

func newMemoryInvitationRepository(userRepo *memoryUserRepository, codes ...string) *memoryInvitationRepository {
	available := make(map[string]bool, len(codes))
	for _, code := range codes {
		available[code] = true
	}
	return &memoryInvitationRepository{userRepo: userRepo, available: available}
}

func (r *memoryInvitationRepository) CreateInvitation(_ context.Context, invitation *model.Invitation) error {
	if _, exists := r.available[invitation.Code]; exists {
		return repository.ErrAlreadyExists
	}
	r.available[invitation.Code] = true
	return nil
}

func (r *memoryInvitationRepository) RegisterUserWithInvitation(
	ctx context.Context,
	user *model.User,
	code string,
) error {
	if !r.available[code] {
		return repository.ErrInvitationUnavailable
	}
	user.InviteCode = &code
	if err := r.userRepo.CreateUser(ctx, user); err != nil {
		return err
	}
	r.available[code] = false
	return nil
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
	invitationRepo := newMemoryInvitationRepository(repo, "CNFL01")
	svc := service.NewUserServiceWithInvitations(repo, invitationRepo)

	err := svc.Register(context.Background(), "conflict_user", "password1", "Conflict", "CNFL01")
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

func (r *memoryUserRepository) UpdateUserStatus(_ context.Context, id uint, status uint) error {
	user := r.byID[id]
	if user != nil {
		user.Status = status
	}
	return nil
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
	invitationRepo := newMemoryInvitationRepository(repo, "PORT01")
	svc := service.NewUserServiceWithInvitations(repo, invitationRepo)

	if err := svc.Register(ctx, "port_user", "password1", "Port User", "PORT01"); err != nil {
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
	if stored.InviteCode == nil || *stored.InviteCode != "PORT01" {
		t.Fatalf("invite_code=%v, want PORT01", stored.InviteCode)
	}

	profile, err := svc.GetMe(ctx, stored.ID)
	if err != nil || profile != stored {
		t.Fatalf("read through repository port: profile=%+v err=%v", profile, err)
	}
}
