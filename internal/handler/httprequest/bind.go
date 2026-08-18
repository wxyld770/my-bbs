// Package httprequest defines the request models and JSON binding rules of the
// HTTP transport layer.
package httprequest

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"my-bbs/pkg/bizerr"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// MaxJSONBodyBytes is the largest JSON request body accepted by BindJSON.
const MaxJSONBodyBytes int64 = 1 << 20 // 1 MiB

// BindJSON decodes and validates one JSON object from the request body.
//
// It accepts application/json (including parameters such as charset), limits
// the body size, rejects unknown fields and additional JSON values, and uses
// Gin's configured validator so the existing binding tags remain authoritative.
func BindJSON(c *gin.Context, dst any) error {
	if err := validateJSONContentType(c.GetHeader("Content-Type")); err != nil {
		return err
	}

	if c.Request.ContentLength > MaxJSONBodyBytes {
		return bizerr.ErrPayloadTooLarge
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, MaxJSONBodyBytes)
	decoder := json.NewDecoder(c.Request.Body)

	var raw json.RawMessage
	if err := decoder.Decode(&raw); err != nil {
		return requestDecodeError(err)
	}

	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return badRequest("请求体只能包含一个 JSON 对象")
		}
		return requestDecodeError(err)
	}

	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 || trimmed[0] != '{' {
		return badRequest("请求体必须是 JSON 对象")
	}

	strictDecoder := json.NewDecoder(bytes.NewReader(raw))
	strictDecoder.DisallowUnknownFields()
	if err := strictDecoder.Decode(dst); err != nil {
		return requestDecodeError(err)
	}

	if err := binding.Validator.ValidateStruct(dst); err != nil {
		return validationError(dst, err)
	}
	return nil
}

func validateJSONContentType(contentType string) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil || !strings.EqualFold(mediaType, binding.MIMEJSON) {
		return bizerr.ErrUnsupportedMediaType
	}
	return nil
}

func requestDecodeError(err error) error {
	var maxBytesError *http.MaxBytesError
	if errors.As(err, &maxBytesError) {
		return bizerr.ErrPayloadTooLarge
	}

	if errors.Is(err, io.EOF) {
		return badRequest("请求体不能为空")
	}

	var typeError *json.UnmarshalTypeError
	if errors.As(err, &typeError) {
		if typeError.Field == "" {
			return badRequest("请求体必须是 JSON 对象")
		}
		return badRequest(fmt.Sprintf("字段 %s 类型错误", typeError.Field))
	}

	var syntaxError *json.SyntaxError
	if errors.As(err, &syntaxError) || errors.Is(err, io.ErrUnexpectedEOF) || err.Error() == "unexpected EOF" {
		return badRequest("请求体 JSON 格式错误")
	}

	if field, ok := unknownJSONField(err); ok {
		return badRequest(fmt.Sprintf("请求体包含未知字段: %s", field))
	}

	// InvalidUnmarshalError indicates a programmer error (for example, dst is
	// not a pointer), so do not disguise it as bad client input.
	var invalidUnmarshalError *json.InvalidUnmarshalError
	if errors.As(err, &invalidUnmarshalError) {
		return err
	}

	return badRequest("请求体 JSON 格式错误")
}

func unknownJSONField(err error) (string, bool) {
	const prefix = "json: unknown field "
	message := err.Error()
	if !strings.HasPrefix(message, prefix) {
		return "", false
	}

	quoted := strings.TrimPrefix(message, prefix)
	field, unquoteErr := strconv.Unquote(quoted)
	if unquoteErr != nil {
		return "", false
	}
	return field, true
}

func validationError(dst any, err error) error {
	var fieldErrors validator.ValidationErrors
	if !errors.As(err, &fieldErrors) || len(fieldErrors) == 0 {
		return badRequest("请求参数校验失败")
	}

	fieldError := fieldErrors[0]
	field := jsonFieldName(dst, fieldError.StructField())
	switch fieldError.Tag() {
	case "required":
		return badRequest(fmt.Sprintf("字段 %s 不能为空", field))
	case "min":
		return badRequest(fmt.Sprintf("字段 %s 长度不能少于 %s 个字符", field, fieldError.Param()))
	case "max":
		return badRequest(fmt.Sprintf("字段 %s 长度不能超过 %s 个字符", field, fieldError.Param()))
	case "oneof":
		return badRequest(fmt.Sprintf("字段 %s 的值无效", field))
	default:
		return badRequest(fmt.Sprintf("字段 %s 参数无效", field))
	}
}

func jsonFieldName(dst any, structField string) string {
	typeOf := reflect.TypeOf(dst)
	for typeOf != nil && typeOf.Kind() == reflect.Pointer {
		typeOf = typeOf.Elem()
	}
	if typeOf != nil && typeOf.Kind() == reflect.Struct {
		if field, ok := typeOf.FieldByName(structField); ok {
			name := strings.Split(field.Tag.Get("json"), ",")[0]
			if name != "" && name != "-" {
				return name
			}
		}
	}
	return structField
}

func badRequest(message string) error {
	return bizerr.ErrBadRequest.WithMessage(message)
}
