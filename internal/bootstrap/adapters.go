package bootstrap

import (
	"net/http"

	"github.com/ThankCat/unio-api/internal/adapter"
	"github.com/ThankCat/unio-api/internal/adapter/openai"
)

// NewAdapterRegistry 创建当前 server 进程支持的 adapter registry。
func NewAdapterRegistry(client *http.Client) (*adapter.Registry, error) {
	openaiAdapter := openai.NewAdapter(client)
	return adapter.NewRegistry(adapter.Registration{
		Key:                "openai",
		Chat:               openaiAdapter,
		StreamChat:         openaiAdapter,
		ChatInputTokenizer: openaiAdapter,
	})
}
