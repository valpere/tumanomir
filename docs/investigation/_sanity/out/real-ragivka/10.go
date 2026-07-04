package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
)

// TenantID uniquely identifies a tenant in the system
type TenantID string

// SessionID uniquely identifies a conversation session
type SessionID string

// DocumentID uniquely identifies a document in the system
type DocumentID string

// ArtifactID uniquely identifies a generated artifact
type ArtifactID string

// ToolName uniquely identifies a registered tool
type ToolName string

// ModelProvider represents supported LLM providers
type ModelProvider string

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderGemini    ModelProvider = "gemini"
	ProviderLocal     ModelProvider = "local"
)

// FSMState represents the state of a conversation session
type FSMState string

const (
	StateActive        FSMState = "active"
	StateWaitingForHuman FSMState = "waiting_for_human"
	StateCompleted     FSMState = "completed"
	StateExpired       FSMState = "expired"
)

// ToolPermission defines access levels for tools
type ToolPermission string

const (
	PermissionRead   ToolPermission = "read"
	PermissionDraft  ToolPermission = "draft"
	PermissionWrite  ToolPermission = "write"
)

// PromptVersion represents a versioned system prompt
type PromptVersion struct {
	Name    string
	Version string
}

// Session represents a conversation session
type Session struct {
	ID          SessionID
	TenantID    TenantID
	State       FSMState
	Version     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	Context     []Message
	LastMessage Message
}

// Message represents a single message in the conversation
type Message struct {
	Role      string
	Content   string
	Timestamp time.Time
}

// ToolRegistration holds tool metadata and configuration
type ToolRegistration struct {
	Name        ToolName
	Permissions []ToolPermission
	Description string
	Schema      interface{}
	CacheTTL    *time.Duration
}

// ModelRouter routes requests to appropriate LLM providers
type ModelRouter interface {
	Route(ctx context.Context, taskType string) (ModelProvider, error)
}

// ToolRegistry manages registered tools and their permissions
type ToolRegistry interface {
	Register(tool ToolRegistration) error
	Get(name ToolName) (*ToolRegistration, error)
	ValidatePermission(name ToolName, permission ToolPermission) bool
}

// StateManager handles session state transitions and persistence
type StateManager interface {
	GetSession(ctx context.Context, id SessionID) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	TransitionState(ctx context.Context, id SessionID, from, to FSMState) error
	ExpireSession(ctx context.Context, id SessionID) error
}

// KnowledgeBase manages document ingestion and retrieval
type KnowledgeBase interface {
	IngestDocument(ctx context.Context, tenantID TenantID, content []byte, source string) (DocumentID, error)
	Retrieve(ctx context.Context, tenantID TenantID, query string, limit int) ([]RetrievedChunk, error)
	GenerateCitations(ctx context.Context, chunks []RetrievedChunk) ([]Citation, error)
}

// RetrievedChunk represents a chunk retrieved during RAG
type RetrievedChunk struct {
	ID          string
	Content     string
	DocumentID  DocumentID
	Ordinal     int
	Score       float64
	Metadata    map[string]interface{}
}

// Citation represents a source citation in an answer
type Citation struct {
	DocumentName string
	Ordinal      int
	Content      string
}

// ArtifactGenerator handles deterministic artifact creation
type ArtifactGenerator interface {
	GeneratePDF(ctx context.Context, data interface{}) (ArtifactID, error)
	GenerateExcel(ctx context.Context, data interface{}) (ArtifactID, error)
}

// ChannelAdapter handles external communication channels
type ChannelAdapter interface {
	SendMessage(ctx context.Context, sessionID SessionID, message string) error
	ReceiveMessage(ctx context.Context, sessionID SessionID) (*Message, error)
}

// TelegramAdapter implements Telegram channel integration
type TelegramAdapter struct {
	BotToken string
}

// WebWidgetAdapter implements web-based chat UI
type WebWidgetAdapter struct {
	APIKey string
}

// RateLimiter enforces per-tenant rate limits
type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID) (bool, error)
	Wait(ctx context.Context, tenantID TenantID) error
}

// AuditLogger logs write operations and state transitions
type AuditLogger interface {
	LogWriteOperation(ctx context.Context, operation string, details map[string]interface{}) error
	LogStateTransition(ctx context.Context, sessionID SessionID, from, to FSMState) error
}

// RiverWorker handles background job processing
type RiverWorker struct {
	Client *river.Client
	Pool   *pgxpool.Pool
}

// APIHandler manages REST API endpoints
type APIHandler struct {
	Router        ModelRouter
	ToolRegistry  ToolRegistry
	StateManager  StateManager
	KnowledgeBase KnowledgeBase
	RateLimiter   RateLimiter
	AuditLogger   AuditLogger
}

// WorkerPool manages concurrent job execution
type WorkerPool struct {
	MaxWorkers int
	Queue      *river.Client
	Pool       *pgxpool.Pool
}

// MetricsCollector gathers system metrics for Prometheus
type MetricsCollector interface {
	CollectLLMUsage(ctx context.Context, tenantID TenantID, promptTokens, completionTokens int) error
	CollectRetrievalLatency(ctx context.Context, duration time.Duration) error
	CollectQueueDepth(ctx context.Context, depth int) error
	CollectErrorRate(ctx context.Context, errType string) error
}

// Tracer handles distributed tracing with OpenTelemetry
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, *Span)
	EndSpan(ctx context.Context, span *Span)
}

// Span represents an OpenTelemetry trace span
type Span struct {
	Name string
	ID   string
}

// Config holds system configuration parameters
type Config struct {
	DatabaseURL      string
	RedisURL         string
	StorageBucket    string
	MaxConcurrency   int
	SessionTimeout   time.Duration
	RateLimit        int
	ModelRouter      ModelRouter
	ToolRegistry     ToolRegistry
	StateManager     StateManager
	KnowledgeBase    KnowledgeBase
	RateLimiter      RateLimiter
	AuditLogger      AuditLogger
	MetricsCollector MetricsCollector
	Tracer           Tracer
}
