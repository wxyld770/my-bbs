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

	// gorm.ErrRecordNotFound 是 Adapter 的实现细节，不向 Port 调用方泄漏。
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return repository.ErrNotFound
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

// translateDeleteResult 把删除的零影响行数转换为稳定的 Port 错误。
// 更新操作不能复用这条规则：MySQL 默认会在“值没有变化”时返回
// RowsAffected == 0，此时记录仍然存在。
func translateDeleteResult(result *gorm.DB) error {
	if result.Error != nil {
		return translateError(result.Error)
	}
	if result.RowsAffected == 0 {
		return repository.ErrNotFound
	}
	return nil
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
