package repository

import "errors"

// Repository Port 只暴露与具体数据库无关的持久化错误。
// Adapter 负责把 MySQL、SQLite 等数据库错误转换为这些哨兵错误。
var (
	ErrAlreadyExists = errors.New("repository record already exists")
	ErrFieldTooLong  = errors.New("repository field value is too long")
	ErrFieldNotFound = errors.New("repository field does not exist")
	ErrFieldRequired = errors.New("repository field is required")
	ErrTableNotFound = errors.New("repository table does not exist")
)
