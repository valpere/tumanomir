package ragivka

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
)

type TenantID string
type UserID string
type SessionID string
type MessageID string
type DocumentID string
type ChunkID string
type ArtifactID string
type JobID string
type PromptName string
type PromptVersion string
type ToolName string
type IdempotencyKey string
type TraceID string

type OrchestrationTier string

const (
	TierL0 OrchestrationTier = "L0"
	TierL1 OrchestrationTier = "L1"
	TierL2 OrchestrationTier = "L2"
	TierL3 OrchestrationTier = "L3"
)

type SessionState string

const (
	StateActive          SessionState = "Active"
	StateWaitingForHuman SessionState = "WaitingForHuman"
	StateCompleted       SessionState = "Completed"
	StateExpired         SessionState = "Expired"
)

type ToolPermission string

const (
	PermissionRead  ToolPermission = "Read"
	PermissionDraft ToolPermission = "Draft"
	PermissionWrite ToolPermission = "Write"
)

type IngestionStatus string

const (
	StatusPending  IngestionStatus = "pending"
	StatusIndexed  IngestionStatus = "indexed"
	StatusStale    IngestionStatus = "stale"
	StatusFailed   IngestionStatus = "failed"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
)

type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelWeb      ChannelType = "web"
)

type ArtifactType string

const (
	ArtifactPDF   ArtifactType = "pdf"
	ArtifactExcel ArtifactType = "excel"
	ArtifactHTML  ArtifactType = "html"
)

type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderOpenRouter LLMProvider = "openrouter"
	ProviderGemini    LLMProvider = "gemini"
	ProviderOllama    LLMProvider = "ollama"
)

type EmbeddingProvider string

const (
	EmbeddingOpenAI   EmbeddingProvider = "openai"
	EmbeddingCohere   EmbeddingProvider = "cohere"
	EmbeddingBGE      EmbeddingProvider = "bge"
	EmbeddingOllama   EmbeddingProvider = "ollama"
)

type Tenant struct {
	ID        TenantID  `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Config    TenantConfig `json:"config"`
}

type TenantConfig struct {
	RateLimitRPM     int              `json:"rate_limit_rpm"`
	RateLimitBurst   int              `json:"rate_limit_burst"`
	DefaultTier      OrchestrationTier `json:"default_tier"`
	MaxSessionTurns  int              `json:"max_session_turns"`
	SessionTimeout   time.Duration    `json:"session_timeout"`
	AllowedProviders []LLMProvider    `json:"allowed_providers"`
}

type User struct {
	ID        UserID    `json:"id"`
	TenantID  TenantID  `json:"tenant_id"`
	Channel   ChannelType `json:"channel"`
	ChannelID string    `json:"channel_id"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID               SessionID         `json:"id"`
	TenantID         TenantID          `json:"tenant_id"`
	UserID           UserID            `json:"user_id"`
	State            SessionState      `json:"state"`
	Version          int64             `json:"version"`
	OrchestrationTier OrchestrationTier `json:"orchestration_tier"`
	Channel          ChannelType       `json:"channel"`
	ContextSummary   string            `json:"context_summary,omitempty"`
	ExpiresAt        time.Time         `json:"expires_at"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type Message struct {
	ID            MessageID       `json:"id"`
	TenantID      TenantID        `json:"tenant_id"`
	SessionID     SessionID       `json:"session_id"`
	Role          MessageRole     `json:"role"`
	Content       string          `json:"content"`
	CitationRefs  []ChunkID       `json:"citation_refs,omitempty"`
	TokenCount    int             `json:"token_count"`
	ToolCalls     []ToolCallRecord `json:"tool_calls,omitempty"`
	JobID         *JobID          `json:"job_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

type ToolCallRecord struct {
	ToolName ToolName        `json:"tool_name"`
	Input    json.RawMessage `json:"input"`
	Output   json.RawMessage `json:"output,omitempty"`
}

type Document struct {
	ID              DocumentID      `json:"id"`
	TenantID        TenantID        `json:"tenant_id"`
	Name            string          `json:"name"`
	ObjectKey       string          `json:"object_key"`
	Version         int             `json:"version"`
	IngestionStatus IngestionStatus `json:"ingestion_status"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type Chunk struct {
	ID         ChunkID         `json:"id"`
	TenantID   TenantID        `json:"tenant_id"`
	DocumentID DocumentID      `json:"document_id"`
	Ordinal    int             `json:"ordinal"`
	Content    string          `json:"content"`
	Vector     pgtype.Vector   `json:"-"`
	TSVector   string          `json:"-"`
	SourceLocation string      `json:"source_location,omitempty"`
	Metadata   json.RawMessage `json:"metadata,omitempty"`
	CreatedAt  time.Time       `json:"created_at"`
}

type Artifact struct {
	ID        ArtifactID      `json:"id"`
	TenantID  TenantID        `json:"tenant_id"`
	SessionID SessionID       `json:"session_id"`
	Type      ArtifactType    `json:"type"`
	ObjectKey string          `json:"object_key"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type PromptVersion struct {
	Name      PromptName      `json:"name"`
	Version   PromptVersion   `json:"version"`
	TenantID  *TenantID       `json:"tenant_id,omitempty"`
	Content   string          `json:"content"`
	Schema    json.RawMessage `json:"schema,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
}

type AuditLog struct {
	ID              string          `json:"id"`
	TenantID        TenantID        `json:"tenant_id"`
	SessionID       SessionID       `json:"session_id"`
	UserID          UserID          `json:"user_id"`
	ToolName        ToolName        `json:"tool_name"`
	IdempotencyKey  IdempotencyKey  `json:"idempotency_key"`
	RequestHash     string          `json:"request_hash"`
	ResponseHash    string          `json:"response_hash"`
	Success         bool            `json:"success"`
	ApprovalRecord  json.RawMessage `json:"approval_record,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

type RiverJob struct {
	ID             JobID           `json:"id"`
	TenantID       TenantID        `json:"tenant_id"`
	SessionID      SessionID       `json:"session_id"`
	IdempotencyKey IdempotencyKey  `json:"idempotency_key"`
	Kind           string          `json:"kind"`
	Payload        json.RawMessage `json:"payload"`
	Attempt        int             `json:"attempt"`
	State          string          `json:"state"`
	CreatedAt      time.Time       `json:"created_at"`
	ScheduledAt    time.Time       `json:"scheduled_at"`
}

type Citation struct {
	ChunkID      ChunkID    `json:"chunk_id"`
	DocumentID   DocumentID `json:"document_id"`
	DocumentName string     `json:"document_name"`
	Ordinal      int        `json:"ordinal"`
	SourceLocation string   `json:"source_location,omitempty"`
	Score        float64    `json:"score"`
}

type RetrievedChunk struct {
	Chunk
	SemanticScore float64 `json:"semantic_score"`
	KeywordScore  float64 `json:"keyword_score"`
	CombinedScore float64 `json:"combined_score"`
}

type GenerationResult struct {
	Content      string          `json:"content"`
	Structured   json.RawMessage `json:"structured,omitempty"`
	Citations    []Citation      `json:"citations,omitempty"`
	ToolCalls    []ToolCallSpec  `json:"tool_calls,omitempty"`
	ModelUsed    LLMProvider     `json:"model_used"`
	TokenUsage   TokenUsage      `json:"token_usage"`
	EstimatedCost float64        `json:"estimated_cost"`
	TraceID      TraceID         `json:"trace_id"`
}

type TokenUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ToolCallSpec struct {
	Name      ToolName        `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
	ID        string          `json:"id"`
}

type ToolDefinition struct {
	Name        ToolName        `json:"name"`
	Description string          `json:"description"`
	Permission  ToolPermission  `json:"permission"`
	Schema      json.RawMessage `json:"schema"`
	CacheTTL    time.Duration   `json:"cache_ttl,omitempty"`
	RequiresApproval bool       `json:"requires_approval,omitempty"`
}

type ToolExecutionResult struct {
	Output      json.RawMessage `json:"output"`
	Error       error           `json:"-"`
	Cached      bool            `json:"cached"`
	AuditLogID  *string         `json:"audit_log_id,omitempty"`
}

type HITLRequest struct {
	ID          string          `json:"id"`
	TenantID    TenantID        `json:"tenant_id"`
	SessionID   SessionID       `json:"session_id"`
	ToolName    ToolName        `json:"tool_name"`
	ProposedAction json.RawMessage `json:"proposed_action"`
	Reason      string          `json:"reason"`
	RequestedAt time.Time       `json:"requested_at"`
	ExpiresAt   time.Time       `json:"expires_at"`
}

type IngestionJobArgs struct {
	DocumentID DocumentID `json:"document_id"`
	TenantID   TenantID   `json:"tenant_id"`
}

type ReportJobArgs struct {
	SessionID      SessionID      `json:"session_id"`
	TenantID       TenantID       `json:"tenant_id"`
	IdempotencyKey IdempotencyKey `json:"idempotency_key"`
	Parameters     json.RawMessage `json:"parameters"`
}

type GraphNode struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Kind        string          `json:"kind"`
	Dependencies []string       `json:"dependencies,omitempty"`
	Timeout     time.Duration   `json:"timeout"`
	MaxRetries  int             `json:"max_retries"`
	Config      json.RawMessage `json:"config,omitempty"`
}

type GraphDefinition struct {
	ID          string      `json:"id"`
	TenantID    TenantID    `json:"tenant_id"`
	Name        string      `json:"name"`
	Nodes       []GraphNode `json:"nodes"`
	Entrypoints []string    `json:"entrypoints"`
	MaxRuntime  time.Duration `json:"max_runtime"`
}

type ModelRouterConfig struct {
	DefaultProvider LLMProvider `json:"default_provider"`
	FallbackProviders []LLMProvider `json:"fallback_providers"`
	ComplexityThreshold float64 `json:"complexity_threshold"`
	CostPolicy      string      `json:"cost_policy"`
}

type LLMRequest struct {
	Provider    LLMProvider     `json:"provider,omitempty"`
	Model       string          `json:"model,omitempty"`
	Messages    []Message       `json:"messages"`
	Temperature float64         `json:"temperature,omitempty"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	JSONSchema  json.RawMessage `json:"json_schema,omitempty"`
	TraceID     TraceID         `json:"trace_id"`
}

type SearchQuery struct {
	TenantID       TenantID `json:"tenant_id"`
	Query          string   `json:"query"`
	TopK           int      `json:"top_k"`
	HybridWeight   float64  `json:"hybrid_weight"`
	RerankTopK     int      `json:"rerank_top_k"`
	DocumentFilter []DocumentID `json:"document_filter,omitempty"`
}

type RateLimitConfig struct {
	RPM      int           `json:"rpm"`
	Burst    int           `json:"burst"`
	Window   time.Duration `json:"window"`
	KeyScope string        `json:"key_scope"`
}

type JWTPayload struct {
	TenantID TenantID  `json:"tenant_id"`
	UserID   UserID    `json:"user_id"`
	Exp      int64     `json:"exp"`
}

type APIKey struct {
	ID        string    `json:"id"`
	TenantID  TenantID  `json:"tenant_id"`
	KeyHash   string    `json:"-"`
	Scopes    []string  `json:"scopes"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type ChannelMessage struct {
	Channel   ChannelType `json:"channel"`
	ChannelID string      `json:"channel_id"`
	Text      string      `json:"text"`
	Metadata  json.RawMessage `json:"metadata,omitempty"`
}

type ChannelResponse struct {
	Text        string          `json:"text"`
	Structured  json.RawMessage `json:"structured,omitempty"`
	Citations   []Citation      `json:"citations,omitempty"`
	Keyboard    json.RawMessage `json:"keyboard,omitempty"`
	JobID       *JobID          `json:"job_id,omitempty"`
}

type QualityMetrics struct {
	RetrievalRecallK float64 `json:"retrieval_recall_k"`
	CitationCoverage float64 `json:"citation_coverage"`
	GroundednessScore float64 `json:"groundedness_score"`
	TraceID          TraceID `json:"trace_id"`
}

type Config struct {
	ServerPort          string            `mapstructure:"server_port"`
	DatabaseURL         string            `mapstructure:"database_url"`
	RedisURL            string            `mapstructure:"redis_url"`
	ObjectStorageURL    string            `mapstructure:"object_storage_url"`
	ObjectStorageBucket string            `mapstructure:"object_storage_bucket"`
	DefaultModelRouter  ModelRouterConfig `mapstructure:"default_model_router"`
	EmbeddingProvider   EmbeddingProvider `mapstructure:"embedding_provider"`
	OllamaURL           string            `mapstructure:"ollama_url"`
	OfflineMode         bool              `mapstructure:"offline_mode"`
	JWTSecret           string            `mapstructure:"jwt_secret"`
	EnableTelegram      bool              `mapstructure:"enable_telegram"`
	EnableWebWidget     bool              `mapstructure:"enable_web_widget"`
}

type Server struct {
	Config         Config
	Pool           *pgxpool.Pool
	RiverClient    *river.Client[pgx.Tx]
	Router         *gin.Engine
	ModelRouter    ModelRouter
	SessionManager SessionManager
	ToolRegistry   ToolRegistry
	RAGEngine      RAGEngine
	ChannelAdapters []ChannelAdapter
}

type Worker struct {
	Config      Config
	Pool        *pgxpool.Pool
	RiverClient *river.Client[pgx.Tx]
	Workers     *river.Workers
}

type Runtime struct {
	Pool           *pgxpool.Pool
	SessionManager SessionManager
	JobClient      *river.Client[pgx.Tx]
	GraphEngine    GraphEngine
	HITLService    HITLService
}

type Database struct {
	Pool *pgxpool.Pool
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, reader io.Reader, size int64, contentType string) error
	Get(ctx context.Context, key string) (io.ReadCloser, int64, error)
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type ModelRouter interface {
	Route(ctx context.Context, req LLMRequest, taskComplexity float64) (GenerationResult, error)
	EstimateCost(provider LLMProvider, model string, usage TokenUsage) float64
}

type PromptRegistry interface {
	Load(ctx context.Context, name PromptName, version PromptVersion, tenantID *TenantID) (PromptVersion, error)
	Latest(ctx context.Context, name PromptName, tenantID *TenantID) (PromptVersion, error)
	Register(ctx context.Context, prompt PromptVersion) error
}

type StructuredOutputParser interface {
	Parse(ctx context.Context, raw []byte, target any) error
	Validate(ctx context.Context, raw []byte, schema json.RawMessage) error
}

type SessionManager interface {
	Create(ctx context.Context, tenantID TenantID, userID UserID, tier OrchestrationTier, channel ChannelType) (Session, error)
	Load(ctx context.Context, tenantID TenantID, sessionID SessionID) (Session, error)
	Update(ctx context.Context, session Session) (Session, error)
	Transition(ctx context.Context, tenantID TenantID, sessionID SessionID, from SessionState, to SessionState) (Session, error)
	ExpireInactive(ctx context.Context, before time.Time) error
	AppendMessage(ctx context.Context, msg Message) error
	GetMessages(ctx context.Context, tenantID TenantID, sessionID SessionID, limit int) ([]Message, error)
	PruneHistory(ctx context.Context, tenantID TenantID, sessionID SessionID, maxTurns int) error
}

type RAGEngine interface {
	Ingest(ctx context.Context, doc Document) error
	Reingest(ctx context.Context, docID DocumentID, tenantID TenantID) error
	Search(ctx context.Context, q SearchQuery) ([]RetrievedChunk, error)
	Rerank(ctx context.Context, query string, chunks []RetrievedChunk, topK int) ([]RetrievedChunk, error)
	DeleteStaleChunks(ctx context.Context, docID DocumentID, tenantID TenantID) error
}

type ToolRegistry interface {
	Register(def ToolDefinition, handler ToolHandler) error
	List(ctx context.Context, permission ToolPermission) []ToolDefinition
	Get(name ToolName) (ToolDefinition, ToolHandler, bool)
	Execute(ctx context.Context, tenantID TenantID, sessionID SessionID, userID UserID, spec ToolCallSpec) (ToolExecutionResult, error)
}

type ToolHandler func(ctx context.Context, tenantID TenantID, sessionID SessionID, userID UserID, args json.RawMessage) (json.RawMessage, error)

type Guardrails interface {
	ValidateInput(ctx context.Context, tenantID TenantID, input string) error
	EvaluateCitations(ctx context.Context, answer string, citations []Citation, chunks []Chunk) (QualityMetrics, error)
	RegisterEvaluationHook(hook EvaluationHook)
}

type EvaluationHook func(ctx context.Context, metrics QualityMetrics)

type GraphEngine interface {
	Execute(ctx context.Context, def GraphDefinition, input json.RawMessage) (json.RawMessage, error)
	DetectDeadlock(def GraphDefinition) error
}

type HITLService interface {
	RequestApproval(ctx context.Context, req HITLRequest) error
	Approve(ctx context.Context, id string, operatorID string) error
	Reject(ctx context.Context, id string, operatorID string, reason string) error
	GetPending(ctx context.Context, tenantID TenantID) ([]HITLRequest, error)
	Notify(ctx context.Context, req HITLRequest) error
}

type ChannelAdapter interface {
	ChannelType() ChannelType
	HandleWebhook(w http.ResponseWriter, r *http.Request) error
	SendMessage(ctx context.Context, channelID string, resp ChannelResponse) error
	FormatResponse(resp ChannelResponse) (string, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, key string, cfg RateLimitConfig) (bool, time.Duration, error)
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
	End()
	SetError(err error)
	SetAttributes(attrs map[string]string)
}

type MetricsCollector interface {
	RecordLLMTokenUsage(provider LLMProvider, model string, usage TokenUsage)
	RecordRetrievalLatency(tenantID TenantID, duration time.Duration, percentile string)
	RecordQueueDepth(queueName string, depth int)
	RecordError(category string, tenantID TenantID)
}

type CostTracker interface {
	Log(ctx context.Context, tenantID TenantID, sessionID SessionID, provider LLMProvider, model string, usage TokenUsage, cost float64) error
	GetBudgetUsage(ctx context.Context, tenantID TenantID, since time.Time) (float64, error)
}

type AuthService interface {
	AuthenticateAPIKey(ctx context.Context, key string) (TenantID, []string, error)
	AuthenticateJWT(ctx context.Context, token string) (JWTPayload, error)
	IssueJWT(ctx context.Context, tenantID TenantID, userID UserID, ttl time.Duration) (string, error)
}

type Repository interface {
	CreateTenant(ctx context.Context, tenant Tenant) error
	GetTenant(ctx context.Context, id TenantID) (Tenant, error)
	CreateUser(ctx context.Context, user User) error
	GetUserByChannel(ctx context.Context, tenantID TenantID, channel ChannelType, channelID string) (User, error)
	CreateSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, tenantID TenantID, id SessionID) (Session, error)
	UpdateSession(ctx context.Context, session Session) (Session, error)
	CreateMessage(ctx context.Context, msg Message) error
	GetMessages(ctx context.Context, tenantID TenantID, sessionID SessionID, limit int) ([]Message, error)
	CreateDocument(ctx context.Context, doc Document) error
	GetDocument(ctx context.Context, tenantID TenantID, id DocumentID) (Document, error)
	UpdateDocumentStatus(ctx context.Context, tenantID TenantID, id DocumentID, status IngestionStatus) error
	CreateChunk(ctx context.Context, chunk Chunk) error
	DeleteChunksForDocument(ctx context.Context, tenantID TenantID, docID DocumentID) error
	SearchSemantic(ctx context.Context, tenantID TenantID, vector []float32, topK int) ([]RetrievedChunk, error)
	SearchKeyword(ctx context.Context, tenantID TenantID, query string, topK int) ([]RetrievedChunk, error)
	CreateArtifact(ctx context.Context, artifact Artifact) error
	GetPrompt(ctx context.Context, name PromptName, version PromptVersion, tenantID *TenantID) (PromptVersion, error)
	CreateAuditLog(ctx context.Context, log AuditLog) error
	GetAuditLogByIdempotencyKey(ctx context.Context, tenantID TenantID, key IdempotencyKey, tool ToolName) (AuditLog, error)
}

type IngestionService interface {
	Submit(ctx context.Context, tenantID TenantID, name string, raw io.Reader, size int64, contentType string, metadata json.RawMessage) (Document, error)
	Process(ctx context.Context, docID DocumentID, tenantID TenantID) error
}

type PDFGenerator interface {
	Generate(ctx context.Context, data json.RawMessage, template string) (io.Reader, int64, error)
}

type ExcelGenerator interface {
	Generate(ctx context.Context, data json.RawMessage) (io.Reader, int64, error)
}

type TelegramAdapter struct {
	BotToken string
	Endpoint string
}

func NewTelegramAdapter(botToken, endpoint string) *TelegramAdapter { return nil }
func (a *TelegramAdapter) ChannelType() ChannelType { return "" }
func (a *TelegramAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) error { return nil }
func (a *TelegramAdapter) SendMessage(ctx context.Context, channelID string, resp ChannelResponse) error { return nil }
func (a *TelegramAdapter) FormatResponse(resp ChannelResponse) (string, error) { return "", nil }

type WebWidgetAdapter struct {
	Upgrader websocket.Upgrader
}

func NewWebWidgetAdapter() *WebWidgetAdapter { return nil }
func (a *WebWidgetAdapter) ChannelType() ChannelType { return "" }
func (a *WebWidgetAdapter) HandleWebhook(w http.ResponseWriter, r *http.Request) error { return nil }
func (a *WebWidgetAdapter) SendMessage(ctx context.Context, channelID string, resp ChannelResponse) error { return nil }
func (a *WebWidgetAdapter) FormatResponse(resp ChannelResponse) (string, error) { return "", nil }

func NewServer(cfg Config) (*Server, error) { return nil, nil }
func (s *Server) Run(ctx context.Context) error { return nil }
func (s *Server) RegisterRoutes() {}
func (s *Server) Shutdown(ctx context.Context) error { return nil }

func NewWorker(cfg Config) (*Worker, error) { return nil, nil }
func (w *Worker) RegisterHandlers() {}
func (w *Worker) Run(ctx context.Context) error { return nil }
func (w *Worker) Shutdown(ctx context.Context) error { return nil }

func NewRuntime(cfg Config) (*Runtime, error) { return nil, nil }

func LoadConfig(path string) (Config, error) { return Config{}, nil }

func NewDatabasePool(ctx context.Context, url string) (*pgxpool.Pool, error) { return nil, nil }

func NewRiverClient(pool *pgxpool.Pool, workers *river.Workers) (*river.Client[pgx.Tx], error) { return nil, nil }

func NewObjectStorage(cfg Config) (ObjectStorage, error) { return nil, nil }

func NewModelRouter(cfg Config) (ModelRouter, error) { return nil, nil }

func NewPromptRegistry(repo Repository) (PromptRegistry, error) { return nil, nil }

func NewToolRegistry(cache Cache) (ToolRegistry, error) { return nil, nil }

func NewRAGEngine(repo Repository, embedder Embedder, reranker Reranker) (RAGEngine, error) { return nil, nil }

type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

type Reranker interface {
	Rerank(ctx context.Context, query string, chunks []Chunk, topK int) ([]RankedChunk, error)
}

type RankedChunk struct {
	Chunk
	Score float64 `json:"score"`
	Rank  int     `json:"rank"`
}

func NewEmbedder(cfg Config) (Embedder, error) { return nil, nil }
func NewReranker(cfg Config) (Reranker, error) { return nil, nil }

func NewGuardrails() Guardrails { return nil }

func NewGraphEngine() GraphEngine { return nil }

func NewHITLService(repo Repository, adapters []ChannelAdapter) HITLService { return nil }

func NewRateLimiter(redisURL string) (RateLimiter, error) { return nil, nil }

func NewTracer(endpoint string) (Tracer, error) { return nil, nil }

func NewMetricsCollector() MetricsCollector { return nil }

func NewCostTracker(repo Repository) CostTracker { return nil }

func NewAuthService(repo Repository, jwtSecret string) AuthService { return nil }

func NewRepository(pool *pgxpool.Pool) Repository { return nil }

func NewIngestionService(repo Repository, storage ObjectStorage, embedder Embedder) IngestionService { return nil }

func NewPDFGenerator() PDFGenerator { return nil }
func NewExcelGenerator() ExcelGenerator { return nil }

func BuildErrorResponse(code string, message string, details map[string]any) gin.H { return nil }

func TenantScopeMiddleware(auth AuthService) gin.HandlerFunc { return nil }
func RateLimitMiddleware(limiter RateLimiter) gin.HandlerFunc { return nil }
func TraceMiddleware(tracer Tracer) gin
