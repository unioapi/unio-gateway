package failure

import "errors"

// Field 表示 Failure 可展开到日志、审计和指标中的结构化上下文字段。
type Field struct {
	Key   string
	Value any
}

// Failure 表示一个已经归类的 Unio 错误。
type Failure struct {
	Code    Code
	Message string
	Cause   error
	Fields  []Field
}

// Error 返回安全的错误摘要。
func (f *Failure) Error() string {
	if f == nil {
		return ""
	}

	if f.Message != "" {
		return f.Message
	}

	return string(f.Code)
}

// Unwrap 返回底层 cause，支持 errors.Is / errors.As 继续匹配原始错误。
func (f *Failure) Unwrap() error {
	if f == nil {
		return nil
	}

	return f.Cause
}

// New 创建一个没有底层 cause 的 Failure。
func New(code Code, opts ...Option) error {
	return build(code, nil, opts...)
}

// Wrap 创建一个携带底层 cause 的 Failure。
func Wrap(code Code, cause error, opts ...Option) error {
	return build(code, cause, opts...)
}

// CodeOf 从 error 链中提取 Failure code；未知错误返回空字符串。
func CodeOf(err error) Code {
	var f *Failure
	if errors.As(err, &f) {
		return f.Code
	}

	return ""
}

// CategoryOf 从 error 链中提取 Failure category；未知错误返回 unknown。
func CategoryOf(err error) Category {
	var f *Failure
	if errors.As(err, &f) {
		return f.Code.Category()
	}

	return CategoryUnknown
}

// FieldsOf 从 error 链中提取结构化错误字段；未知错误返回 nil。
func FieldsOf(err error) []Field {
	var f *Failure
	if !errors.As(err, &f) || len(f.Fields) == 0 {
		return nil
	}

	fields := make([]Field, len(f.Fields))
	copy(fields, f.Fields)
	return fields
}

func build(code Code, cause error, opts ...Option) error {
	f := &Failure{
		Code:  code,
		Cause: cause,
	}

	for _, opt := range opts {
		opt(f)
	}

	return f
}
