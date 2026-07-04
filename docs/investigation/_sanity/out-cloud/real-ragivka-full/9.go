package ragivka

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/trace"
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

type OrchestrationTier int

const (
	TierL0 OrchestrationTier = iota
	TierL1
	TierL2
	TierL3
)

type SessionState int

const (
	StateActive SessionState = iota
	StateWaitingForHuman
	StateCompleted
	StateExpired
)

type ToolPermission int

const (
	PermissionRead ToolPermission = iota
	PermissionDraft
	PermissionWrite
)

type IngestionStatus int

const (
	StatusPending IngestionStatus = iota
	StatusIndexed
	StatusStale
)

type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelWeb      ChannelType = "web"
)

type Tenant struct {
	ID        TenantID
	Name      string
	CreatedAt time.Time
}

type User struct {
	ID          UserID
	TenantID    TenantID
	ChannelType ChannelType
	ChannelID   string
	CreatedAt   time.Time
}

type Session struct {
	ID                SessionID
	TenantID          TenantID
	UserID            UserID
	State             SessionState
	Version           int
	OrchestrationTier OrchestrationTier
	Channel           ChannelType
	ExpiresAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Message struct {
	ID            MessageID
	TenantID      TenantID
	SessionID     SessionID
	Role          string
	Content       string
	CitationRefs  []string
	TokenCount    int
	JobID         *JobID
	CreatedAt     time.Time
}

type Document struct {
	ID              DocumentID
	TenantID        TenantID
	S3Key           string
	Version         int
	IngestionStatus IngestionStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Chunk struct {
	ID         ChunkID
	TenantID   TenantID
	DocumentID DocumentID
	Ordinal    int
	Content    string
	Vector     []float32
	TsVector   string
	Metadata   map[string]string
}

type Artifact struct {
	ID        ArtifactID
	TenantID  TenantID
	SessionID SessionID
	S3Key     string
	Type      string
	CreatedAt time.Time
}

type PromptVersion struct {
	ID        int
	TenantID  TenantID
	Name      PromptName
	Version   PromptVersion
	Content   string
	CreatedAt time.Time
}

type AuditLog struct {
	ID             int
	TenantID       TenantID
	UserID         *UserID
	SessionID      *SessionID
	ToolName       ToolName
	IdempotencyKey IdempotencyKey
	RequestHash    string
	ResponseHash   string
	ApprovalRecord json.RawMessage
	CreatedAt      time.Time
}

type RiverJob struct {
	ID             JobID
	TenantID       TenantID
	SessionID      *SessionID
	IdempotencyKey IdempotencyKey
	Payload        json.RawMessage
	Attempt        int
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type LLMProvider string

const (
	ProviderOpenAI    LLMProvider = "openai"
	ProviderAnthropic LLMProvider = "anthropic"
	ProviderOpenRouter LLMProvider = "openrouter"
	ProviderGemini    LLMProvider = "gemini"
	ProviderOllama    LLMProvider = "ollama"
)

type ModelConfig struct {
	Provider   LLMProvider
	ModelName  string
	APIKey     string
	BaseURL    string
	MaxTokens  int
	Temperature float64
}

type LLMRequest struct {
	Model      string
	Messages   []LLMMessage
	Schema     json.RawMessage
	Provider   LLMProvider
}

type LLMMessage struct {
	Role    string
	Content string
}

type LLMResponse struct {
	Content    string
	Structured json.RawMessage
	Provider   LLMProvider
	Model      string
	Usage      TokenUsage
	Cost       float64
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type RetrievalResult struct {
	ChunkID    ChunkID
	DocumentID DocumentID
	DocumentName string
	Ordinal    int
	Content    string
	Score      float64
	Source     string
}

type Citation struct {
	DocumentName string
	Ordinal      int
	SourceLocation string
}

type RAGContext struct {
	Chunks   []RetrievalResult
	Citations []Citation
}

type ToolDefinition struct {
	Name        ToolName
	Description string
	Permission  ToolPermission
	Schema      json.RawMessage
	Handler     ToolHandler
	CacheTTL    time.Duration
}

type ToolHandler func(ctx context.Context, args json.RawMessage) (json.RawMessage, error)

type ToolInvocation struct {
	Name ToolName
	Args json.RawMessage
}

type ToolResult struct {
	Name    ToolName
	Output  json.RawMessage
	Error   error
	FromCache bool
}

type FunctionCall struct {
	Name ToolName
	Args json.RawMessage
}

type GraphNode struct {
	ID       string
	Name     string
	Timeout  time.Duration
	Depends  []string
	Execute  func(ctx context.Context, input json.RawMessage) (json.RawMessage, error)
}

type Graph struct {
	ID     string
	Nodes  []GraphNode
}

type HITLRequest struct {
	SessionID      SessionID
	ToolName       *ToolName
	IdempotencyKey *IdempotencyKey
	Reason         string
	Payload        json.RawMessage
}

type HITLResponse struct {
	SessionID SessionID
	Approved  bool
	ActorID   string
}

type IngestionJob struct {
	DocumentID DocumentID
	TenantID   TenantID
	SourceURL  string
}

type ReportJob struct {
	SessionID      SessionID
	TenantID       TenantID
	IdempotencyKey IdempotencyKey
	Parameters     json.RawMessage
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

type Config struct {
	DatabaseURL        string
	RedisURL           string
	S3Endpoint         string
	S3Bucket           string
	OpenAIKey          string
	AnthropicKey       string
	OpenRouterKey        string
	GeminiKey          string
	OllamaURL          string
	DefaultEmbeddingModel string
	OfflineMode        bool
	ServerMode         string
	WorkerConcurrency  int
}

type Server struct {
	DB     *pgxpool.Pool
	Tracer trace.Tracer
	Config Config
}

type Worker struct {
	DB     *pgxpool.Pool
	Tracer trace.Tracer
	Config Config
}

type TelegramAdapter struct {
	Token string
}

type WebWidgetAPI struct {
	Server *Server
}

type SessionManager struct {
	DB *pgxpool.Pool
}

type ModelRouter interface {
	Route(ctx context.Context, task string, complexity int, budget float64) (ModelConfig, error)
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

type PromptRegistry interface {
	Load(ctx context.Context, tenantID TenantID, name PromptName, version PromptVersion) (string, error)
	Save(ctx context.Context, tenantID TenantID, name PromptName, version PromptVersion, content string) error
}

type StructuredOutputParser interface {
	Parse(ctx context.Context, raw json.RawMessage, target interface{}) error
	Schema(target interface{}) (json.RawMessage, error)
}

type KnowledgeStore interface {
	Ingest(ctx context.Context, doc Document, reader io.Reader) error
	Retrieve(ctx context.Context, tenantID TenantID, query string, topK int) (RAGContext, error)
	ReRank(ctx context.Context, query string, results []RetrievalResult, topK int) ([]RetrievalResult, error)
}

type ToolRegistry interface {
	Register(def ToolDefinition) error
	Invoke(ctx context.Context, tenantID TenantID, name ToolName, args json.RawMessage, idempotencyKey *IdempotencyKey) (ToolResult, error)
}

type FSMAdapter interface {
	Load(ctx context.Context, tx pgx.Tx, sessionID SessionID) (Session, error)
	Transition(ctx context.Context, tx pgx.Tx, sessionID SessionID, from, to SessionState, version int) (int, error)
	ExpireInactive(ctx context.Context, before time.Time) error
}

type AuditLogger interface {
	Log(ctx context.Context, tx pgx.Tx, entry AuditLog) error
}

type HITLGate interface {
	RequestApproval(ctx context.Context, req HITLRequest) error
	Resolve(ctx context.Context, res HITLResponse) error
}

type GraphEngine interface {
	Execute(ctx context.Context, graph Graph, input json.RawMessage) (json.RawMessage, error)
}

type CriticHook interface {
	Evaluate(ctx context.Context, answer string, context RAGContext) (QualityMetrics, error)
}

type QualityMetrics struct {
	RetrievalRecallK float64
	CitationCoverage float64
	Groundedness     float64
}

type CostTracker interface {
	Record(ctx context.Context, tenantID TenantID, reqID string, usage TokenUsage, cost float64) error
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID) (bool, error)
}

func NewServer(cfg Config) (*Server, error) {}

func (s *Server) Start(ctx context.Context) error {}

func (s *Server) Stop(ctx context.Context) error {}

func NewWorker(cfg Config) (*Worker, error) {}

func (w *Worker) Start(ctx context.Context) error {}

func (w *Worker) Stop(ctx context.Context) error {}

func NewTelegramAdapter(token string) *TelegramAdapter {}

func (a *TelegramAdapter) HandleWebhook(ctx context.Context, body []byte) error {}

func (a *TelegramAdapter) SendMessage(ctx context.Context, chatID int64, text string, format string) error {}

func NewWebWidgetAPI(server *Server) *WebWidgetAPI {}

func (api *WebWidgetAPI) RegisterRoutes() {}

func (api *WebWidgetAPI) CreateSession(ctx context.Context, tenantID TenantID, userID UserID) (Session, error) {}

func (api *WebWidgetAPI) PostMessage(ctx context.Context, sessionID SessionID, content string) (Message, error) {}

func (api *WebWidgetAPI) GetHistory(ctx context.Context, sessionID SessionID) ([]Message, error) {}

func NewSessionManager(db *pgxpool.Pool) *SessionManager {}

func (m *SessionManager) CreateSession(ctx context.Context, tenantID TenantID, userID UserID, tier OrchestrationTier, channel ChannelType) (Session, error) {}

func (m *SessionManager) LoadSession(ctx context.Context, sessionID SessionID) (Session, error) {}

func (m *SessionManager) UpdateState(ctx context.Context, sessionID SessionID, from, to SessionState, version int) (int, error) {}

func (m *SessionManager) ExpireSessions(ctx context.Context, timeout time.Duration) error {}

func (m *SessionManager) TrimHistory(ctx context.Context, sessionID SessionID, maxTurns int) error {}

func NewModelRouter(configs map[LLMProvider]ModelConfig) ModelRouter {}

func (r *modelRouter) Route(ctx context.Context, task string, complexity int, budget float64) (ModelConfig, error) {}

func (r *modelRouter) Complete(ctx context.Context, req LLMRequest) (LLMResponse, error) {}

func NewPromptRegistry(db *pgxpool.Pool) PromptRegistry {}

func (p *promptRegistry) Load(ctx context.Context, tenantID TenantID, name PromptName, version PromptVersion) (string, error) {}

func (p *promptRegistry) Save(ctx context.Context, tenantID TenantID, name PromptName, version PromptVersion, content string) error {}

func NewStructuredOutputParser() StructuredOutputParser {}

func (p *structuredOutputParser) Parse(ctx context.Context, raw json.RawMessage, target interface{}) error {}

func (p *structuredOutputParser) Schema(target interface{}) (json.RawMessage, error) {}

func NewKnowledgeStore(db *pgxpool.Pool, embedder ModelRouter) KnowledgeStore {}

func (k *knowledgeStore) Ingest(ctx context.Context, doc Document, reader io.Reader) error {}

func (k *knowledgeStore) Retrieve(ctx context.Context, tenantID TenantID, query string, topK int) (RAGContext, error) {}

func (k *knowledgeStore) ReRank(ctx context.Context, query string, results []RetrievalResult, topK int) ([]RetrievalResult, error) {}

func NewToolRegistry(db *pgxpool.Pool, audit AuditLogger, cache RedisCache) ToolRegistry {}

func (t *toolRegistry) Register(def ToolDefinition) error {}

func (t *toolRegistry) Invoke(ctx context.Context, tenantID TenantID, name ToolName, args json.RawMessage, idempotencyKey *IdempotencyKey) (ToolResult, error) {}

func NewFSMAdapter(db *pgxpool.Pool) FSMAdapter {}

func (f *fsmAdapter) Load(ctx context.Context, tx pgx.Tx, sessionID SessionID) (Session, error) {}

func (f *fsmAdapter) Transition(ctx context.Context, tx pgx.Tx, sessionID SessionID, from, to SessionState, version int) (int, error) {}

func (f *fsmAdapter) ExpireInactive(ctx context.Context, before time.Time) error {}

func NewAuditLogger() AuditLogger {}

func (a *auditLogger) Log(ctx context.Context, tx pgx.Tx, entry AuditLog) error {}

func NewHITLGate(notifier Notifier) HITLGate {}

func (h *hitlGate) RequestApproval(ctx context.Context, req HITLRequest) error {}

func (h *hitlGate) Resolve(ctx context.Context, res HITLResponse) error {}

func NewGraphEngine() GraphEngine {}

func (g *graphEngine) Execute(ctx context.Context, graph Graph, input json.RawMessage) (json.RawMessage, error) {}

func NewCriticHook() CriticHook {}

func (c *criticHook) Evaluate(ctx context.Context, answer string, context RAGContext) (QualityMetrics, error) {}

func NewCostTracker(db *pgxpool.Pool) CostTracker {}

func (ct *costTracker) Record(ctx context.Context, tenantID TenantID, reqID string, usage TokenUsage, cost float64) error {}

func NewRateLimiter(redis RedisCache) RateLimiter {}

func (rl *rateLimiter) Allow(ctx context.Context, tenantID TenantID) (bool, error) {}

type RedisCache interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string, ttl time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, ttl time.Duration) error
}

type Notifier interface {
	Notify(ctx context.Context, recipient string, payload json.RawMessage) error
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, reader io.Reader) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type PDFGenerator interface {
	Generate(ctx context.Context, data interface{}, template string) (io.Reader, error)
}

type ExcelGenerator interface {
	Generate(ctx context.Context, data interface{}) (io.Reader, error)
}

type AuthMiddleware struct {
	DB *pgxpool.Pool
}

func (m *AuthMiddleware) Authenticate(ctx context.Context, token string) (TenantID, error) {}

func RunL0(ctx context.Context, server *Server, session Session, prompt string) (string, error) {}

func RunL1(ctx context.Context, server *Server, session Session, message string) (Message, error) {}

func RunL2(ctx context.Context, server *Server, session Session, job river.Job) error {}

func RunL3(ctx context.Context, server *Server, session Session, graph Graph, input json.RawMessage) (json.RawMessage, error) {}
