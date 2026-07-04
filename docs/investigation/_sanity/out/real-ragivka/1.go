package ragivka

import (
	"context"
	"time"

	"github.com/riverqueue/river"
	"github.com/uptrace/bun"
)

// Tenant represents a multi-tenant isolation boundary
type Tenant struct {
	ID     string `bun:"id"`
	Name   string `bun:"name"`
	APIKey string `bun:"api_key"`
}

// Session represents a conversation session with FSM state
type Session struct {
	ID            string    `bun:"id"`
	TenantID      string    `bun:"tenant_id"`
	State         SessionState `bun:"state"`
	Version       int       `bun:"version"`
	ExpiresAt     time.Time `bun:"expires_at"`
	CreatedAt     time.Time `bun:"created_at"`
	UpdatedAt     time.Time `bun:"updated_at"`
}

type SessionState string

const (
	SessionActive         SessionState = "active"
	SessionWaitingForHuman SessionState = "waiting_for_human"
	SessionCompleted      SessionState = "completed"
	SessionExpired        SessionState = "expired"
)

// Message represents a turn in the conversation
type Message struct {
	ID         string    `bun:"id"`
	SessionID  string    `bun:"session_id"`
	Role       string    `bun:"role"` // "user", "assistant"
	Content    string    `bun:"content"`
	CreatedAt  time.Time `bun:"created_at"`
}

// Document represents an ingested document
type Document struct {
	ID           string    `bun:"id"`
	TenantID     string    `bun:"tenant_id"`
	Name         string    `bun:"name"`
	Size         int64     `bun:"size"`
	ContentType  string    `bun:"content_type"`
	CreatedAt    time.Time `bun:"created_at"`
	UpdatedAt    time.Time `bun:"updated_at"`
}

// Chunk represents a chunked portion of a document
type Chunk struct {
	ID           string    `bun:"id"`
	DocumentID   string    `bun:"document_id"`
	Ordinal      int       `bun:"ordinal"`
	Content      string    `bun:"content"`
	Metadata     string    `bun:"metadata"` // JSON encoded
	Vector       []float32 `bun:"vector"`
	CreatedAt    time.Time `bun:"created_at"`
}

// Tool represents a registered tool with permissions
type Tool struct {
	Name        string `bun:"name"`
	Description string `bun:"description"`
	Permissions []ToolPermission `bun:"permissions"` // read, draft, write
	Schema      string `bun:"schema"` // JSON schema for arguments
}

type ToolPermission string

const (
	ToolRead   ToolPermission = "read"
	ToolDraft  ToolPermission = "draft"
	ToolWrite  ToolPermission = "write"
)

// Job represents a background task
type Job struct {
	ID        string    `bun:"id"`
	TenantID  string    `bun:"tenant_id"`
	Type      string    `bun:"type"`
	Payload   []byte    `bun:"payload"`
	State     JobState  `bun:"state"`
	Retries   int       `bun:"retries"`
	CreatedAt time.Time `bun:"created_at"`
	UpdatedAt time.Time `bun:"updated_at"`
}

type JobState string

const (
	JobPending   JobState = "pending"
	JobRunning   JobState = "running"
	JobCompleted JobState = "completed"
	JobFailed    JobState = "failed"
)

// AuditLog records write tool executions and FSM transitions
type AuditLog struct {
	ID              string    `bun:"id"`
	TenantID        string    `bun:"tenant_id"`
	IdempotencyKey  string    `bun:"idempotency_key"`
	ToolName        string    `bun:"tool_name"`
	Action          string    `bun:"action"` // "execute", "transition"
	RequestHash     string    `bun:"request_hash"`
	ResponseHash    string    `bun:"response_hash"`
	CreatedAt       time.Time `bun:"created_at"`
}

// Prompt represents a versioned system prompt
type Prompt struct {
	Name      string    `bun:"name"`
	Version   int       `bun:"version"`
	Content   string    `bun:"content"`
	CreatedAt time.Time `bun:"created_at"`
}

// Artifact represents generated content stored in object storage
type Artifact struct {
	ID           string    `bun:"id"`
	TenantID     string    `bun:"tenant_id"`
	Name         string    `bun:"name"`
	Size         int64     `bun:"size"`
	ContentType  string    `bun:"content_type"`
	URL          string    `bun:"url"`
	CreatedAt    time.Time `bun:"created_at"`
}

// RateLimitBucket represents a sliding window rate limit
type RateLimitBucket struct {
	TenantID string    `bun:"tenant_id"`
	Key      string    `bun:"key"`
	Limit    int       `bun:"limit"`
	Window   time.Duration `bun:"window"`
	Current  int       `bun:"current"`
	ResetAt  time.Time `bun:"reset_at"`
}

// ModelRouter routes requests to appropriate LLM providers
type ModelRouter interface {
	Route(ctx context.Context, task string) (string, error)
	GetModelProvider(modelName string) (ModelProvider, error)
}

// ModelProvider interface for different LLM backends
type ModelProvider interface {
	Generate(ctx context.Context, prompt string, options *GenerationOptions) (*GenerationResult, error)
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// GenerationOptions holds parameters for model generation
type GenerationOptions struct {
	MaxTokens     int
	Temperature   float64
	TopP          float64
	StopSequences []string
}

// GenerationResult holds the output of a model generation
type GenerationResult struct {
	Text      string
	TokenUsage TokenUsage
}

// TokenUsage tracks token consumption
type TokenUsage struct {
	PromptTokens int
	CompletionTokens int
	TotalTokens  int
	Cost         float64 // USD
}

// ToolRegistry manages registered tools
type ToolRegistry interface {
	Register(tool *Tool) error
	Get(name string) (*Tool, error)
	List() []*Tool
}

// FSMManager handles session state transitions
type FSMManager interface {
	Transition(sessionID string, newState SessionState, version int) error
	WaitForHuman(sessionID string, version int) error
	Complete(sessionID string, version int) error
	Expire(sessionID string, version int) error
}

// RAGService handles retrieval-augmented generation pipeline
type RAGService interface {
	Retrieve(ctx context.Context, tenantID string, query string, k int) ([]*Chunk, error)
	Rerank(ctx context.Context, chunks []*Chunk, query string) ([]*Chunk, error)
	GenerateAnswer(ctx context.Context, tenantID string, query string, contextChunks []*Chunk) (*GenerationResult, error)
}

// ChannelAdapter handles external communication channels
type ChannelAdapter interface {
	HandleMessage(ctx context.Context, tenant *Tenant, message *Message) error
	SendResponse(ctx context.Context, sessionID string, response *Message) error
}

// TelegramAdapter implements ChannelAdapter for Telegram
type TelegramAdapter struct{}

// WebWidgetAdapter implements ChannelAdapter for web chat
type WebWidgetAdapter struct{}

// JobQueue handles background job processing with River
type JobQueue interface {
	Enqueue(ctx context.Context, job *river.Job) error
	Worker() *river.Worker
}

// Service encapsulates all framework services
type Service struct {
	Database      *bun.DB
	JobQueue      JobQueue
	ModelRouter   ModelRouter
	ToolRegistry  ToolRegistry
	FSMManager    FSMManager
	RAGService    RAGService
	ChannelAdapters map[string]ChannelAdapter
}

// Initialize initializes the Ragivka framework with given options
func (s *Service) Initialize(ctx context.Context, opts *ServiceOptions) error {
	return nil
}

type ServiceOptions struct {
	DatabaseURL     string
	RedisURL        string
	ObjectStorage   ObjectStorage
	ModelProviders  []ModelProvider
	ToolRegistry    ToolRegistry
	JobQueue        JobQueue
}

// ObjectStorage interface for storing artifacts and documents
type ObjectStorage interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

// SessionManager handles session lifecycle and concurrency
type SessionManager interface {
	CreateSession(ctx context.Context, tenantID string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, sessionID string) error
}

// ConversationManager handles conversation history and limits
type ConversationManager interface {
	AddMessage(ctx context.Context, sessionID string, message *Message) error
	GetConversationHistory(ctx context.Context, sessionID string, limit int) ([]*Message, error)
	TrimHistory(ctx context.Context, sessionID string, maxTurns int) error
}

// IngestionPipeline manages document ingestion and chunking
type IngestionPipeline interface {
	IngestDocument(ctx context.Context, tenantID string, document *Document) error
	ChunkDocument(ctx context.Context, document *Document) ([]*Chunk, error)
	CleanupStaleChunks(ctx context.Context, tenantID string) error
}

// MetricsCollector gathers system metrics for Prometheus
type MetricsCollector interface {
	RecordTokenUsage(tenantID string, usage TokenUsage)
	RecordRetrievalLatency(latency time.Duration)
	RecordQueueDepth(depth int)
	RecordError(errorType string)
}
