package service

import (
	"strings"
	"unicode/utf8"

	"my-bbs/pkg/bizerr"
)

// MySQL TEXT 最多保存 65,535 字节。HTTP 的 body 上限只是防止过大请求，
// 应用层仍需校验真正会写入数据库的字段大小。
const maxTextFieldBytes = 65_535

func requiredTrimmed(value, emptyMessage string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", bizerr.ErrBadRequest.WithMessage(emptyMessage)
	}
	return value, nil
}

func validateRuneLength(value, field string, minLength, maxLength int) error {
	length := utf8.RuneCountInString(value)
	if minLength > 0 && length < minLength {
		return bizerr.ErrBadRequest.WithMessagef("%s长度不能少于%d个字符", field, minLength)
	}
	if maxLength > 0 && length > maxLength {
		return bizerr.ErrBadRequest.WithMessagef("%s长度不能超过%d个字符", field, maxLength)
	}
	return nil
}

func validateByteLength(value, field string, maxLength int) error {
	if len(value) > maxLength {
		return bizerr.ErrBadRequest.WithMessagef("%s不能超过%d字节", field, maxLength)
	}
	return nil
}
