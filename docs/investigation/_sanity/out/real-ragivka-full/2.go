package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
)

// Tenant represents a customer or organization using the system
type Tenant struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents an end-user interacting via a channel
type User struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ChannelType   string    `json:"channel_type"` // telegram/web
	ChannelID     string    `json:"channel_id"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// SessionState represents the FSM states for conversation management
type SessionState string

const (
	SessionActive        SessionState = "active"
	SessionWaitingForHuman SessionState = "waiting_for_human"
	SessionCompleted     SessionState = "completed"
	SessionExpired       SessionState = "expired"
)

// Session represents a conversation state machine
type Session struct {
	ID                string      `json:"id"`
	TenantID          string      `json:"tenant_id"`
	UserID            string      `json:"user_id"`
	State             SessionState `json:"state"`
	Version           int         `json:"version"`
	OrchestrationTier string      `json:"orchestration_tier"` // L0, L1, L2, L3
	Channel           string      `json:"channel"`
	ExpiresAt         time.Time   `json:"expires_at"`
	CreatedAt         time.Time   `json:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at"`
}

// Message represents an individual chat turn
type Message struct {
	ID           string    `json:"id"`
	SessionID    string    `json:"session_id"`
	Role         string    `json:"role"` // user/assistant/system
	Content      string    `json:"content"`
	CitationRefs []string  `json:"citation_refs"`
	TokenCount   int       `json:"token_count"`
	JobID        *string   `json:"job_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

// RiverJob represents an asynchronous job in the queue
type RiverJob struct {
	ID              string         `json:"id"`
	TenantID        string         `json:"tenant_id"`
	SessionID       string         `json:"session_id"`
	IdempotencyKey  string         `json:"idempotency_key"`
	Payload         map[string]any `json:"payload"`
	Attempt         int            `json:"attempt"`
	MaxAttempts     int            `json:"max_attempts"`
	Status          string         `json:"status"` // pending, running, completed, failed
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

// AuditLog records write tool executions and state transitions
type AuditLog struct {
	ID              string    `json:"id"`
	TenantID        string    `json:"tenant_id"`
	SessionID       string    `json:"session_id"`
	UserID          string    `json:"user_id"`
	ToolName        string    `json:"tool_name"`
	IdempotencyKey  string    `json:"idempotency_key"`
	RequestHash     string    `json:"request_hash"`
	ResponseHash    string    `json:"response_hash"`
	ApprovalRecord  *string   `json:"approval_record,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// Document represents a raw file uploaded to the knowledge layer
type Document struct {
	ID                string    `json:"id"`
	TenantID          string    `json:"tenant_id"`
	S3Key             string    `json:"s3_key"`
	Version           string    `json:"version"`
	IngestionStatus   string    `json:"ingestion_status"` // pending/indexed/stale
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// Chunk represents a text segment belonging to a document
type Chunk struct {
	ID           string    `json:"id"`
	DocumentID   string    `json:"document_id"`
	Ordinal      int       `json:"ordinal"`
	Content      string    `json:"content"`
	Vector       []float32 `json:"vector"`
	TsVector     string    `json:"ts_vector"`
	Metadata     map[string]any `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}

// PromptVersion represents version-controlled system prompts
type PromptVersion struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Artifact represents generated output files
type Artifact struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	S3Key      string    `json:"s3_key"`
	Type       string    `json:"type"` // pdf, excel
	CreatedAt  time.Time `json:"created_at"`
}

// ModelRouter interface for routing requests to appropriate LLM providers
type ModelRouter interface {
	Route(ctx context.Context, taskType string) (string, error)
}

// ToolRegistry interface for managing registered tools
type ToolRegistry interface {
	Register(tool Tool) error
	Get(name string) (Tool, error)
	List() []Tool
}

// Tool represents a function that can be invoked by the AI
type Tool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Permissions []string    `json:"permissions"` // read/draft/write
	Handler     func(ctx context.Context, args map[string]any) (map[string]any, error)
	CacheTTL    *time.Duration `json:"cache_ttl,omitempty"`
}

// ChannelAdapter interface for different communication channels
type ChannelAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	SendMessage(sessionID string, content string) error
	ParseMessage(message string) (*Message, error)
}

// KnowledgePipeline interface for ingestion and retrieval
type KnowledgePipeline interface {
	Ingest(ctx context.Context, document *Document) error
	Retrieve(ctx context.Context, query string, tenantID string) ([]*Chunk, error)
	ReRank(ctx context.Context, chunks []*Chunk, query string) ([]*Chunk, error)
}

// SessionManager interface for conversation lifecycle management
type SessionManager interface {
	CreateSession(ctx context.Context, userID string, channel string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSessionState(ctx context.Context, sessionID string, state SessionState) error
	ExpireSession(ctx context.Context, sessionID string) error
}

// AIService interface for AI operations
type AIService interface {
	GenerateAnswer(ctx context.Context, prompt string, contextChunks []*Chunk) (string, []string, error)
	ExtractStructuredData(ctx context.Context, prompt string, schema map[string]any) (map[string]any, error)
}

// JobQueue interface for background job processing
type JobQueue interface {
	Enqueue(ctx context.Context, job *RiverJob) error
	Worker() *river.Worker
}

// Config holds the application configuration
type Config struct {
	DatabaseURL      string
	RedisURL         string
	OpenAIKey        string
	AnthropicKey     string
	GoogleGeminiKey  string
	OllamaURL        string
	MaxConcurrency   int
	RateLimit        int
	SessionTimeout   time.Duration
}

// Framework represents the main Ragivka framework instance
type Framework struct {
	Config          *Config
	DB              *pgxpool.Pool
	JobQueue        JobQueue
	ModelRouter     ModelRouter
	ToolRegistry    ToolRegistry
	KnowledgePipeline KnowledgePipeline
	SessionManager  SessionManager
	AIService       AIService
	ChannelAdapters map[string]ChannelAdapter
}

// NewFramework creates a new framework instance
func NewFramework(config *Config) (*Framework, error) {
	return &Framework{}, nil
}

// Start initializes and starts the framework services
func (f *Framework) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down the framework services
func (f *Framework) Stop(ctx context.Context) error {
	return nil
}

// HandleMessage processes incoming messages from channels
func (f *Framework) HandleMessage(ctx context.Context, message *Message) error {
	return nil
}

// ExecuteJob runs a queued background job
func (f *Framework) ExecuteJob(ctx context.Context, job *RiverJob) error {
	return nil
}
