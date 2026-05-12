package httpapi

import (
	"net/http"

	"github.com/ThankCat/unio-api/internal/httpx"
)

// handleModels 返回当前可用模型列表的 OpenAI-compatible 占位响应。
func handleModels(w http.ResponseWriter, r *http.Request) {
	// TODO(阶段6/production): /v1/models 不能长期返回空列表，必须从 model catalog 和 channel availability 生成当前 project 可用模型。
	_ = httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"object": "list",
		"data":   []any{},
	})
}
