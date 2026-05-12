package openai

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionResponse struct {
	ID      string              `json:"id"`
	Model   string              `json:"model"`
	Choices []chatChoice        `json:"choices"`
	Usage   chatCompletionUsage `json:"usage"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}

type chatCompletionUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}
