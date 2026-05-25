package adapter

// ChatInputTokenizer 定义某个 provider adapter 对 chat 输入 token 的计数能力。
type ChatInputTokenizer interface {
	CountChatInputTokens(req ChatInputTokenizeRequest) (int64, error)
}

// ChatInputTokenizeRequest 表示 adapter 计算 chat 输入 token 所需的最小输入。
type ChatInputTokenizeRequest struct {
	Model    string
	Messages []ChatMessage
}
