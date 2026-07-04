package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/trace"
)

// Tenant represents a multi-tenant boundary
type Tenant struct {
	ID           string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	Name         string
	IsolationKey string
}

// User represents an end-user in a channel
type User struct {
	ID          string
	TenantID    string
	ChannelType string // telegram, web
	ChannelID   string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// SessionState represents the FSM states
type SessionState string

const (
	SessionActive       SessionState = "active"
	SessionWaitingForHuman SessionState = "waiting_for_human"
	SessionCompleted    SessionState = "completed"
	SessionExpired      SessionState = "expired"
)

// Session represents a conversation session with FSM state
type Session struct {
	ID                  string
	TenantID            string
	UserID              string
	State               SessionState
	Version             int64
	OrchestrationTier   string // L0, L1, L2, L3
	Channel             string
	ExpiresAt           time.Time
	CreatedAt           time.Time
	UpdatedAt           time.Time
	InactivityTimeout   time.Duration
}

// Message represents a chat turn in a session
type Message struct {
	ID              string
	SessionID       string
	Role            string // user, assistant, system
	Content         string
	CitationRefs    []string
	TokenCount      int
	JobID           *string
	CreatedAt       time.Time
}

// RiverJob represents an async job in the queue
type RiverJob struct {
	ID               string
	SessionID        string
	TenantID         string
	IdempotencyKey   string
	Payload          map[string]interface{}
	Attempt          int
	Status           string
	MaxAttempts      int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	StartedAt        *time.Time
	CompletedAt      *time.Time
	FailedAt         *time.Time
	LastError        *string
}

// AuditLog records write tool executions and state transitions
type AuditLog struct {
	ID                 string
	SessionID          string
	UserID             string
	ToolName           string
	IdempotencyKey     string
	RequestHash        string
	ResponseHash       string
	ApprovalRecord     *string
	CreatedAt          time.Time
}

// Document represents an ingested file
type Document struct {
	ID                string
	TenantID          string
	S3Key             string
	Version           int
	IngestionStatus   string // pending, indexed, stale
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// Chunk represents a text segment from a document
type Chunk struct {
	ID              string
	DocumentID      string
	OrdinalPosition int
	Content         string
	Vector          []float32
	TsVector        string
	Metadata        map[string]interface{}
	CreatedAt       time.Time
}

// PromptVersion represents a version-controlled system prompt
type PromptVersion struct {
	ID        string
	Name      string
	Version   string
	Content   string
	CreatedAt time.Time
}

// Artifact represents generated output files
type Artifact struct {
	ID        string
	SessionID string
	S3Key     string
	Type      string // pdf, excel
	CreatedAt time.Time
}

// ToolPermission represents read/draft/write permissions
type ToolPermission string

const (
	ToolRead   ToolPermission = "read"
	ToolDraft  ToolPermission = "draft"
	ToolWrite  ToolPermission = "write"
)

// Tool represents a registered function with permission boundaries
type Tool struct {
	Name           string
	Permission     ToolPermission
	Description    string
	Schema         map[string]interface{}
	CacheTTL       *time.Duration
	IsCachable     bool
	CreatedAt      time.Time
}

// ModelProvider represents LLM provider types
type ModelProvider string

const (
	ModelOpenAI   ModelProvider = "openai"
	ModelAnthropic ModelProvider = "anthropic"
	ModelGemini   ModelProvider = "gemini"
	ModelOllama   ModelProvider = "ollama"
)

// ModelRouter routes requests to appropriate LLMs based on task complexity
type ModelRouter interface {
	GetModel(ctx context.Context, task string, tenantID string) (string, error)
}

// PromptRegistry stores and retrieves system prompts by version
type PromptRegistry interface {
	GetPrompt(ctx context.Context, name, version string) (string, error)
}

// ToolRegistry manages registered tools with permission checks
type ToolRegistry interface {
	RegisterTool(ctx context.Context, tool Tool) error
	GetTool(ctx context.Context, name string) (*Tool, error)
	ExecuteTool(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error)
}

// SessionManager handles session lifecycle and state transitions
type SessionManager interface {
	CreateSession(ctx context.Context, tenantID, userID, channel string) (*Session, error)
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSessionState(ctx context.Context, sessionID string, newState SessionState) error
	ResumeSession(ctx context.Context, sessionID string) (*Session, error)
	ExpireSession(ctx context.Context, sessionID string) error
}

// RAGPipeline handles document ingestion and retrieval
type RAGPipeline interface {
	IngestDocument(ctx context.Context, tenantID, s3Key string) error
	RetrieveContext(ctx context.Context, sessionID, query string) ([]*Chunk, error)
	ReRankResults(ctx context.Context, chunks []*Chunk, query string) ([]*Chunk, error)
}

// AIEngine handles LLM interactions and structured output parsing
type AIEngine interface {
	GenerateResponse(ctx context.Context, sessionID, prompt string, chunks []*Chunk) (map[string]interface{}, error)
	ValidateOutput(ctx context.Context, output map[string]interface{}, schema map[string]interface{}) error
}

// ChannelAdapter handles communication with external channels
type ChannelAdapter interface {
	HandleMessage(ctx context.Context, tenantID, channel, message string) error
	SendResponse(ctx context.Context, sessionID, response string) error
}

// WorkerPool manages background job processing
type WorkerPool struct {
	Pool *river.Client
}

// FSMStateTransition represents a state change in the conversation
type FSMStateTransition struct {
	SessionID   string
	OldState    SessionState
	NewState    SessionState
	TransitionAt time.Time
	UserID      *string
}

// Config holds framework configuration
type Config struct {
	DatabaseURL        string
	RedisURL           string
	ObjectStorageURL   string
	TracingEnabled     bool
	TraceProvider      trace.TracerProvider
	MaxConcurrency     int
	RateLimitThreshold int
}

// Repository interfaces for data access
type TenantRepository interface {
	GetTenant(ctx context.Context, id string) (*Tenant, error)
	CreateTenant(ctx context.Context, tenant *Tenant) error
	UpdateTenant(ctx context.Context, tenant *Tenant) error
}

type UserRepository interface {
	GetUser(ctx context.Context, id string) (*User, error)
	CreateUser(ctx context.Context, user *User) error
	UpdateUser(ctx context.Context, user *User) error
}

type SessionRepository interface {
	GetSession(ctx context.Context, id string) (*Session, error)
	CreateSession(ctx context.Context, session *Session) error
	UpdateSession(ctx context.Context, session *Session) error
	DeleteSession(ctx context.Context, id string) error
	ListSessionsByTenant(ctx context.Context, tenantID string) ([]*Session, error)
}

type MessageRepository interface {
	GetMessages(ctx context.Context, sessionID string) ([]*Message, error)
	CreateMessage(ctx context.Context, message *Message) error
	UpdateMessage(ctx context.Context, message *Message) error
}

type DocumentRepository interface {
	GetDocument(ctx context.Context, id string) (*Document, error)
	CreateDocument(ctx context.Context, document *Document) error
	UpdateDocument(ctx context.Context, document *Document) error
}

type ChunkRepository interface {
	GetChunksByDocument(ctx context.Context, documentID string) ([]*Chunk, error)
	CreateChunk(ctx context.Context, chunk *Chunk) error
	UpdateChunk(ctx context.Context, chunk *Chunk) error
	SearchChunks(ctx context.Context, tenantID, query string, limit int) ([]*Chunk, error)
}

type AuditLogRepository interface {
	GetAuditLogsBySession(ctx context.Context, sessionID string) ([]*AuditLog, error)
	CreateAuditLog(ctx context.Context, log *AuditLog) error
	UpdateAuditLog(ctx context.Context, log *AuditLog) error
}

type RiverJobRepository interface {
	GetJob(ctx context.Context, id string) (*RiverJob, error)
	CreateJob(ctx context.Context, job *RiverJob) error
	UpdateJob(ctx context.Context, job *RiverJob) error
	QueueJob(ctx context.Context, job *RiverJob) error
	CompleteJob(ctx context.Context, id string) error
	FailJob(ctx context.Context, id string, err error) error
}

// Framework provides core services
type Framework struct {
	TenantRepo          TenantRepository
	UserRepo            UserRepository
	SessionRepo         SessionRepository
	MessageRepo         MessageRepository
	DocumentRepo        DocumentRepository
	ChunkRepo           ChunkRepository
	AuditLogRepo        AuditLogRepository
	RiverJobRepo        RiverJobRepository
	ModelRouter         ModelRouter
	PromptRegistry      PromptRegistry
	ToolRegistry        ToolRegistry
	SessionManager      SessionManager
	RAGPipeline         RAGPipeline
	AIEngine            AIEngine
	ChannelAdapter      ChannelAdapter
	WorkerPool          *WorkerPool
	DBPool              *pgxpool.Pool
	TraceProvider       trace.TracerProvider
	Config              *Config
}

// Service interfaces
type APIService interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

type WorkerService interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// HTTPError represents structured API errors
type HTTPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

// MetricCollector collects Prometheus metrics
type MetricCollector interface {
	RecordLLMUsage(ctx context.Context, tenantID, model string, promptTokens, completionTokens int)
	RecordRetrievalLatency(ctx context.Context, tenantID, query string, latency time.Duration)
	RecordQueueDepth(ctx context.Context, queueName string, depth int)
	RecordErrorRate(ctx context.Context, errorType string, count int)
}

// EvaluationHook provides offline evaluation capabilities
type EvaluationHook interface {
	EvaluateRetrievalRecall(ctx context.Context, sessionID string, k int) (float64, error)
	EvaluateCitationCoverage(ctx context.Context, sessionID string) (float64, error)
	RecordGroundednessTest(ctx context.Context, sessionID string, passed bool) error
}

// BackupManager handles database backup and recovery
type BackupManager interface {
	CreateBackup(ctx context.Context) error
	RestoreFromBackup(ctx context.Context, backupID string) error
	EnableContinuousArchiving(ctx context.Context) error
}

// DisasterRecoveryManager handles RTO/RPO requirements
type DisasterRecoveryManager interface {
	PrepareDisasterRecovery(ctx context.Context) error
	ActivateDisasterRecovery(ctx context.Context) error
	RestoreFromDRP(ctx context.Context) error
}
