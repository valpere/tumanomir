package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/trace"
)

type TenantID string
type SessionID string
type UserID string
type MessageID string
type DocumentID string
type ChunkID string
type ArtifactID string
type JobID string
type ToolName string
type PromptName string
type Version string

type FSMState string
type OrchestrationTier string
type ChannelType string
type IngestionStatus string
type JobStatus string
type ToolPermission string
type AuditAction string

const (
	FSMStateActive          FSMState = "active"
	FSMStateWaitingForHuman FSMState = "waiting_for_human"
	FSMStateCompleted       FSMState = "completed"
	FSMStateExpired         FSMState = "expired"

	OrchestrationTierL0 OrchestrationTier = "l0"
	OrchestrationTierL1 OrchestrationTier = "l1"
	OrchestrationTierL2 OrchestrationTier = "l2"
	OrchestrationTierL3 OrchestrationTier = "l3"

	ChannelTypeTelegram ChannelType = "telegram"
	ChannelTypeWeb      ChannelType = "web"

	IngestionStatusPending   IngestionStatus = "pending"
	IngestionStatusIndexed   IngestionStatus = "indexed"
	IngestionStatusStale     IngestionStatus = "stale"
	IngestionStatusFailed    IngestionStatus = "failed"

	ToolPermissionRead  ToolPermission = "read"
	ToolPermissionDraft ToolPermission = "draft"
	ToolPermissionWrite ToolPermission = "write"

	AuditActionCreate AuditAction = "create"
	AuditActionUpdate AuditAction = "update"
	AuditActionDelete AuditAction = "delete"
)

type Tenant struct {
	ID        TenantID    `db:"id"`
	CreatedAt time.Time   `db:"created_at"`
	UpdatedAt time.Time   `db:"updated_at"`
	Name      string      `db:"name"`
	IsActive  bool        `db:"is_active"`
	Settings  *TenantSettings `db:"settings"`
}

type TenantSettings struct {
	RateLimitRequestsPerMinute int    `json:"rate_limit_requests_per_minute"`
	AllowedLanguages           []string `json:"allowed_languages"`
}

type User struct {
	ID         UserID      `db:"id"`
	TenantID   TenantID    `db:"tenant_id"`
	CreatedAt  time.Time   `db:"created_at"`
	UpdatedAt  time.Time   `db:"updated_at"`
	ChannelType ChannelType `db:"channel_type"`
	ChannelID  string      `db:"channel_id"`
	IsActive   bool        `db:"is_active"`
}

type Session struct {
	ID                 SessionID           `db:"id"`
	TenantID           TenantID            `db:"tenant_id"`
	UserID             UserID              `db:"user_id"`
	State              FSMState            `db:"state"`
	Version            int                 `db:"version"`
	OrchestrationTier  OrchestrationTier   `db:"orchestration_tier"`
	Channel            ChannelType         `db:"channel"`
	ExpiresAt          time.Time           `db:"expires_at"`
	CreatedAt          time.Time           `db:"created_at"`
	UpdatedAt          time.Time           `db:"updated_at"`
	InactivityTimeout  time.Duration       `db:"inactivity_timeout"`
	ContextWindowLimit int                 `db:"context_window_limit"`
}

type Message struct {
	ID              MessageID   `db:"id"`
	SessionID       SessionID   `db:"session_id"`
	TenantID        TenantID    `db:"tenant_id"`
	Role            string      `db:"role"`
	Content         string      `db:"content"`
	CitationRefs    []string    `db:"citation_refs"`
	TokenCount      int         `db:"token_count"`
	CreatedAt       time.Time   `db:"created_at"`
	JobID           *JobID      `db:"job_id"`
	IsSystemMessage bool        `db:"is_system_message"`
}

type Document struct {
	ID                DocumentID    `db:"id"`
	TenantID          TenantID      `db:"tenant_id"`
	CreatedAt         time.Time     `db:"created_at"`
	UpdatedAt         time.Time     `db:"updated_at"`
	Name              string        `db:"name"`
	S3Key             string        `db:"s3_key"`
	Version           string        `db:"version"`
	IngestionStatus   IngestionStatus `db:"ingestion_status"`
	Size              int64         `db:"size"`
	ContentType       string        `db:"content_type"`
	IsPublic          bool          `db:"is_public"`
	LastProcessedAt   *time.Time    `db:"last_processed_at"`
}

type Chunk struct {
	ID             ChunkID   `db:"id"`
	DocumentID     DocumentID `db:"document_id"`
	TenantID       TenantID  `db:"tenant_id"`
	CreatedAt      time.Time `db:"created_at"`
	Content        string    `db:"content"`
	Vector         []float32 `db:"vector"`
	TsVector       string    `db:"ts_vector"`
	Ordinal        int       `db:"ordinal"`
	SourceLocation string    `db:"source_location"`
	Metadata       string    `db:"metadata"`
}

type Artifact struct {
	ID          ArtifactID   `db:"id"`
	SessionID   SessionID    `db:"session_id"`
	TenantID    TenantID     `db:"tenant_id"`
	CreatedAt   time.Time    `db:"created_at"`
	S3Key       string       `db:"s3_key"`
	Type        string       `db:"type"`
	Name        string       `db:"name"`
	Size        int64        `db:"size"`
	Description string       `db:"description"`
}

type PromptVersion struct {
	Name      PromptName  `db:"name"`
	Version   Version     `db:"version"`
	Content   string      `db:"content"`
	CreatedAt time.Time   `db:"created_at"`
	IsCurrent bool        `db:"is_current"`
}

type RiverJob struct {
	ID                JobID         `db:"id"`
	TenantID          TenantID      `db:"tenant_id"`
	SessionID         SessionID     `db:"session_id"`
	IdempotencyKey    string        `db:"idempotency_key"`
	Payload           []byte        `db:"payload"`
	Attempt           int           `db:"attempt"`
	MaxAttempts       int           `db:"max_attempts"`
	Status            JobStatus     `db:"status"`
	LastError         *string       `db:"last_error"`
	CreatedAt         time.Time     `db:"created_at"`
	UpdatedAt         time.Time     `db:"updated_at"`
	ScheduledAt       time.Time     `db:"scheduled_at"`
	StartedAt         *time.Time    `db:"started_at"`
	CompletedAt       *time.Time    `db:"completed_at"`
	BackoffStrategy   string        `db:"backoff_strategy"`
}

type AuditLog struct {
	ID               string           `db:"id"`
	TenantID         TenantID         `db:"tenant_id"`
	SessionID        SessionID        `db:"session_id"`
	UserID           UserID           `db:"user_id"`
	ToolName         ToolName         `db:"tool_name"`
	IdempotencyKey   string           `db:"idempotency_key"`
	Action           AuditAction      `db:"action"`
	RequestHash      string           `db:"request_hash"`
	ResponseHash     string           `db:"response_hash"`
	CreatedAt        time.Time        `db:"created_at"`
	ApprovalRecord   *ApprovalRecord  `db:"approval_record"`
}

type ApprovalRecord struct {
	ApprovedBy    UserID     `json:"approved_by"`
	ApprovedAt    time.Time  `json:"approved_at"`
	Notes         string     `json:"notes"`
	IsExplicit    bool       `json:"is_explicit"`
}

type ToolRegistry struct {
	ReadTools   map[ToolName]ReadTool   `db:"read_tools"`
	DraftTools  map[ToolName]DraftTool  `db:"draft_tools"`
	WriteTools  map[ToolName]WriteTool  `db:"write_tools"`
}

type ReadTool struct {
	Name        ToolName      `json:"name"`
	Permission  ToolPermission `json:"permission"`
	Description string        `json:"description"`
	CacheTTL    *time.Duration `json:"cache_ttl,omitempty"`
}

type DraftTool struct {
	Name        ToolName      `json:"name"`
	Permission  ToolPermission `json:"permission"`
	Description string        `json:"description"`
}

type WriteTool struct {
	Name              ToolName       `json:"name"`
	Permission        ToolPermission `json:"permission"`
	Description       string         `json:"description"`
	RequiresApproval  bool           `json:"requires_approval"`
	IdempotencyKey    string         `json:"idempotency_key"`
}

type ModelRouter interface {
	Route(ctx context.Context, request *ModelRequest) (*ModelResponse, error)
	GetAvailableModels() []string
	SetFallbackProvider(provider string)
}

type ModelRequest struct {
	Content        string            `json:"content"`
	Tools          []ToolName        `json:"tools,omitempty"`
	Prompt         string            `json:"prompt"`
	MaxTokens      int               `json:"max_tokens"`
	Temperature    float32           `json:"temperature"`
	Structured     bool              `json:"structured"`
	ResponseFormat   *ResponseFormat  `json:"response_format,omitempty"`
}

type ModelResponse struct {
	Content       string         `json:"content"`
	Citations     []Citation     `json:"citations"`
	TokenUsage    TokenUsage     `json:"token_usage"`
	ToolCalls     []ToolCall     `json:"tool_calls,omitempty"`
	ModelName     string         `json:"model_name"`
	GeneratedAt   time.Time      `json:"generated_at"`
}

type Citation struct {
	DocumentID DocumentID `json:"document_id"`
	ChunkID    ChunkID    `json:"chunk_id"`
	Ordinal    int        `json:"ordinal"`
	Content    string     `json:"content"`
}

type TokenUsage struct {
	PromptTokens   int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens    int `json:"total_tokens"`
	CostUSD        float64 `json:"cost_usd"`
}

type ToolCall struct {
	Name      ToolName  `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

type ResponseFormat struct {
	Type   string `json:"type"`
	Schema any    `json:"schema,omitempty"`
}

type RAGPipeline interface {
	Retrieve(ctx context.Context, query string, tenantID TenantID, sessionID SessionID) (*RAGResponse, error)
	ReRank(ctx context.Context, chunks []Chunk, query string) ([]Chunk, error)
	GenerateCitations(ctx context.Context, content string, chunks []Chunk) ([]Citation, error)
}

type RAGResponse struct {
	Chunks     []Chunk      `json:"chunks"`
	Citations  []Citation   `json:"citations"`
	Context    string       `json:"context"`
	ReRankTime time.Duration `json:"re_rank_time"`
}

type ChannelAdapter interface {
	HandleMessage(ctx context.Context, message *ChannelMessage) error
	SendMessage(ctx context.Context, sessionID SessionID, content string, options *SendMessageOptions) error
	GetSessionState(ctx context.Context, sessionID SessionID) (*FSMState, error)
}

type ChannelMessage struct {
	ID           string      `json:"id"`
	SessionID    SessionID   `json:"session_id"`
	Content      string      `json:"content"`
	Sender       UserID      `json:"sender"`
	TenantID     TenantID    `json:"tenant_id"`
	CreatedAt    time.Time   `json:"created_at"`
	ChannelType  ChannelType `json:"channel_type"`
	IsSystem     bool        `json:"is_system"`
}

type SendMessageOptions struct {
	ReplyTo      *MessageID    `json:"reply_to,omitempty"`
	InlineKeyboard []KeyboardButton `json:"inline_keyboard,omitempty"`
	Format       string        `json:"format,omitempty"`
}

type KeyboardButton struct {
	Text  string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL   string `json:"url,omitempty"`
}

type Worker struct {
	Pool *pgxpool.Pool
	RiverClient *river.Client
	Tracer trace.Tracer
	ModelRouter ModelRouter
	RAGPipeline RAGPipeline
	ToolRegistry ToolRegistry
}

func (w *Worker) ProcessJob(ctx context.Context, job *RiverJob) error { return nil }

type APIHandler struct {
	Tracer trace.Tracer
	ToolRegistry ToolRegistry
	SessionManager SessionManager
}

func (h *APIHandler) HandleMessage(ctx context.Context, message *ChannelMessage) (*Message, error) { return nil, nil }
func (h *APIHandler) CreateSession(ctx context.Context, userID UserID, channel ChannelType) (*Session, error) { return nil, nil }
func (h *APIHandler) GetSession(ctx context.Context, sessionID SessionID) (*Session, error) { return nil, nil }

type SessionManager interface {
	GetOrCreateSession(ctx context.Context, userID UserID, channel ChannelType) (*Session, error)
	UpdateSessionState(ctx context.Context, sessionID SessionID, state FSMState, version int) error
	ExtendSessionTimeout(ctx context.Context, sessionID SessionID, timeout time.Duration) error
	GetSessionWithLock(ctx context.Context, sessionID SessionID) (*Session, error)
}

type Guardrails interface {
	EvaluateOutput(ctx context.Context, content string, citations []Citation, chunks []Chunk) (*EvaluationResult, error)
	ValidatePromptInjection(ctx context.Context, input string) error
}

type EvaluationResult struct {
	IsGrounded     bool           `json:"is_grounded"`
	CitationCoverage float64     `json:"citation_coverage"`
	RetrievalRecall  float64     `json:"retrieval_recall"`
	Confidence     float64        `json:"confidence"`
	Flags          []string       `json:"flags"`
}

type MetricsCollector interface {
	RecordTokenUsage(ctx context.Context, tenantID TenantID, usage TokenUsage) error
	RecordQueueDepth(ctx context.Context, depth int) error
	RecordLatency(ctx context.Context, duration time.Duration, operation string) error
	RecordError(ctx context.Context, err error, operation string) error
}

type EventLogger interface {
	LogAuditEvent(ctx context.Context, event *AuditLog) error
	LogTrace(ctx context.Context, traceID string, operation string, metadata map[string]interface{}) error
}
