package requestlog

import (
	"context"
	"time"
)

// RequestStatus 表示用户可见请求的生命周期状态。
type RequestStatus string

const (
	RequestStatusPending   RequestStatus = "pending"
	RequestStatusRunning   RequestStatus = "running"
	RequestStatusSucceeded RequestStatus = "succeeded"
	RequestStatusFailed    RequestStatus = "failed"
	RequestStatusCanceled  RequestStatus = "canceled"
)

// AttemptStatus 表示一次上游 channel 尝试的生命周期状态。
type AttemptStatus string

const (
	AttemptStatusRunning   AttemptStatus = "running"
	AttemptStatusSucceeded AttemptStatus = "succeeded"
	AttemptStatusFailed    AttemptStatus = "failed"
	AttemptStatusCanceled  AttemptStatus = "canceled"
)

// CreateRequestParams 表示创建 request record 所需的请求事实。
type CreateRequestParams struct {
	RequestID        string
	UserID           int64
	ProjectID        int64
	APIKeyID         int64
	RequestedModelID string
	Stream           bool
	StartedAt        time.Time
}

// RequestRecord 表示一次用户可见请求记录。
type RequestRecord struct {
	ID                  int64
	RequestID           string
	UserID              int64
	ProjectID           int64
	APIKeyID            int64
	RequestedModelID    string
	ResponseModelID     *string
	Stream              bool
	Status              RequestStatus
	FinalProviderID     *int64
	FinalChannelID      *int64
	ErrorCode           *string
	ErrorMessage        *string
	InternalErrorDetail *string
	StartedAt           time.Time
	CompletedAt         *time.Time
}

// MarkRequestSucceededParams 表示标记请求成功所需的最终事实。
type MarkRequestSucceededParams struct {
	ID              int64
	ResponseModelID string
	FinalProviderID int64
	FinalChannelID  int64
	CompletedAt     time.Time
}

// MarkRequestFailedParams 表示标记请求失败所需的错误事实。
type MarkRequestFailedParams struct {
	ID                  int64
	ErrorCode           string
	ErrorMessage        string
	InternalErrorDetail string
	CompletedAt         time.Time
}

// MarkRequestCanceledParams 表示标记请求取消所需的错误事实。
type MarkRequestCanceledParams struct {
	ID                  int64
	ErrorCode           string
	ErrorMessage        string
	InternalErrorDetail string
	CompletedAt         time.Time
}

// CreateAttemptParams 表示创建 request attempt 所需的上游尝试事实。
type CreateAttemptParams struct {
	RequestRecordID int64
	AttemptIndex    int
	ProviderID      int64
	ChannelID       int64
	AdapterKey      string
	UpstreamModel   string
	StartedAt       time.Time
}

// AttemptRecord 表示一次上游 channel 尝试记录。
type AttemptRecord struct {
	ID                    int64
	RequestRecordID       int64
	AttemptIndex          int
	ProviderID            int64
	ChannelID             int64
	AdapterKey            string
	UpstreamModel         string
	UpstreamResponseModel *string
	Status                AttemptStatus
	UpstreamStatusCode    *int
	UpstreamRequestID     *string
	ErrorCode             *string
	ErrorMessage          *string
	InternalErrorDetail   *string
	StartedAt             time.Time
	CompletedAt           *time.Time
}

// MarkAttemptSucceededParams 表示标记上游尝试成功所需的最终事实。
type MarkAttemptSucceededParams struct {
	ID                    int64
	UpstreamResponseModel string
	UpstreamStatusCode    int
	UpstreamRequestID     *string
	CompletedAt           time.Time
}

// MarkAttemptFailedParams 表示标记上游尝试失败所需的错误事实。
type MarkAttemptFailedParams struct {
	ID                  int64
	UpstreamStatusCode  *int
	UpstreamRequestID   *string
	ErrorCode           string
	ErrorMessage        string
	InternalErrorDetail string
	CompletedAt         time.Time
}

// MarkAttemptCanceledParams 表示标记上游尝试取消所需的错误事实。
type MarkAttemptCanceledParams struct {
	ID                  int64
	ErrorCode           string
	ErrorMessage        string
	InternalErrorDetail string
	CompletedAt         time.Time
}

// Service 定义 request log 写入能力。
// 它只负责请求与上游尝试的审计状态，不负责 usage、price snapshot 或 ledger 扣费。
type Service interface {
	CreateRequest(ctx context.Context, params CreateRequestParams) (RequestRecord, error)
	MarkRequestRunning(ctx context.Context, id int64) (RequestRecord, error)
	MarkRequestSucceeded(ctx context.Context, params MarkRequestSucceededParams) (RequestRecord, error)
	MarkRequestFailed(ctx context.Context, params MarkRequestFailedParams) (RequestRecord, error)
	MarkRequestCanceled(ctx context.Context, params MarkRequestCanceledParams) (RequestRecord, error)

	CreateAttempt(ctx context.Context, params CreateAttemptParams) (AttemptRecord, error)
	MarkAttemptSucceeded(ctx context.Context, params MarkAttemptSucceededParams) (AttemptRecord, error)
	MarkAttemptFailed(ctx context.Context, params MarkAttemptFailedParams) (AttemptRecord, error)
	MarkAttemptCanceled(ctx context.Context, params MarkAttemptCanceledParams) (AttemptRecord, error)
}
