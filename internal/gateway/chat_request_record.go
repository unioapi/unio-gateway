package gateway

import (
	"context"
	"errors"
	"time"

	"github.com/ThankCat/unio-api/internal/auth"
	"github.com/ThankCat/unio-api/internal/failure"
	"github.com/ThankCat/unio-api/internal/httpapi"
	"github.com/ThankCat/unio-api/internal/requestlog"
	"github.com/ThankCat/unio-api/internal/routing"
)

// createRequestRecord 创建用户可见请求记录，并立即推进到 running 状态。
// request_records.request_id 由服务端生成，用作数据库唯一事实 ID；
// HTTP X-Request-ID 只作为日志 correlation id，不能直接复用为账务请求 ID。
func (s *ChatCompletionService) createRequestRecord(ctx context.Context, principal *auth.APIKeyPrincipal, req httpapi.ChatCompletionRequest, stream bool) (requestlog.RequestRecord, error) {
	requestID, err := requestlog.GenerateRequestID()
	if err != nil {
		return requestlog.RequestRecord{}, err
	}

	record, err := s.requestLog.CreateRequest(ctx, requestlog.CreateRequestParams{
		RequestID:        requestID,
		UserID:           principal.UserID,
		ProjectID:        principal.ProjectID,
		APIKeyID:         principal.APIKeyID,
		RequestedModelID: req.Model,
		Stream:           stream,
		StartedAt:        time.Now(),
	})
	if err != nil {
		return requestlog.RequestRecord{}, err
	}

	record, err = s.requestLog.MarkRequestRunning(ctx, record.ID)
	if err != nil {
		return requestlog.RequestRecord{}, err
	}

	return record, nil
}

// createAttemptRecord 创建一次上游 channel 尝试记录。
// attempt 记录 fallback 链路中的单次 provider/channel 调用，必须先于 adapter 调用创建。
func (s *ChatCompletionService) createAttemptRecord(ctx context.Context, requestRecord requestlog.RequestRecord, attemptIndex int, candidate routing.ChatRouteCandidate) (requestlog.AttemptRecord, error) {
	attempt, err := s.requestLog.CreateAttempt(ctx, requestlog.CreateAttemptParams{
		RequestRecordID: requestRecord.ID,
		AttemptIndex:    attemptIndex,
		ProviderID:      candidate.ProviderID,
		ChannelID:       candidate.Channel.ID,
		AdapterKey:      candidate.AdapterKey,
		UpstreamModel:   candidate.UpstreamModel,
		StartedAt:       time.Now(),
	})
	if err != nil {
		return requestlog.AttemptRecord{}, err
	}

	return attempt, nil
}

// markRequestRecordSucceeded 把请求标记为成功，并记录最终响应模型和 provider/channel。
func (s *ChatCompletionService) markRequestRecordSucceeded(ctx context.Context, requestRecord requestlog.RequestRecord, req httpapi.ChatCompletionRequest, candidate routing.ChatRouteCandidate) error {
	_, err := s.requestLog.MarkRequestSucceeded(ctx, requestlog.MarkRequestSucceededParams{
		ID:              requestRecord.ID,
		ResponseModelID: req.Model,
		FinalProviderID: candidate.ProviderID,
		FinalChannelID:  candidate.Channel.ID,
		CompletedAt:     time.Now(),
	})
	return err
}

// markAttemptRecordSucceeded 把一次上游尝试标记为成功，并记录上游实际响应模型。
func (s *ChatCompletionService) markAttemptRecordSucceeded(ctx context.Context, attempt requestlog.AttemptRecord, upstreamModel string) error {
	_, err := s.requestLog.MarkAttemptSucceeded(ctx, requestlog.MarkAttemptSucceededParams{
		ID:                    attempt.ID,
		UpstreamResponseModel: upstreamModel,
		UpstreamStatusCode:    200,
		UpstreamRequestID:     nil,
		CompletedAt:           time.Now(),
	})
	return err
}

// routingFailureCode 将 routing 内部错误转换成 request_records.error_code。
func routingFailureCode(err error) string {
	if code := failure.CodeOf(err); code != "" {
		return string(code)
	}

	switch {
	case errors.Is(err, routing.ErrModelNotFound):
		return "model_not_found"
	case errors.Is(err, routing.ErrModelNotAvailable):
		return "model_not_available"
	case errors.Is(err, routing.ErrNoAvailableChannel):
		return "no_available_channel"
	default:
		return "routing_error"
	}
}

// markRequestRecordFailed 把 request record 标记为失败。
// 失败状态写入是审计动作，调用方仍然返回原始业务错误，避免状态写入细节覆盖根因。
func (s *ChatCompletionService) markRequestRecordFailed(ctx context.Context, requestRecord requestlog.RequestRecord, code string, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	// TODO(阶段7/production): [GAP-7-005] request_records.error_message 当前保存原始内部错误，未来后台暴露请求日志时可能泄漏上游路径、配置细节或敏感上下文；开放请求日志查询前；区分 safe_user_message、internal_error_detail，并对后台展示做脱敏。
	_, _ = s.requestLog.MarkRequestFailed(ctx, requestlog.MarkRequestFailedParams{
		ID:           requestRecord.ID,
		ErrorCode:    failureCodeOrFallback(err, code),
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})
}

// markAttemptRecordFailed 把一次上游尝试标记为失败。
// 失败状态写入是审计动作，调用方仍然返回原始业务错误，避免状态写入细节覆盖根因。
func (s *ChatCompletionService) markAttemptRecordFailed(ctx context.Context, attempt requestlog.AttemptRecord, code string, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	_, _ = s.requestLog.MarkAttemptFailed(ctx, requestlog.MarkAttemptFailedParams{
		ID:           attempt.ID,
		ErrorCode:    failureCodeOrFallback(err, code),
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})
}

func failureCodeOrFallback(err error, fallback string) string {
	if code := failure.CodeOf(err); code != "" {
		return string(code)
	}

	return fallback
}

// markRequestCanceled 把 request record 和当前 attempt 标记为客户端取消。
func (s *ChatCompletionService) markRequestCanceled(ctx context.Context, requestRecord requestlog.RequestRecord, attemptRecord requestlog.AttemptRecord, err error) {
	message := ""
	if err != nil {
		message = err.Error()
	}

	// 账务 release 或 risk_exposure 由调用方在进入 canceled 状态前处理；这里仅写请求审计状态。
	// 客户端断开时原请求 ctx 通常已经取消；这里脱离请求取消，
	// 给审计写入一个很短的补偿窗口，避免 canceled 状态写不进去。
	auditCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
	defer cancel()

	_, _ = s.requestLog.MarkAttemptCanceled(auditCtx, requestlog.MarkAttemptCanceledParams{
		ID:           attemptRecord.ID,
		ErrorCode:    "client_canceled",
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})

	_, _ = s.requestLog.MarkRequestCanceled(auditCtx, requestlog.MarkRequestCanceledParams{
		ID:           requestRecord.ID,
		ErrorCode:    "client_canceled",
		ErrorMessage: message,
		CompletedAt:  time.Now(),
	})
}
