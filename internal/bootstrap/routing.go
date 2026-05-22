package bootstrap

import (
	"time"

	"github.com/ThankCat/unio-api/internal/credential"
	"github.com/ThankCat/unio-api/internal/routing"
)

const defaultChatRouteTimeout = 30 * time.Second

// NewChatRouter 创建当前 server 进程使用的 chat routing 组件。
func NewChatRouter(store routing.Store) (*routing.Router, error) {
	credentialResolver := credential.NewStaticResolver(make(map[string]string))
	return routing.NewRouter(store, credentialResolver, defaultChatRouteTimeout), nil
}
