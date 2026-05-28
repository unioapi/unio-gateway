package credential

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ThankCat/unio-api/internal/platform/failure"
)

var (
	// ErrCredentialRefMissing 表示 credential_ref 为空。
	ErrCredentialRefMissing = errors.New("credential ref is empty")

	// ErrCredentialNotFound 表示 credential_ref 没有对应凭据。
	ErrCredentialNotFound = errors.New("credential not found")
)

// Resolver 根据凭据引用解析上游调用所需的明文凭据。
type Resolver interface {
	Resolve(ctx context.Context, credentialRef string) (string, error)
}

// StaticResolver 是开发期凭据解析器，通过内存映射解析 credential_ref。
// TODO(阶段6/production): [GAP-6-001] 静态凭据映射无法支持后台动态管理和安全轮换；接入 KMS/master key 或 secret manager 时；替换为安全凭据解析实现，并确保数据库只保存 credential_ref 或密文。
type StaticResolver struct {
	values map[string]string
}

// NewStaticResolver 创建开发期静态凭据解析器。
func NewStaticResolver(values map[string]string) *StaticResolver {
	copied := make(map[string]string, len(values))
	for k, v := range values {
		copied[k] = v
	}

	return &StaticResolver{values: copied}
}

// Resolve 根据 credential_ref 返回对应上游 API key。
func (r *StaticResolver) Resolve(ctx context.Context, credentialRef string) (string, error) {
	if strings.TrimSpace(credentialRef) == "" {
		return "", failure.Wrap(
			failure.CodeCredentialRefMissing,
			ErrCredentialRefMissing,
			failure.WithMessage(ErrCredentialRefMissing.Error()),
		)
	}

	value, ok := r.values[credentialRef]
	if !ok {
		return "", failure.Wrap(
			failure.CodeCredentialNotFound,
			ErrCredentialNotFound,
			failure.WithMessage(fmt.Sprintf("credential ref %q not found", credentialRef)),
		)
	}

	return value, nil
}
