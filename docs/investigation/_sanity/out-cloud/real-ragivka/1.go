package ragivka

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"time"
)

type TenantID string
type SessionID string
type JobID string
type OperationKey string
type ToolName string
type PromptName string
type PromptVersion string
type DocumentID string
type ChunkID string
type ArtifactID string
type ModelProvider string
type Channel string
type FSMState string
type ToolPermission string
type ErrorCode string

const (
	StateActive          FSMState = "Active"
	StateWaitingForHuman FSMState = "WaitingForHuman"
	StateCompleted       FSMState = "Completed"
	StateExpired         FSMState = "Expired"
)

const (
	PermissionRead  ToolPermission = "Read"
	PermissionDraft ToolPermission = "Draft"
	PermissionWrite ToolPermission = "Write"
)

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderOpenRouter ModelProvider = "openrouter"
	ProviderGemini    ModelProvider = "gemini"
	ProviderOllama    ModelProvider = "ollama"
)

const (
	ChannelTelegram  Channel = "telegram"
	ChannelWebWidget Channel = "webwidget"
)

type Tenant struct {
	ID        TenantID
	Name      string
	Config    json.RawMessage
	CreatedAt time.Time
}

type Session struct {
	ID           SessionID
	TenantID     TenantID
	State        FSMState
	Context      json.RawMessage
	Version      int
	LastActivity time.Time
	ExpiresAt    *time.Time
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Message struct {
	ID        string
	SessionID SessionID
	Role      string
	Content   string
	Metadata  json.RawMessage
	CreatedAt time.Time
}

type KnowledgeDocument struct {
	ID         DocumentID
	TenantID   TenantID
	Title      string
	SourceURI  string
	RawObject  string
	Version    int
	Metadata   json.RawMessage
	ReIngested *time.Time
	CreatedAt  time.Time
}

type Chunk struct {
	ID         ChunkID
	DocumentID DocumentID
	TenantID   TenantID
	Ordinal    int
	SourceLoc  string
	Text       string
	Embedding  []float32
	Tokens     int
	Metadata   json.RawMessage
}

type Artifact struct {
	ID        ArtifactID
	TenantID  TenantID
	SessionID SessionID
	Kind      string
	ObjectKey string
	Metadata  json.RawMessage
	CreatedAt time.Time
}

type Prompt struct {
	ID        string
	Name      PromptName
	Version   PromptVersion
	Template  string
	Variables []string
	CreatedAt time.Time
}

type AuditLog struct {
	ID             string
	IdempotencyKey OperationKey
	TenantID       TenantID
	ToolName       ToolName
	FSMState       FSMState
	RequestHash    string
	ResponseHash   string
	CreatedAt      time.Time
}

type ToolSchema struct {
	Name        ToolName
	Permission  ToolPermission
	InputSchema json.RawMessage
	OutputSchema json.RawMessage
	CachePolicy *CachePolicy
}

type CachePolicy struct {
	TTL         time.Duration
	KeyFields   []string
	Invalidates []ToolName
}

type ToolResult struct {
	Data       json.RawMessage
	Error      *ToolError
	CacheHit   bool
	ApprovedBy *string
}

type ToolError struct {
	Code    ErrorCode
	Message string
	Details json.RawMessage
}

type APIError struct {
	Code    ErrorCode
	Message string
	Details json.RawMessage
}

type ModelRouter interface {
	Route(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

type LLMRequest struct {
	TenantID TenantID
	Task     string
	Messages []Message
	Prompt   PromptName
	Version  PromptVersion
	Schema   json.RawMessage
}

type LLMResponse struct {
	Provider    ModelProvider
	Model       string
	Usage       TokenUsage
	CostUSD     float64
	Structured  json.RawMessage
	RawText     string
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type Retriever interface {
	Retrieve(ctx context.Context, req *RetrievalRequest) (*RetrievalResult, error)
}

type RetrievalRequest struct {
	TenantID   TenantID
	Query      string
	TopK       int
	MinScore   float64
	Filter     map[string]string
	UseHybrid  bool
	ReRank     bool
}

type RetrievalResult struct {
	Results []RankedChunk
	RecallK float64
}

type RankedChunk struct {
	Chunk   *Chunk
	Score   float64
	Rank    int
}

type Citation struct {
	DocumentName string
	ChunkOrdinal int
	SourceLoc    string
	Quote        string
}

type RAGAnswer struct {
	Text      string
	Citations []Citation
}

type FSMDriver interface {
	Transition(ctx context.Context, sessionID SessionID, event Event, dbTx *sql.Tx) (*Session, error)
}

type Event struct {
	Type    string
	Payload json.RawMessage
}

type Job interface {
	Kind() string
	Args() json.RawMessage
	InsertOpts() *JobInsertOptions
}

type JobInsertOptions struct {
	Queue      string
	Priority   int
	ScheduledAt *time.Time
}

type Worker interface {
	Register(kind string, handler JobHandler)
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type JobHandler func(ctx context.Context, job *JobRecord) error

type JobRecord struct {
	ID      JobID
	Kind    string
	Args    json.RawMessage
	Attempt int
}

type ChannelAdapter interface {
	Channel() Channel
	Send(ctx context.Context, recipient string, msg *OutboundMessage) error
	Receive(ctx context.Context, payload io.Reader) (*InboundMessage, error)
}

type OutboundMessage struct {
	Text         string
	HTML         string
	Markdown     string
	Keyboard     []Button
	Attachments  []Attachment
}

type InboundMessage struct {
	Channel   Channel
	SenderID  string
	Text      string
	Payload   json.RawMessage
	Timestamp time.Time
}

type Button struct {
	Text string
	Data string
}

type Attachment struct {
	Name     string
	ObjectKey string
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID, resource string) (bool, *RateLimitStatus, error)
}

type RateLimitStatus struct {
	Limit     int
	Remaining int
	ResetAt   time.Time
}

type ObjectStorage interface {
	Put(ctx context.Context, key string, body io.Reader, size int64) error
	Get(ctx context.Context, key string) (io.ReadCloser, error)
	Delete(ctx context.Context, key string) error
}

type PGPool interface{}

type EvalHook interface {
	LogRetrievalRecallK(ctx context.Context, reqID string, k int, recall float64) error
	LogCitationCoverage(ctx context.Context, reqID string, coverage float64) error
	LogGroundedness(ctx context.Context, reqID string, score *float64) error
}

type PIIStripper interface {
	Strip(ctx context.Context, text string) (string, error)
}

type CostTracker interface {
	Record(ctx context.Context, tenantID TenantID, reqID string, usage TokenUsage, costUSD float64) error
}

type MetricsCollector interface {
	RecordLLMUsage(ctx context.Context, provider ModelProvider, usage TokenUsage)
	RecordRetrievalLatency(ctx context.Context, p50, p95 time.Duration)
	RecordQueueDepth(ctx context.Context, queue string, depth int)
	RecordError(ctx context.Context, code ErrorCode)
}

type Auth interface {
	Authenticate(ctx context.Context, token string) (*TenantID, error)
	AuthorizeAPIKey(ctx context.Context, key string) (*TenantID, error)
}

type Parser interface {
	Parse(ctx context.Context, doc *KnowledgeDocument) ([]*Chunk, error)
}

type Chunker interface {
	Chunk(ctx context.Context, text string) ([]*Chunk, error)
}

type ReRanker interface {
	ReRank(ctx context.Context, query string, chunks []*Chunk) ([]*RankedChunk, error)
}

type ArtifactGenerator interface {
	GeneratePDF(ctx context.Context, data json.RawMessage) (io.Reader, error)
	GenerateExcel(ctx context.Context, data json.RawMessage) (io.Reader, error)
}

type IdempotencyStore interface {
	Claim(ctx context.Context, key OperationKey, ttl time.Duration) (bool, error)
	Complete(ctx context.Context, key OperationKey, result json.RawMessage) error
	Get(ctx context.Context, key OperationKey) (*IdempotencyRecord, error)
}

type IdempotencyRecord struct {
	Key       OperationKey
	Result    json.RawMessage
	ExpiresAt time.Time
}

func NewSessionManager(db *sql.DB, fsm FSMDriver, limit int) *SessionManager { return nil }

type SessionManager struct{}

func (m *SessionManager) Create(ctx context.Context, tenantID TenantID) (*Session, error) { return nil, nil }
func (m *SessionManager) Get(ctx context.Context, id SessionID) (*Session, error) { return nil, nil }
func (m *SessionManager) AppendMessage(ctx context.Context, id SessionID, msg *Message) (*Session, error) { return nil, nil }
func (m *SessionManager) SummarizeHistory(ctx context.Context, id SessionID, keep int) error { return nil }
func (m *SessionManager) Expire(ctx context.Context) error { return nil }

func NewToolRegistry() *ToolRegistry { return nil }

type ToolRegistry struct{}

func (r *ToolRegistry) Register(tool ToolSchema, handler ToolHandler) error { return nil }
func (r *ToolRegistry) Get(name ToolName) (*ToolSchema, ToolHandler, error) { return nil, nil, nil }
func (r *ToolRegistry) Schema() json.RawMessage { return nil }

type ToolHandler func(ctx context.Context, tenantID TenantID, args json.RawMessage) (*ToolResult, error)

func NewModelRouter(providers map[ModelProvider]LLMClient, policy *RoutingPolicy) ModelRouter { return nil }

type LLMClient interface {
	Complete(ctx context.Context, req *LLMRequest) (*LLMResponse, error)
}

type RoutingPolicy struct {
	FallbackProvider ModelProvider
	CostTiers        map[string]ModelProvider
}

func NewRetriever(pg PGPool, reranker ReRanker) Retriever { return nil }

func NewRateLimiter(redisAddr string) RateLimiter { return nil }

func NewChannelAdapter(channel Channel, cfg json.RawMessage) (ChannelAdapter, error) { return nil, nil }

func NewWorker(connStr string) Worker { return nil }

func NewArtifactGenerator() ArtifactGenerator { return nil }

func NewObjectStorage(cfg json.RawMessage) (ObjectStorage, error) { return nil, nil }

func NewAuth(jwkURL string, db *sql.DB) Auth { return nil }

func NewCostTracker(db *sql.DB) CostTracker { return nil }

func NewMetricsCollector() MetricsCollector { return nil }

func NewPIIStripper() PIIStripper { return nil }

func NewEvalHook(db *sql.DB) EvalHook { return nil }

func NewParser() Parser { return nil }

func NewChunker(chunkSize, overlap int) Chunker { return nil }

func NewFSMDriver(db *sql.DB) FSMDriver { return nil }

func NewIdempotencyStore(redisAddr string) IdempotencyStore { return nil }
