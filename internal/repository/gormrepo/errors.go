package gormrepo

import (
	"errors"
	"fmt"
	"strings"

	"my-bbs/internal/repository"

	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func translateError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return wrapRepositoryError(repository.ErrAlreadyExists, err)
	}
	if errors.Is(err, gorm.ErrInvalidField) {
		return wrapRepositoryError(repository.ErrFieldNotFound, err)
	}

	var mysqlErr *mysql.MySQLError
	if errors.As(err, &mysqlErr) {
		if repositoryErr := mysqlRepositoryError(mysqlErr.Number); repositoryErr != nil {
			return wrapRepositoryError(repositoryErr, err)
		}
	}

	if isDuplicateKey(err) {
		return wrapRepositoryError(repository.ErrAlreadyExists, err)
	}
	return err
}

// mysqlRepositoryError 按 MySQL 错误码转换为 Repository Port 的稳定错误语义。
func mysqlRepositoryError(number uint16) error {
	switch number {
	case 1062: // ER_DUP_ENTRY
		return repository.ErrAlreadyExists
	case 1406: // ER_DATA_TOO_LONG
		return repository.ErrFieldTooLong
	case 1054: // ER_BAD_FIELD_ERROR
		return repository.ErrFieldNotFound
	case 1048, 1364: // ER_BAD_NULL_ERROR, ER_NO_DEFAULT_FOR_FIELD
		return repository.ErrFieldRequired
	case 1146: // ER_NO_SUCH_TABLE
		return repository.ErrTableNotFound
	default:
		return nil
	}
}

// wrapRepositoryError 同时保留稳定的哨兵错误和数据库原始错误。
// 调用方可用 errors.Is 判断错误类型，日志仍能输出具体字段、表名等信息。
func wrapRepositoryError(repositoryErr, cause error) error {
	return fmt.Errorf("%w: %w", repositoryErr, cause)
}

// isDuplicateKey 兼容尚未开启 TranslateError 时的 MySQL 1062 错误。
func isDuplicateKey(err error) bool {
	message := err.Error()
	return strings.Contains(message, "Duplicate entry") || strings.Contains(message, "1062")
}
