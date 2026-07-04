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

// ToolName identifies a registered tool
type ToolName string

// ModelProvider represents supported LLM providers
type ModelProvider string

const (
	ProviderOpenAI   ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderGemini   ModelProvider = "gemini"
	ProviderLocal    ModelProvider = "local"
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

// Tool represents a registered tool with its metadata and permissions
type Tool struct {
	Name        ToolName
	Permissions []ToolPermission
	Handler     func(context.Context, map[string]interface{}) (interface{}, error)
}

// Session represents a conversation session
type Session struct {
	ID         SessionID
	TenantID   TenantID
	State      FSMState
	Version    int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	ExpiresAt  time.Time
	LastActive time.Time
}

// Message represents a message in a conversation
type Message struct {
	ID        string
	SessionID SessionID
	Content   string
	Role      string // "user", "assistant", "system"
	CreatedAt time.Time
}

// Document represents an ingested document
type Document struct {
	ID          DocumentID
	TenantID    TenantID
	Name        string
	Size        int64
	MimeType    string
	ContentType string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// Artifact represents a generated file
type Artifact struct {
	ID         ArtifactID
	DocumentID DocumentID
	Name       string
	Size       int64
	MimeType   string
	URL        string
	CreatedAt  time.Time
}

// Chunk represents a semantic chunk of text
type Chunk struct {
	ID          string
	DocumentID  DocumentID
	Content     string
	Ordinal     int
	StartPosition int
	EndPosition   int
	Vector      []float32
	CreatedAt   time.Time
}

// RetrievedChunk represents a chunk retrieved during RAG
type RetrievedChunk struct {
	Chunk
	Score float32
}

// PromptVersion represents a versioned system prompt
type PromptVersion struct {
	Name    string
	Version string
	Content string
	CreatedAt time.Time
}

// Job represents a background job processed by River
type Job struct {
	ID        string
	Type      string
	Payload   map[string]interface{}
	State     river.JobState
	CreatedAt time.Time
	UpdatedAt time.Time
	Attempts  int
}

// AuditLogEntry represents an audit log entry for write operations
type AuditLogEntry struct {
	IDempotencyKey string
	ToolName       ToolName
	RequestHash    string
	ResponseHash   string
	CreatedAt      time.Time
}

// ModelRouter routes requests to appropriate LLM providers
type ModelRouter interface {
	Route(ctx context.Context, taskType string) (ModelProvider, error)
}

// ToolRegistry manages registered tools and their permissions
type ToolRegistry interface {
	Register(tool *Tool) error
	Get(name ToolName) (*Tool, error)
	ValidatePermissions(name ToolName, required []ToolPermission) error
}

// SessionManager handles session lifecycle and state transitions
type SessionManager interface {
	Create(ctx context.Context, tenantID TenantID) (*Session, error)
	Get(ctx context.Context, id SessionID) (*Session, error)
	UpdateState(ctx context.Context, id SessionID, newState FSMState, version int64) error
	TransitionToWaitingForHuman(ctx context.Context, id SessionID) error
	ExpireInactiveSessions(ctx context.Context, timeout time.Duration) error
}

// KnowledgeBase manages document ingestion and retrieval
type KnowledgeBase interface {
	IngestDocument(ctx context.Context, tenantID TenantID, doc *Document) error
	Retrieve(ctx context.Context, tenantID TenantID, query string, topK int) ([]RetrievedChunk, error)
	GetDocumentChunks(ctx context.Context, docID DocumentID) ([]Chunk, error)
}

// ArtifactGenerator generates deterministic artifacts from LLM outputs
type ArtifactGenerator interface {
	GeneratePDF(ctx context.Context, content string, metadata map[string]interface{}) (*Artifact, error)
	GenerateExcel(ctx context.Context, data [][]interface{}, headers []string) (*Artifact, error)
}

// ChannelAdapter handles communication with external channels like Telegram or Web Widget
type ChannelAdapter interface {
	SendMessage(ctx context.Context, sessionID SessionID, message string) error
	HandleWebhook(ctx context.Context, payload map[string]interface{}) error
}

// RateLimiter enforces rate limits per tenant using Redis sliding window
type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID, limit int, window time.Duration) (bool, error)
}

// Tracer handles OpenTelemetry tracing
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, *Span)
}

// MetricsCollector tracks system metrics for Prometheus
type MetricsCollector interface {
	RecordTokenUsage(tenantID TenantID, promptTokens, completionTokens int)
	RecordRetrievalLatency(latency time.Duration)
	RecordQueueDepth(depth int64)
	RecordError(name string)
}

// CostTracker logs per-request token costs for tenant attribution
type CostTracker interface {
	LogCost(ctx context.Context, tenantID TenantID, promptTokens, completionTokens int, cost float64) error
}

// Configuration holds system-wide configuration values
type Configuration struct {
	DatabaseURL           string
	RedisURL              string
	ObjectStorageEndpoint string
	MaxConcurrency        int
	SessionTimeout        time.Duration
	RateLimit             int
	ChunkSize             int
	Overlap               float32
	ReRankTopK            int
}

// Service represents the main Ragivka service
type Service struct {
	db                  *pgxpool.Pool
	riverClient         *river.Client
	sessionManager      SessionManager
	knowledgeBase       KnowledgeBase
	toolRegistry        ToolRegistry
	modelRouter         ModelRouter
	artifactGenerator   ArtifactGenerator
	channelAdapter      ChannelAdapter
	rateLimiter         RateLimiter
	tracer              Tracer
	metricsCollector    MetricsCollector
	costTracker         CostTracker
}

// NewService creates a new Ragivka service instance
func NewService(config *Configuration) (*Service, error) {
	return &Service{}, nil
}

// Start initializes the service and begins processing jobs
func (s *Service) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down the service gracefully
func (s *Service) Stop(ctx context.Context) error {
	return nil
}

// HandleRequest processes an incoming request through the RAG pipeline
func (s *Service) HandleRequest(ctx context.Context, tenantID TenantID, sessionID SessionID, message string) (*Session, error) {
	return &Session{}, nil
}

// ProcessJob processes a background job from River
func (s *Service) ProcessJob(ctx context.Context, job *Job) error {
	return nil
}

// RegisterTool registers a new tool with the system
func (s *Service) RegisterTool(ctx context.Context, tool *Tool) error {
	return nil
}

// GetPrompt retrieves a versioned prompt from storage
func (s *Service) GetPrompt(ctx context.Context, name, version string) (*PromptVersion, error) {
	return &PromptVersion{}, nil
}

// GetSessionHistory returns conversation history for a session
func (s *Service) GetSessionHistory(ctx context.Context, sessionID SessionID) ([]Message, error) {
	return []Message{}, nil
}

// GenerateArtifact generates an artifact from structured data
func (s *Service) GenerateArtifact(ctx context.Context, sessionID SessionID, artifactType string, data interface{}) (*Artifact, error) {
	return &Artifact{}, nil
}

// Span represents an OpenTelemetry span
type Span struct {
	Name string
}

// StartSpan starts a new tracing span
func (t *Tracer) StartSpan(ctx context.Context, name string) (context.Context, *Span) {
	return ctx, &Span{Name: name}
}
