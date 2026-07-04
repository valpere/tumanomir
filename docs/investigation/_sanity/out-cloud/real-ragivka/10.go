package ragivka

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// ─── Constants & Named Types ──────────────────────────────────────────

type DeploymentMode string

const (
	DeploymentLocal        DeploymentMode = "local"
	DeploymentDockerCompose DeploymentMode = "docker-compose"
	DeploymentScaled       DeploymentMode = "scaled"
	DeploymentOffline      DeploymentMode = "offline"
)

type Tier string

const (
	TierL0 Tier = "L0"
	TierL1 Tier = "L1"
	TierL2 Tier = "L2"
	TierL3 Tier = "L3"
)

type FSMState string

const (
	FSMStateActive         FSMState = "Active"
	FSMStateWaitingForHuman FSMState = "WaitingForHuman"
	FSMStateCompleted       FSMState = "Completed"
	FSMStateExpired         FSMState = "Expired"
)

type ToolPermission string

const (
	ToolPermissionRead  ToolPermission = "Read"
	ToolPermissionDraft ToolPermission = "Draft"
	ToolPermissionWrite ToolPermission = "Write"
)

type LLMProvider string

const (
	LLMProviderOpenAI    LLMProvider = "openai"
	LLMProviderAnthropic LLMProvider = "anthropic"
	LLMProviderOpenRouter LLMProvider = "openrouter"
	LLMProviderGemini    LLMProvider = "gemini"
	LLMProviderOllama    LLMProvider = "ollama"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusRunning    JobStatus = "running"
	JobStatusRetrying   JobStatus = "retrying"
	JobStatusCompleted  JobStatus = "completed"
	JobStatusFailed     JobStatus = "failed"
	JobStatusDeadLetter JobStatus = "dead_letter"
)

type ChannelKind string

const (
	ChannelKindTelegram  ChannelKind = "telegram"
	ChannelKindWebWidget ChannelKind = "web-widget"
)

type ErrorCode string

type MessageRole string

const (
	MessageRoleUser      MessageRole = "user"
	MessageRoleAssistant MessageRole = "assistant"
	MessageRoleTool      MessageRole = "tool"
	MessageRoleSystem    MessageRole = "system"
)

type ConnectorType string

// ─── Domain Primitives ─────────────────────────────────────────────────

type TenantID uuid.UUID
type SessionID uuid.UUID
type DocumentID uuid.UUID
type ChunkID uuid.UUID
type ArtifactID uuid.UUID
type JobID uuid.UUID
type PromptVersion int64
type OperationKey string
type TraceID string

// ─── Configuration ──────────────────────────────────────────────────────

type Config struct {
	DeploymentMode      DeploymentMode
	APIListenAddr       string
	DatabaseURL         string
	RedisAddr           string
	ObjectStorageConfig ObjectStorageConfig
	LLMProviders        map[LLMProvider]ProviderConfig
	RateLimits          map[TenantID]RateLimitConfig
	PITRSettings        PITRSettings
	OfflineSettings     OfflineSettings
	Tracing             TracingConfig
}

type ObjectStorageConfig struct {
	Endpoint  string
	Bucket    string
	Region    string
	AccessKey string
	SecretKey string
	UseSSL    bool
}

type ProviderConfig struct {
	APIKey      string
	BaseURL     string
	Model       string
	MaxRetries  int
	Timeout     time.Duration
	Enabled     bool
	OfflineOnly bool
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
	Window            time.Duration
	Algorithm         string
}

type PITRSettings struct {
	Enabled               bool
	ArchiveDestination    string
	RetentionWindow       time.Duration
	RecoveryPointObjective time.Duration
	RecoveryTimeObjective time.Duration
}

type OfflineSettings struct {
	OllamaURL          string
	EmbeddingModel     string
	EmbeddingDimension int
}

type TracingConfig struct {
	Enabled      bool
	ExporterURL  string
	SampleRate   float64
	ServiceName  string
}

// ─── Multi-Tenancy & Auth ─────────────────────────────────────────────

type Tenant struct {
	ID        TenantID
	Name      string
	CreatedAt time.Time
	Settings  TenantSettings
}

type TenantSettings struct {
	DefaultTier       Tier
	RateLimitConfig   RateLimitConfig
	AllowedProviders  []LLMProvider
	BudgetLimit       *float64
	MaxContextTurns   int
	InactivityTimeout time.Duration
}

type AuthContext struct {
	TenantID  TenantID
	UserID    *string
	APIKeyID  *uuid.UUID
	Scopes    []string
	JWTExpiry time.Time
}

type Authenticator interface {
	Authenticate(ctx context.Context, r *http.Request) (AuthContext, error)
}

// ─── Session / FSM ──────────────────────────────────────────────────────

type Session struct {
	ID            SessionID
	TenantID      TenantID
	State         FSMState
	Version       int64
	ContextWindow []Message
	Metadata      map[string]any
	LastActivity  time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type Message struct {
	ID        uuid.UUID
	SessionID SessionID
	Role      MessageRole
	Content   string
	ToolCalls []ToolCall
	Turn      int
	CreatedAt time.Time
}

type StateTransition struct {
	SessionID     SessionID
	From          FSMState
	To            FSMState
	TriggeredBy   string
	Reason        string
	IdempotencyKey OperationKey
	OccurredAt    time.Time
}

type SessionManager interface {
	CreateSession(ctx context.Context, tenantID TenantID) (Session, error)
	AppendMessage(ctx context.Context, sessionID SessionID, msg Message) (Session, error)
	Transition(ctx context.Context, sessionID SessionID, to FSMState, reason string) (Session, error)
	GetSession(ctx context.Context, sessionID SessionID) (Session, error)
	ExpireInactive(ctx context.Context, before time.Time) ([]SessionID, error)
	EnforceHistoryLimit(ctx context.Context, sessionID SessionID, limit int) error
}

// ─── Orchestration Tiers ────────────────────────────────────────────────

type Orchestrator interface {
	Execute(ctx context.Context, req ExecutionRequest) (ExecutionResult, error)
	Enqueue(ctx context.Context, req ExecutionRequest) (JobID, error)
}

type ExecutionRequest struct {
	Tier       Tier
	SessionID  SessionID
	Auth       AuthContext
	Input      string
	Metadata   map[string]any
	Timeout    *time.Duration
	Callbacks  []CallbackSpec
}

type ExecutionResult struct {
	SessionID      SessionID
	State          FSMState
	Output         string
	ToolCalls        []ToolCall
	Artifacts        []ArtifactID
	TraceID          TraceID
	CostUSD          float64
	TokenUsage       TokenUsage
	Citations        []Citation
	ErrorCode        *ErrorCode
	BlockingHITL     bool
}

type CallbackSpec struct {
	URL     string
	Events  []string
	Headers map[string]string
}

type DAGSpec struct {
	Nodes       []DAGNode
	Edges       []DAGEdge
	GlobalTimeout time.Duration
}

type DAGNode struct {
	ID            string
	Kind          string
	AgentID       string
	Timeout       time.Duration
	Dependencies  []string
	MaxRetries    int
}

type DAGEdge struct {
	From string
	To   string
}

type AgentPool interface {
	InvokeNode(ctx context.Context, node DAGNode, inputs map[string]any) (map[string]any, error)
	DetectDeadlock(ctx context.Context, dag DAGSpec) error
}

// ─── Knowledge / RAG ────────────────────────────────────────────────────

type Document struct {
	ID          DocumentID
	TenantID    TenantID
	Connector   ConnectorType
	SourceURI   string
	RawObjectKey string
	Status      DocumentStatus
	Version     int64
	Metadata    map[string]any
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type DocumentStatus string

const (
	DocumentStatusPending   DocumentStatus = "pending"
	DocumentStatusParsed    DocumentStatus = "parsed"
	DocumentStatusChunked   DocumentStatus = "chunked"
	DocumentStatusEmbedded  DocumentStatus = "embedded"
	DocumentStatusStale     DocumentStatus = "stale"
	DocumentStatusFailed    DocumentStatus = "failed"
)

type Chunk struct {
	ID           ChunkID
	TenantID     TenantID
	DocumentID   DocumentID
	Ordinal      int
	Text         string
	Embedding    []float32
	SourceLocation string
	Metadata     map[string]any
	CreatedAt    time.Time
}

type SearchRequest struct {
	TenantID    TenantID
	Query       string
	TopK        int
	RerankTopK  int
	UseHybrid   bool
	Filters     map[string]any
	Timeout     time.Duration
}

type SearchResult struct {
	Chunk      Chunk
	Similarity float64
	KeywordRank float64
	RerankScore float64
}

type Citation struct {
	DocumentName string
	ChunkOrdinal int
	SourceURI    string
	Excerpt      string
}

type Retriever interface {
	Search(ctx context.Context, req SearchRequest) ([]SearchResult, error)
	Index(ctx context.Context, chunks []Chunk) error
	DeleteStale(ctx context.Context, tenantID TenantID, documentID DocumentID, keep []ChunkID) error
}

type ChunkingConfig struct {
	TargetTokens int
	OverlapPct   float64
	Separator    string
}

type Chunker interface {
	Chunk(ctx context.Context, doc Document, text string, cfg ChunkingConfig) ([]Chunk, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, results []SearchResult, topK int) ([]SearchResult, error)
}

// ─── Ingestion Pipeline ─────────────────────────────────────────────────

type IngestionRequest struct {
	TenantID    TenantID
	Connector   ConnectorType
	SourceURI   string
	RawBytes    io.Reader
	Force       bool
	Metadata    map[string]any
}

type IngestionJob struct {
	ID            JobID
	Request       IngestionRequest
	Stage         string
	Version       int64
	PIIStripped   bool
	CreatedAt     time.Time
}

type Connector interface {
	Fetch(ctx context.Context, sourceURI string) ([]byte, map[string]any, error)
}

type Parser interface {
	Parse(ctx context.Context, contentType string, data []byte) (string, map[string]any, error)
}

type IngestionPipeline interface {
	Ingest(ctx context.Context, req IngestionRequest) (DocumentID, error)
	Reingest(ctx context.Context, documentID DocumentID) error
}

// ─── AI Layer ───────────────────────────────────────────────────────────

type Prompt struct {
	Name       string
	Version    PromptVersion
	Template   string
	Variables  []string
	Metadata   map[string]any
	CreatedAt  time.Time
}

type PromptRegistry interface {
	Load(ctx context.Context, name string, version *PromptVersion) (Prompt, error)
	Render(ctx context.Context, prompt Prompt, variables map[string]any) (string, error)
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	CostUSD          float64
	Provider         LLMProvider
	Model            string
}

type LLMRequest struct {
	Provider    LLMProvider
	Model       string
	Messages    []Message
	Temperature float64
	MaxTokens   int
	JSONSchema  *json.RawMessage
	Timeout     time.Duration
}

type LLMResponse struct {
	Content    string
	Structured json.RawMessage
	Usage      TokenUsage
	Provider   LLMProvider
	Model      string
}

type ModelRouter interface {
	Route(ctx context.Context, task TaskSpec) (LLMProvider, string, error)
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
	Fallback(ctx context.Context, req LLMRequest, err error) (LLMResponse, error)
}

type TaskSpec struct {
	Tier         Tier
	TaskType     string
	Complexity   int
	CostPolicy   string
	RequiredJSON bool
}

// ─── Tool Layer ─────────────────────────────────────────────────────────

type Tool interface {
	Name() string
	Permission() ToolPermission
	Schema() ToolSchema
	Execute(ctx context.Context, args json.RawMessage, ctx ToolContext) (json.RawMessage, error)
}

type ToolSchema struct {
	Name        string
	Description string
	InputSchema  json.RawMessage
	OutputSchema json.RawMessage
	Permission  ToolPermission
}

type ToolContext struct {
	SessionID      SessionID
	TenantID       TenantID
	Auth           AuthContext
	IdempotencyKey OperationKey
	TraceID        TraceID
}

type ToolCall struct {
	ID           string
	ToolName     string
	Arguments    json.RawMessage
	Result       json.RawMessage
	Error        *ToolError
	ExecutedAt   time.Time
}

type ToolError struct {
	Code    string
	Message string
	Retryable bool
}

type ToolRegistry interface {
	Register(tool Tool) error
	Get(name string) (Tool, bool)
	List(permission *ToolPermission) []ToolSchema
	Execute(ctx context.Context, call ToolCall, tc ToolContext) (ToolCall, error)
}

type HITLGate interface {
	RequireApproval(ctx context.Context, sessionID SessionID, toolCall ToolCall, reason string) error
	Escalate(ctx context.Context, sessionID SessionID, reason string) error
}

// ─── Artifact Generation ──────────────────────────────────────────────

type Artifact struct {
	ID          ArtifactID
	TenantID    TenantID
	SessionID   SessionID
	Kind        string
	ObjectKey   string
	SizeBytes   int64
	ContentHash string
	Metadata    map[string]any
	CreatedAt   time.Time
}

type ArtifactGenerator interface {
	GeneratePDF(ctx context.Context, sessionID SessionID, data json.RawMessage) (ArtifactID, error)
	GenerateExcel(ctx context.Context, sessionID SessionID, data json.RawMessage) (ArtifactID, error)
}

// ─── Jobs ───────────────────────────────────────────────────────────────

type JobWorker interface {
	Enqueue(ctx context.Context, kind string, payload []byte, opts JobOptions) (JobID, error)
	Claim(ctx context.Context, kinds []string) (Job, error)
	Complete(ctx context.Context, jobID JobID) error
	Fail(ctx context.Context, jobID JobID, err error, retry bool) error
	DeadLetter(ctx context.Context, jobID JobID, err error) error
}

type Job struct {
	ID        JobID
	Kind      string
	Payload   []byte
	Attempt   int
	Status    JobStatus
	CreatedAt time.Time
}

type JobOptions struct {
	MaxRetries     int
	BackoffInitial time.Duration
	BackoffMax     time.Duration
	Queue          string
	Priority       int
	Delay          time.Duration
}

// ─── Channel Adapters ───────────────────────────────────────────────────

type ChannelAdapter interface {
	Kind() ChannelKind
	Receive(ctx context.Context, payload []byte) (ChannelRequest, error)
	Send(ctx context.Context, resp ChannelResponse) error
}

type ChannelRequest struct {
	ChannelID   string
	UserID      string
	TenantID    TenantID
	MessageText string
	Metadata    map[string]any
}

type ChannelResponse struct {
	ChannelID     string
	Text          string
	FormattedHTML string
	Keyboard      [][]KeyboardButton
	Artifacts     []ArtifactID
}

type KeyboardButton struct {
	Text         string
	CallbackData string
	URL          string
}

type TelegramAdapter struct{}
type WebWidgetAdapter struct{}

// ─── Observability ──────────────────────────────────────────────────────

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
	InjectHTTP(ctx context.Context, r *http.Request)
}

type Span interface {
	End()
	SetError(err error)
	SetAttribute(key string, value any)
}

type MetricsCollector interface {
	RecordLLMTokens(ctx context.Context, usage TokenUsage)
	RecordRetrievalLatency(ctx context.Context, tenantID TenantID, d time.Duration)
	RecordQueueDepth(ctx context.Context, queue string, depth int)
	RecordError(ctx context.Context, code string)
}

type CostTracker interface {
	Log(ctx context.Context, tenantID TenantID, usage TokenUsage) error
	EnforceBudget(ctx context.Context, tenantID TenantID) error
}

type QualityGate interface {
	RecordRecallAtK(ctx context.Context, k int, score float64)
	RecordCitationCoverage(ctx context.Context, score float64)
	GroundednessHook(ctx context.Context, answer string, citations []Citation) (float64, error)
}

// ─── Audit & Rate Limiting ──────────────────────────────────────────────

type AuditLogEntry struct {
	ID             uuid.UUID
	TenantID       TenantID
	SessionID      *SessionID
	IdempotencyKey OperationKey
	ToolName       string
	RequestHash    string
	ResponseHash   string
	Action         string
	Actor          string
	OccurredAt     time.Time
}

type Auditor interface {
	Log(ctx context.Context, entry AuditLogEntry) error
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID) (bool, time.Duration, error)
	AllowWithKey(ctx context.Context, key string, cfg RateLimitConfig) (bool, time.Duration, error)
}

// ─── API / REST ─────────────────────────────────────────────────────────

type APIError struct {
	Code       ErrorCode   `json:"code"`
	Message    string      `json:"message"`
	Details    map[string]any `json:"details,omitempty"`
	RequestID  TraceID     `json:"request_id,omitempty"`
	StatusCode int         `json:"-"`
}

type RESTHandler interface {
	HandleError(w http.ResponseWriter, err error)
}

type WebsocketConn interface {
	WriteJSON(v any) error
	ReadJSON(v any) error
	Close() error
}

type WebWidgetServer interface {
	HandleChat(w http.ResponseWriter, r *http.Request)
	HandleWebSocket(ctx context.Context, conn WebsocketConn, tenantID TenantID) error
}

// ─── Security & Privacy ───────────────────────────────────────────────

type InputValidator interface {
	ValidatePromptInput(ctx context.Context, input string) error
	ValidateToolArguments(ctx context.Context, schema ToolSchema, args json.RawMessage) error
}

type PIISanitizer interface {
	Strip(ctx context.Context, text string) (string, map[string]any, error)
}

// ─── Service Functions (no bodies) ──────────────────────────────────────

func NewConfig() Config                                                                             {}
func ValidateConfig(cfg Config) error                                                               {}
func NewSessionManager(db *sql.DB, lockTimeout time.Duration) SessionManager                        {}
func NewOrchestrator(api Orchestrator, worker JobWorker) Orchestrator                             {}
func NewRetriever(pool *pgxpool.Pool) Retriever                                                      {}
func NewChunker(cfg ChunkingConfig) Chunker                                                         {}
func NewReranker(provider LLMProvider, model string) Reranker                                       {}
func NewIngestionPipeline(connectors map[ConnectorType]Connector, parsers map[string]Parser, chunker Chunker, retriever Retriever, sanitizer PIISanitizer) IngestionPipeline {}
func NewModelRouter(providers map[LLMProvider]ProviderConfig) ModelRouter                           {}
func NewPromptRegistry(db *sql.DB) PromptRegistry                                                 {}
func NewToolRegistry() ToolRegistry                                                               {}
func NewArtifactGenerator(store ObjectStore) ArtifactGenerator                                      {}
func NewJobWorker(pool *pgxpool.Pool, redisAddr string) JobWorker                                   {}
func NewTelegramAdapter(token string) ChannelAdapter                                              {}
func NewWebWidgetAdapter(auth Authenticator) ChannelAdapter                                       {}
func NewRateLimiter(redisAddr string) RateLimiter                                                   {}
func NewAuditor(db *sql.DB) Auditor                                                               {}
func NewTracer(cfg TracingConfig) Tracer                                                          {}
func NewMetricsCollector() MetricsCollector                                                       {}
func NewCostTracker(db *sql.DB) CostTracker                                                       {}
func NewQualityGate() QualityGate                                                                 {}
func NewInputValidator() InputValidator                                                           {}
func NewPIISanitizer() PIISanitizer                                                               {}

// ─── Package Aliases / External Types (stub imports for compilation) ────

type pgxpool struct{}
