package gormrepo

import (
	"errors"
	"strings"
	"testing"

	"my-bbs/internal/repository"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestTranslateError_MapsDatabaseErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want error
	}{
		{
			name: "gorm duplicated key",
			err:  gorm.ErrDuplicatedKey,
			want: repository.ErrAlreadyExists,
		},
		{
			name: "gorm invalid field",
			err:  gorm.ErrInvalidField,
			want: repository.ErrFieldNotFound,
		},
		{
			name: "mysql duplicated key",
			err:  &mysql.MySQLError{Number: 1062, Message: "Duplicate entry 'alice' for key 'users.username'"},
			want: repository.ErrAlreadyExists,
		},
		{
			name: "mysql data too long",
			err:  &mysql.MySQLError{Number: 1406, Message: "Data too long for column 'nickname'"},
			want: repository.ErrFieldTooLong,
		},
		{
			name: "mysql unknown column",
			err:  &mysql.MySQLError{Number: 1054, Message: "Unknown column 'missing' in 'field list'"},
			want: repository.ErrFieldNotFound,
		},
		{
			name: "mysql column cannot be null",
			err:  &mysql.MySQLError{Number: 1048, Message: "Column 'username' cannot be null"},
			want: repository.ErrFieldRequired,
		},
		{
			name: "mysql field has no default",
			err:  &mysql.MySQLError{Number: 1364, Message: "Field 'username' doesn't have a default value"},
			want: repository.ErrFieldRequired,
		},
		{
			name: "mysql table does not exist",
			err:  &mysql.MySQLError{Number: 1146, Message: "Table 'bbs.missing' doesn't exist"},
			want: repository.ErrTableNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := translateError(tt.err)
			if !errors.Is(got, tt.want) {
				t.Fatalf("translateError() = %v, want errors.Is(_, %v)", got, tt.want)
			}
			if !errors.Is(got, tt.err) {
				t.Fatalf("translateError() did not preserve original error %v", tt.err)
			}
		})
	}
}

func TestTranslateError_MapsRawMySQLDuplicateKeyMessage(t *testing.T) {
	const message = "Error 1062: Duplicate entry 'alice' for key 'users.username'"
	err := translateError(errors.New(message))
	if !errors.Is(err, repository.ErrAlreadyExists) {
		t.Fatalf("want ErrAlreadyExists, got %v", err)
	}
	if !strings.Contains(err.Error(), message) {
		t.Fatalf("translated error does not contain original message: %v", err)
	}
}

func TestTranslateError_LeavesUnknownErrorUntouched(t *testing.T) {
	original := errors.New("connection reset")
	if got := translateError(original); got != original {
		t.Fatalf("translateError() = %v, want original error", got)
	}
}

func TestTranslateError_Nil(t *testing.T) {
	if got := translateError(nil); got != nil {
		t.Fatalf("translateError(nil) = %v, want nil", got)
	}
}
