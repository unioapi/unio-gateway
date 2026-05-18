package routing

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThankCat/unio-api/internal/channel"
	"github.com/ThankCat/unio-api/internal/store/sqlc"
)

const defaultChannelTimeout = 30 * time.Second

var (
	// ErrModelNotFound 表示请求的模型不存在或没有启用。
	ErrModelNotFound = errors.New("model not found")
	// ErrNoAvailableChannel 表示模型存在但当前没有可用渠道。
	ErrNoAvailableChannel = errors.New("no available channel")
)

// ChatRouteRequest 表示一次 routing 选择所需上下文。
type ChatRouteRequest struct {
	ProjectID int64
	ModelID   string
}

// ChatRouteCandidate 表示一个可尝试的 chat 上游候选。
type ChatRouteCandidate struct {
	ModelDBID     int64
	ProviderID    int64
	AdapterKey    string
	Channel       channel.Runtime
	UpstreamModel string
}

// ChatRoutePlan 表示一次 chat 请求的同模型候选计划。
type ChatRoutePlan struct {
	RequestedModel string
	Candidates     []ChatRouteCandidate
}

// Store 定义 routing 查询候选渠道所需的最小数据库能力。
type Store interface {
	FindRouteCandidates(ctx context.Context, arg sqlc.FindRouteCandidatesParams) ([]sqlc.FindRouteCandidatesRow, error)
}

// CredentialResolver 根据 channel 保存的凭据引用解析出上游 API key。
type CredentialResolver interface {
	Resolve(ctx context.Context, credentialRef string) (string, error)
}

// Router 负责根据 project 和 requested model 选择可用 channel。
type Router struct {
	store              Store
	credentialResolver CredentialResolver
	defaultTimeout     time.Duration
}

// NewRouter 创建 routing router。
func NewRouter(store Store, credentialResolver CredentialResolver, defaultTimeout time.Duration) *Router {
	if defaultTimeout <= 0 {
		defaultTimeout = defaultChannelTimeout
	}

	return &Router{
		store:              store,
		credentialResolver: credentialResolver,
		defaultTimeout:     defaultTimeout,
	}
}

// PlanChat 为 chat completion 请求生成有序候选计划。
func (r *Router) PlanChat(ctx context.Context, req ChatRouteRequest) (ChatRoutePlan, error) {
	rows, err := r.findCandidateRows(ctx, req)
	if err != nil {
		return ChatRoutePlan{}, err
	}

	candidates := make([]ChatRouteCandidate, 0, len(rows))
	for _, row := range rows {
		candidate, err := r.buildChatRouteCandidate(ctx, row)
		if err != nil {
			return ChatRoutePlan{}, err
		}
		candidates = append(candidates, candidate)
	}

	return ChatRoutePlan{
		RequestedModel: req.ModelID,
		Candidates:     candidates,
	}, nil
}

func (r *Router) findCandidateRows(ctx context.Context, req ChatRouteRequest) ([]sqlc.FindRouteCandidatesRow, error) {
	rows, err := r.store.FindRouteCandidates(ctx, sqlc.FindRouteCandidatesParams{
		RequestedModelID: req.ModelID,
		ProjectID:        req.ProjectID,
	})
	if err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		// TODO(阶段6/production): 当前候选查询无法区分模型不存在和模型存在但无可用 channel，错误映射会不准确；实现 gateway 错误映射或后台模型可见性校验时；增加 ModelExists/GetEnabledModelByID 查询，并分别返回 ErrModelNotFound 与 ErrNoAvailableChannel。
		return nil, ErrNoAvailableChannel
	}

	return rows, nil
}

func (r *Router) buildChatRouteCandidate(ctx context.Context, row sqlc.FindRouteCandidatesRow) (ChatRouteCandidate, error) {
	apiKey, err := r.credentialResolver.Resolve(ctx, row.CredentialRef)
	if err != nil {
		return ChatRouteCandidate{}, fmt.Errorf("resolve channel credential: %w", err)
	}

	timeout := r.defaultTimeout
	if row.TimeoutMs.Valid {
		timeout = time.Duration(row.TimeoutMs.Int32) * time.Millisecond
	}

	return ChatRouteCandidate{
		ModelDBID:  row.ModelDbID,
		ProviderID: row.ProviderID,
		AdapterKey: row.AdapterKey,
		Channel: channel.Runtime{
			ID:      row.ChannelID,
			BaseURL: row.BaseUrl,
			APIKey:  apiKey,
			Timeout: timeout,
		},
		UpstreamModel: row.UpstreamModel,
	}, nil
}
