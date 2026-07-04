package ragivka

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type TenantID uuid.UUID
type UserID uuid.UUID
type SessionID uuid.UUID
type MessageID uuid.UUID
type DocumentID uuid.UUID
type ChunkID uuid.UUID
type ArtifactID uuid.UUID
type JobID uuid.UUID
type PromptName string
type PromptVersion string
type ToolName string
type IdempotencyKey string

type OrchestrationTier int

const (
	TierL0Deterministic OrchestrationTier = iota
	TierL1ToolAssistant
	TierL2WorkflowPipeline
	TierL3MultiAgentGraph
)

type FSMState string

const (
	StateActive          FSMState = "active"
	StateWaitingForHuman FSMState = "waiting_for_human"
	StateCompleted       FSMState = "completed"
	StateExpired         FSMState = "expired"
)

type ToolPermission string

const (
	PermissionRead  ToolPermission = "read"
	PermissionDraft ToolPermission = "draft"
	PermissionWrite ToolPermission = "write"
)

type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderOpenRouter Provider = "openrouter"
	ProviderGemini    Provider = "gemini"
	ProviderOllama    Provider = "ollama"
)

type IngestionStatus string

const (
	StatusPending  IngestionStatus = "pending"
	StatusIndexed  IngestionStatus = "indexed"
	StatusStale    IngestionStatus = "stale"
	StatusFailed   IngestionStatus = "failed"
)

type ChannelType string

const (
	ChannelTelegram ChannelType = "telegram"
	ChannelWeb      ChannelType = "web"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

type Tenant struct {
	ID        TenantID
	Name      string
	APIKey    string
	CreatedAt time.Time
}

type User struct {
	ID          UserID
	TenantID    TenantID
	ChannelType ChannelType
	ChannelID   string
	Metadata    json.RawMessage
	CreatedAt   time.Time
}

type Session struct {
	ID                SessionID
	TenantID          TenantID
	UserID            UserID
	State             FSMState
	Version           int
	OrchestrationTier OrchestrationTier
	Channel           ChannelType
	ExpiresAt         time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Message struct {
	ID           MessageID
	TenantID     TenantID
	SessionID    SessionID
	Role         Role
	Content      string
	CitationRefs []ChunkID
	TokenCount   int
	JobID        *JobID
	CreatedAt    time.Time
}

type Document struct {
	ID              DocumentID
	TenantID        TenantID
	S3Key           string
	Filename        string
	Version         int
	IngestionStatus IngestionStatus
	Metadata        json.RawMessage
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type Chunk struct {
	ID         ChunkID
	TenantID   TenantID
	DocumentID DocumentID
	Ordinal    int
	Content    string
	SourceLoc  string
	Vector     []float32
	Tsvector   string
	Metadata   json.RawMessage
	CreatedAt  time.Time
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
	ID        uuid.UUID
	TenantID  *TenantID
	Name      PromptName
	Version   PromptVersion
	Content   string
	CreatedAt time.Time
}

type AuditLog struct {
	ID             uuid.UUID
	TenantID       TenantID
	UserID         *UserID
	SessionID      *SessionID
	ToolName       ToolName
	IdempotencyKey IdempotencyKey
	RequestHash    string
	ResponseHash   string
	ApprovalRecord *json.RawMessage
	CreatedAt      time.Time
}

type RiverJob struct {
	ID             JobID
	TenantID       TenantID
	SessionID      *SessionID
	IdempotencyKey IdempotencyKey
	JobType        string
	Payload        json.RawMessage
	Attempt        int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ModelRequest struct {
	TenantID TenantID
	TaskType string
	Prompt   string
	Schema   json.RawMessage
	Provider *Provider
}

type ModelResponse struct {
	Provider     Provider
	Model        string
	RawContent   string
	Structured   json.RawMessage
	TokenUsage   TokenUsage
	Errors       []string
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCostUSD float64
}

type RetrievalResult struct {
	Chunk     Chunk
	Score     float64
	RankScore float64
}

type Citation struct {
	DocumentName string
	Ordinal      int
	SourceLoc    string
}

type ToolArgument struct {
	Name  string
	Value json.RawMessage
}

type ToolResult struct {
	ToolName ToolName
	Output   json.RawMessage
	Errors   []string
	Cached   bool
}

type HITLRequest struct {
	SessionID      SessionID
	ToolName         *ToolName
	IdempotencyKey   *IdempotencyKey
	Reason           string
	OperatorChannel  string
}

type HITLResponse struct {
	SessionID SessionID
	Approved  bool
	OperatorID string
	RespondedAt time.Time
}

type GraphNode struct {
	ID       string
	Name     string
	Timeout  time.Duration
	Depends  []string
	Execute  func(context.Context, GraphContext) (json.RawMessage, error)
}

type GraphContext struct {
	SessionID SessionID
	TenantID  TenantID
	Inputs    map[string]json.RawMessage
	Tracer    io.Writer
}

type GraphEngine interface {
	Execute(ctx context.Context, nodes []GraphNode) (map[string]json.RawMessage, error)
}

type ModelRouter interface {
	Route(ctx context.Context, req ModelRequest) (*ModelResponse, error)
}

type PromptRegistry interface {
	Get(ctx context.Context, tenantID *TenantID, name PromptName, version PromptVersion) (string, error)
}

type StructuredOutputParser interface {
	Parse(ctx context.Context, raw json.RawMessage, schema json.RawMessage, target interface{}) error
}

type RetrievalEngine interface {
	HybridSearch(ctx context.Context, tenantID TenantID, query string, topK int) ([]RetrievalResult, error)
	ReRank(ctx context.Context, results []RetrievalResult, query string, topK int) ([]RetrievalResult, error)
}

type Tool interface {
	Name() ToolName
	Permission() ToolPermission
	Schema() json.RawMessage
	Execute(ctx context.Context, args []ToolArgument) (*ToolResult, error)
}

type ToolRegistry interface {
	Register(tool Tool) error
	Get(name ToolName) (Tool, error)
	List(permission ToolPermission) []Tool
}

type ReadTool interface {
	Tool
	CacheTTL() time.Duration
}

type DraftTool interface {
	Tool
}

type WriteTool interface {
	Tool
	IdempotencyKey(args []ToolArgument) (IdempotencyKey, error)
	RequiresApproval() bool
}

type SessionManager interface {
	Create(ctx context.Context, tenantID TenantID, userID UserID, tier OrchestrationTier, channel ChannelType) (*Session, error)
	Load(ctx context.Context, tenantID TenantID, sessionID SessionID) (*Session, error)
	Transition(ctx context.Context, sessionID SessionID, from FSMState, to FSMState) (*Session, error)
	Expire(ctx context.Context, before time.Time) error
}

type ConversationStore interface {
	SaveMessage(ctx context.Context, msg *Message) error
	ListMessages(ctx context.Context, tenantID TenantID, sessionID SessionID, limit int) ([]Message, error)
	TrimHistory(ctx context.Context, tenantID TenantID, sessionID SessionID, keep int) error
}

type IngestionPipeline interface {
	Enqueue(ctx context.Context, tenantID TenantID, reader io.Reader, filename string) (*Document, error)
	Process(ctx context.Context, documentID DocumentID) error
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID, key string, limit int, window time.Duration) (bool, error)
}

type ChannelAdapter interface {
	Receive(ctx context.Context, r *http.Request) (*ChannelMessage, error)
	Send(ctx context.Context, tenantID TenantID, channelID string, response ChannelResponse) error
}

type ChannelMessage struct {
	TenantID  TenantID
	UserID    UserID
	SessionID *SessionID
	Channel   ChannelType
	ChannelID string
	Text      string
	Metadata  json.RawMessage
}

type ChannelResponse struct {
	Text      string
	HTML      string
	Markdown  string
	Keyboard  []InlineButton
	Citations []Citation
}

type InlineButton struct {
	Text string
	Data string
}

type AuthService interface {
	AuthenticateAPIKey(ctx context.Context, key string) (*Tenant, error)
	AuthenticateJWT(ctx context.Context, token string) (*Tenant, error)
}

type Config struct {
	DatabaseURL        string
	RedisAddr          string
	ObjectStorageURL   string
	OpenAIKey          string
	AnthropicKey       string
	OpenRouterKey        string
	GeminiKey          string
	OllamaURL          string
	OfflineMode        bool
	ServerMode         string
	WorkerConcurrency  int
	RateLimitRequests int
	RateLimitWindow   time.Duration
	EmbeddingModel    string
	RerankerModel     string
}

type RagivkaRuntime struct {
	Config            Config
	DB                *sql.DB
	SessionManager    SessionManager
	ConversationStore ConversationStore
	ModelRouter       ModelRouter
	PromptRegistry    PromptRegistry
	RetrievalEngine   RetrievalEngine
	ToolRegistry      ToolRegistry
	GraphEngine       GraphEngine
	IngestionPipeline IngestionPipeline
	RateLimiter       RateLimiter
	ChannelAdapters   map[ChannelType]ChannelAdapter
	AuthService       AuthService
}

func NewRuntime(cfg Config) (*RagivkaRuntime, error) {
	return nil, nil
}

func (r *RagivkaRuntime) HandleL0(ctx context.Context, msg ChannelMessage) (*ChannelResponse, error) {
	return nil, nil
}

func (r *RagivkaRuntime) HandleL1(ctx context.Context, msg ChannelMessage) (*ChannelResponse, error) {
	return nil, nil
}

func (r *RagivkaRuntime) HandleL2(ctx context.Context, msg ChannelMessage) (JobID, error) {
	return JobID{}, nil
}

func (r *RagivkaRuntime) HandleL3(ctx context.Context, msg ChannelMessage, graph []GraphNode) (JobID, error) {
	return JobID{}, nil
}

func (r *RagivkaRuntime) RunHTTPServer(ctx context.Context, addr string) error {
	return nil
}

func (r *RagivkaRuntime) RunWorker(ctx context.Context) error {
	return nil
}

func (r *RagivkaRuntime) RunFSMExpiry(ctx context.Context, interval time.Duration) error {
	return nil
}

//go:embed migrations/*.sql
var Migrations embed.FS
