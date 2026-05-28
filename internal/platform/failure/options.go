package failure

// Option 修改 Failure 的可选字段。
type Option func(*Failure)

// WithMessage 设置安全错误摘要。
func WithMessage(message string) Option {
	return func(f *Failure) {
		f.Message = message
	}
}

// WithField 追加一个结构化上下文字段。
func WithField(key string, value any) Option {
	return func(f *Failure) {
		if key == "" {
			return
		}

		f.Fields = append(f.Fields, Field{
			Key:   key,
			Value: value,
		})
	}
}
