package failure

// LogArgs 将 error 展开成 slog 可直接接收的 key/value 参数。
func LogArgs(err error) []any {
	if err == nil {
		return nil
	}

	args := []any{
		"error", err,
	}

	code := CodeOf(err)
	if code == "" {
		return args
	}

	args = append(args,
		"error_code", string(code),
		"error_category", string(code.Category()),
	)

	for _, field := range FieldsOf(err) {
		if field.Key == "" {
			continue
		}

		args = append(args, field.Key, field.Value)
	}

	return args
}
