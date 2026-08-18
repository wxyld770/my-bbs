package repository_test

import (
	"context"
	"errors"
	"testing"

	"my-bbs/internal/model"
	"my-bbs/internal/repository"
	"my-bbs/internal/repository/gormrepo"
	"my-bbs/tests/testutil"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestUserRepository_TranslatesDatabaseErrors(t *testing.T) {
	tests := []struct {
		name  string
		cause error
		want  error
	}{
		{
			name:  "duplicate key",
			cause: &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'alice' for key 'users.username'"},
			want:  repository.ErrAlreadyExists,
		},
		{
			name:  "field too long",
			cause: &mysql.MySQLError{Number: 1406, Message: "Data too long for column 'nickname'"},
			want:  repository.ErrFieldTooLong,
		},
		{
			name:  "field not found",
			cause: &mysql.MySQLError{Number: 1054, Message: "Unknown column 'missing' in 'field list'"},
			want:  repository.ErrFieldNotFound,
		},
		{
			name:  "null field",
			cause: &mysql.MySQLError{Number: 1048, Message: "Column 'username' cannot be null"},
			want:  repository.ErrFieldRequired,
		},
		{
			name:  "missing required field",
			cause: &mysql.MySQLError{Number: 1364, Message: "Field 'username' doesn't have a default value"},
			want:  repository.ErrFieldRequired,
		},
		{
			name:  "table not found",
			cause: &mysql.MySQLError{Number: 1146, Message: "Table 'bbs.missing' doesn't exist"},
			want:  repository.ErrTableNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testutil.NewTestDB(t)
			if err := db.Callback().Create().Before("gorm:create").Register("test:inject_database_error", func(tx *gorm.DB) {
				tx.AddError(tt.cause)
			}); err != nil {
				t.Fatalf("register callback: %v", err)
			}

			repo := gormrepo.NewUserRepository(db)
			err := repo.CreateUser(context.Background(), &model.User{
				Username: "alice",
				Password: "hash",
				Nickname: "Alice",
			})
			if !errors.Is(err, tt.want) {
				t.Fatalf("CreateUser() error = %v, want errors.Is(_, %v)", err, tt.want)
			}
			if !errors.Is(err, tt.cause) {
				t.Fatalf("CreateUser() error did not preserve database cause: %v", err)
			}
		})
	}
}
