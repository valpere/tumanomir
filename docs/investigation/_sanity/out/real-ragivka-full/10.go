package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
)

// Tenant represents a customer tenant with isolated data
type Tenant struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// User represents an end-user interacting via a channel
type User struct {
	ID            string    `json:"id"`
	TenantID      string    `json:"tenant_id"`
	ChannelType   string    `json:"channel_type"`
	ChannelID     string    `json:"channel_id"`
	CreatedAt     time.Time `json:"created_at"`
}

// SessionState represents the finite state of a conversation
type SessionState string

const (
	SessionActive       SessionState = "active"
	SessionWaitingHuman SessionState = "waiting_for_human"
	SessionCompleted    SessionState = "completed"
	SessionExpired      SessionState = "expired"
)

// Session represents a conversation with FSM state tracking
type Session struct {
	ID                string         `json:"id"`
	TenantID          string         `json:"tenant_id"`
	State             SessionState   `json:"state"`
	Version           int            `json:"version"`
	OrchestrationTier string         `json:"orchestration_tier"`
	Channel           string         `json:"channel"`
	ExpiresAt         time.Time      `json:"expires_at"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// Message represents a turn in the conversation
type Message struct {
	ID             string    `json:"id"`
	SessionID      string    `json:"session_id"`
	Role           string    `json:"role"`
	Content        string    `json:"content"`
	CitationRefs   []string  `json:"citation_refs"`
	TokenCount     int       `json:"token_count"`
	JobID          *string   `json:"job_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// RiverJob represents an asynchronous task in the queue
type RiverJob struct {
	ID                string    `json:"id"`
	SessionID         string    `json:"session_id"`
	TenantID          string    `json:"tenant_id"`
	IdempotencyKey    string    `json:"idempotency_key"`
	Payload           []byte    `json:"payload"`
	Attempt           int       `json:"attempt"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
	CompletedAt       *time.Time `json:"completed_at"`
	FailedAt          *time.Time `json:"failed_at"`
}

// AuditLog records write tool executions and state transitions
type AuditLog struct {
	ID               string    `json:"id"`
	ToolName         string    `json:"tool_name"`
	IdempotencyKey   string    `json:"idempotency_key"`
	RequestHash      string    `json:"request_hash"`
	ResponseHash     string    `json:"response_hash"`
	ApprovalRecord   *string   `json:"approval_record"`
	CreatedAt        time.Time `json:"created_at"`
}

// Document represents an uploaded file in the knowledge base
type Document struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	S3Key              string    `json:"s3_key"`
	Version            int       `json:"version"`
	IngestionStatus    string    `json:"ingestion_status"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

// Chunk represents a text segment from a document with vector embeddings
type Chunk struct {
	ID           string    `json:"id"`
	DocumentID   string    `json:"document_id"`
	Ordinal      int       `json:"ordinal"`
	Content      string    `json:"content"`
	Vector       []float32 `json:"vector"`
	TsVector     []string  `json:"ts_vector"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time `json:"created_at"`
}

// PromptVersion represents a version-controlled system prompt
type PromptVersion struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Version   int       `json:"version"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// Artifact represents a generated output file
type Artifact struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	S3Key      string    `json:"s3_key"`
	Type       string    `json:"type"`
	CreatedAt  time.Time `json:"created_at"`
}

// ModelRouter interface for routing requests to appropriate LLM providers
type ModelRouter interface {
	Route(ctx context.Context, task string) (string, error)
	GetProvider(ctx context.Context, model string) (string, error)
}

// ToolRegistry interface for managing registered tools
type ToolRegistry interface {
	RegisterTool(name string, tool interface{}) error
	GetTool(name string) (interface{}, bool)
	ValidatePermission(toolName string, permission string) error
}

// FSMStateTransition represents a state change in the conversation
type FSMStateTransition struct {
	SessionID   string    `json:"session_id"`
	FromState   SessionState `json:"from_state"`
	ToState     SessionState `json:"to_state"`
	Reason      string    `json:"reason"`
	TransitionedAt time.Time `json:"transitioned_at"`
}

// RAGPipeline interface for handling retrieval-augmented generation
type RAGPipeline interface {
	Retrieve(ctx context.Context, query string, tenantID string) ([]*Chunk, error)
	ReRank(ctx context.Context, chunks []*Chunk, query string) ([]*Chunk, error)
	GenerateCitations(ctx context.Context, answer string, chunks []*Chunk) ([]string, error)
}

// ChannelAdapter interface for different communication channels
type ChannelAdapter interface {
	HandleMessage(ctx context.Context, message *Message) error
	SendResponse(ctx context.Context, sessionID string, content string) error
	ProcessWebhook(ctx context.Context, payload []byte) error
}

// JobQueue interface for background job processing
type JobQueue interface {
	Enqueue(ctx context.Context, job *RiverJob) error
	ClaimJob(ctx context.Context) (*RiverJob, error)
	CompleteJob(ctx context.Context, jobID string) error
	FailJob(ctx context.Context, jobID string, err error) error
}

// DatabaseClient interface for database operations
type DatabaseClient interface {
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSession(ctx context.Context, session *Session) error
	GetMessage(ctx context.Context, messageID string) (*Message, error)
	InsertMessage(ctx context.Context, message *Message) error
	GetDocument(ctx context.Context, documentID string) (*Document, error)
	GetChunk(ctx context.Context, chunkID string) (*Chunk, error)
	GetPromptVersion(ctx context.Context, name string, version int) (*PromptVersion, error)
	GetTenant(ctx context.Context, tenantID string) (*Tenant, error)
	GetUser(ctx context.Context, userID string) (*User, error)
	InsertAuditLog(ctx context.Context, log *AuditLog) error
}

// MetricsCollector interface for collecting observability metrics
type MetricsCollector interface {
	RecordTokenUsage(ctx context.Context, promptTokens int, completionTokens int) error
	RecordRetrievalLatency(ctx context.Context, latency time.Duration) error
	RecordQueueDepth(ctx context.Context, depth int) error
	RecordErrorRate(ctx context.Context, errType string) error
}

// Tracer interface for distributed tracing
type Tracer interface {
	StartSpan(ctx context.Context, operationName string) (context.Context, interface{})
	EndSpan(span interface{}, err error) error
}

// Configuration struct for framework settings
type Configuration struct {
	DatabaseURL          string
	RedisURL             string
	ObjectStorageURL     string
	MaxConcurrency       int
	SessionTimeout       time.Duration
	RateLimitThreshold   int
	EnableTracing        bool
	EnableMetrics        bool
	ModelRouterConfig    map[string]interface{}
	ToolRegistryConfig   map[string]interface{}
}

// Framework represents the main Ragivka framework instance
type Framework struct {
	dbPool            *pgxpool.Pool
	jobQueue          JobQueue
	modelRouter       ModelRouter
	toolRegistry      ToolRegistry
	ragPipeline       RAGPipeline
	channelAdapters   map[string]ChannelAdapter
	metricsCollector  MetricsCollector
	tracer            Tracer
	config            *Configuration
}

// NewFramework creates a new instance of the Ragivka framework
func NewFramework(config *Configuration) (*Framework, error) {
	return &Framework{}, nil
}

// Start initializes and starts all framework components
func (f *Framework) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down all framework components
func (f *Framework) Stop(ctx context.Context) error {
	return nil
}

// ProcessMessage handles incoming messages from channels
func (f *Framework) ProcessMessage(ctx context.Context, message *Message) error {
	return nil
}

// ExecuteJob processes background jobs from the queue
func (f *Framework) ExecuteJob(ctx context.Context, job *RiverJob) error {
	return nil
}

// GenerateAnswer generates a response using RAG and LLM
func (f *Framework) GenerateAnswer(ctx context.Context, sessionID string, query string) (string, error) {
	return "", nil
}

// RunTool executes a registered tool with given arguments
func (f *Framework) RunTool(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	return nil, nil
}

// ValidateInput validates user input for prompt injection prevention
func (f *Framework) ValidateInput(ctx context.Context, input string) error {
	return nil
}

// GenerateArtifact creates deterministic output files like PDFs or Excel sheets
func (f *Framework) GenerateArtifact(ctx context.Context, sessionID string, format string, data interface{}) (string, error) {
	return "", nil
}
