package httpapi

// ChatCompletionRequest 表示 OpenAI-compatible chat completions 请求体。
type ChatCompletionRequest struct {
	Model string `json:"model"`

	// 聊天消息列表
	Messages []ChatMessage `json:"messages"`

	// 是否启用流式响应。流式响应指服务端逐段返回内容，而不是一次性返回完整结果。
	Stream *bool `json:"stream,omitempty"`

	// 采样温度，控制输出随机性。
	Temperature *float64 `json:"temperature,omitempty"`

	// 核采样参数，控制候选 token 的概率范围。可以理解为另一种控制随机性的参数。
	TopP *float64 `json:"top_p,omitempty"`

	// 最大输出 token 数。token 是模型处理文本的基本单位，可以粗略理解为词片段。
	MaxTokens *int `json:"max_tokens,omitempty"`

	// 存在惩罚，用来降低模型重复谈论已经出现过主题的倾向。
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`

	// 频率惩罚，用来降低模型重复使用已经出现过词语的倾向。
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`

	// 停止序列。模型生成内容遇到这些字符串时停止。
	Stop []string `json:"stop,omitempty"`

	//终端用户标识。一般用于审计、风控或上游服务追踪。
	User *string `json:"user,omitempty"`
}

// ChatMessage 表示 chat completions 请求或响应中的一条消息。
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatCompletionResponse 表示 OpenAI-compatible chat completions 响应体。
type ChatCompletionResponse struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int64                  `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatCompletionChoice `json:"choices"`
	Usage   ChatCompletionUsage    `json:"usage"`
}

// ChatCompletionChoice 表示 chat completions 响应中的一个候选结果。
type ChatCompletionChoice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// ChatCompletionUsage 表示 chat completions 请求的 token 用量统计。
type ChatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
