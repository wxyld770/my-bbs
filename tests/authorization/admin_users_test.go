package authorization_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/authorization"
	"my-bbs/internal/model"
)

type userLookupStub struct {
	users map[string]*model.User
}

func (s userLookupStub) FindUserByUsername(_ context.Context, username string) (*model.User, error) {
	return s.users[username], nil
}

func TestParseAdminUsers_CommaSeparatedConfiguration(t *testing.T) {
	admins, err := authorization.ParseAdminUsers(" admin, moderator ,admin ")
	if err != nil {
		t.Fatalf("ParseAdminUsers: %v", err)
	}
	for _, username := range []string{"admin", "moderator"} {
		if !admins.IsAdminUsername(username) {
			t.Fatalf("configured username %q must be an admin", username)
		}
	}
	if admins.IsAdminUsername("ordinary") || admins.IsAdminUsername("Admin") {
		t.Fatal("admin matching must use configured account names exactly")
	}
}

func TestParseAdminUsers_RejectsEmptyConfiguration(t *testing.T) {
	_, err := authorization.ParseAdminUsers(" , ")
	if !errors.Is(err, authorization.ErrNoAdminUsernames) {
		t.Fatalf("error=%v, want ErrNoAdminUsernames", err)
	}
}

func TestValidateExistingAdminUsers_AcceptsOnlyPreExistingExactAccounts(t *testing.T) {
	admins := authorization.NewAdminUsers("admin", "moderator")
	lookup := userLookupStub{users: map[string]*model.User{
		"admin":     {Username: "admin"},
		"moderator": {Username: "moderator"},
	}}

	if err := authorization.ValidateExistingAdminUsers(context.Background(), lookup, admins); err != nil {
		t.Fatalf("ValidateExistingAdminUsers: %v", err)
	}
}

func TestValidateExistingAdminUsers_RejectsMissingAccountWithoutCreatingIt(t *testing.T) {
	admins := authorization.NewAdminUsers("admin")
	lookup := userLookupStub{users: make(map[string]*model.User)}

	err := authorization.ValidateExistingAdminUsers(context.Background(), lookup, admins)
	if !errors.Is(err, authorization.ErrAdminUserNotFound) {
		t.Fatalf("error=%v, want ErrAdminUserNotFound", err)
	}
	if _, exists := lookup.users["admin"]; exists {
		t.Fatal("validating ADMIN_USERNAMES must not create an account")
	}
}

func TestValidateExistingAdminUsers_RejectsDatabaseCollationMismatch(t *testing.T) {
	admins := authorization.NewAdminUsers("admin")
	lookup := userLookupStub{users: map[string]*model.User{
		"admin": {Username: "Admin"},
	}}

	err := authorization.ValidateExistingAdminUsers(context.Background(), lookup, admins)
	if !errors.Is(err, authorization.ErrAdminUserNotFound) {
		t.Fatalf("error=%v, want ErrAdminUserNotFound", err)
	}
}
