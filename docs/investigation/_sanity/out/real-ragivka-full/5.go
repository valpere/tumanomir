package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
)

// Tenant represents a tenant in the system
type Tenant struct {
	ID          string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Name        string
	IsolationKey string
}

// User represents a user interacting via a channel
type User struct {
	ID         string
	TenantID   string
	Channel    string // telegram/web
	ChannelID  string // e.g., Telegram user ID
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// SessionState represents the state of a conversation session
type SessionState string

const (
	SessionActive       SessionState = "active"
	SessionWaitingForHuman SessionState = "waiting_for_human"
	SessionCompleted    SessionState = "completed"
	SessionExpired      SessionState = "expired"
)

// Session represents a conversation session with FSM
type Session struct {
	ID                 string
	TenantID           string
	UserID             string
	State              SessionState
	Version            int64
	OrchestrationTier  string // L0/L1/L2/L3
	Channel            string
	ExpiresAt          time.Time
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Message represents a single chat turn
type Message struct {
	ID             string
	SessionID      string
	Role           string // user/assistant/system
	Content        string
	CitationRefs   []string
	TokenCount     int
	JobID          *string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// RiverJob represents an asynchronous job in the queue
type RiverJob struct {
	ID                string
	TenantID          string
	SessionID         string
	IdempotencyKey    string
	Payload           map[string]interface{}
	Attempt           int
	MaxAttempts       int
	Status            string
	UpdatedAt         time.Time
	CreatedAt         time.Time
}

// AuditLog records write tool executions and FSM transitions
type AuditLog struct {
	ID                 string
	TenantID           string
	UserID             *string
	SessionID          string
	ToolName           string
	IdempotencyKey     string
	RequestHash        string
	ResponseHash       string
	ApprovalRecord     *string
	CreatedAt          time.Time
}

// Document represents an uploaded raw file in knowledge base
type Document struct {
	ID                  string
	TenantID            string
	S3Key               string
	Version             int64
	IngestionStatus     string // pending/indexed/stale
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// Chunk represents a text segment belonging to a document
type Chunk struct {
	ID              string
	DocumentID      string
	OrdinalPosition int
	Content         string
	Vector          []float32
	TsVector        string // BM25
	Metadata        map[string]interface{}
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// PromptVersion represents version-controlled system prompts
type PromptVersion struct {
	ID        string
	Name      string
	Version   int64
	Content   string
	CreatedAt time.Time
}

// Artifact represents generated output files (PDF, Excel)
type Artifact struct {
	ID         string
	SessionID  string
	S3Key      string
	Type       string // pdf/excel
	CreatedAt  time.Time
}

// ModelProvider represents supported LLM providers
type ModelProvider string

const (
	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderGemini    ModelProvider = "gemini"
	ProviderLocal     ModelProvider = "local"
)

// ToolPermission represents permission levels for tools
type ToolPermission string

const (
	ToolRead   ToolPermission = "read"
	ToolDraft  ToolPermission = "draft"
	ToolWrite  ToolPermission = "write"
)

// ToolRegistry represents a registered tool
type ToolRegistry struct {
	Name             string
	Provider         ModelProvider
	Permission       ToolPermission
	Schema           map[string]interface{}
	CacheTTL         *time.Duration
	IsIdempotent     bool
	RequiresApproval bool
	CreatedAt        time.Time
}

// ChannelAdapter represents a communication channel adapter
type ChannelAdapter interface {
	HandleMessage(ctx context.Context, sessionID string, message string) error
	SendResponse(ctx context.Context, sessionID string, response string) error
	NotifyHuman(ctx context.Context, sessionID string, message string) error
}

// ModelRouter routes requests to appropriate LLM providers
type ModelRouter interface {
	GetModelForTask(taskType string) (string, error)
	FallbackProvider() ModelProvider
}

// PromptRegistry manages system prompts
type PromptRegistry interface {
	GetPrompt(ctx context.Context, name string, version int64) (string, error)
	StorePrompt(ctx context.Context, name string, version int64, content string) error
}

// StructuredOutputParser parses LLM responses into structured data
type StructuredOutputParser interface {
	Parse(ctx context.Context, response string, schema interface{}) (interface{}, error)
}

// GuardrailEvaluator evaluates LLM outputs for hallucinations and citations
type GuardrailEvaluator interface {
	Evaluate(ctx context.Context, answer string, retrievedChunks []string) (bool, string, error)
}

// RAGPipeline handles ingestion and retrieval
type RAGPipeline interface {
	IngestDocument(ctx context.Context, tenantID string, document Document) error
	RetrieveContext(ctx context.Context, tenantID string, query string, limit int) ([]Chunk, error)
	ReRank(ctx context.Context, chunks []Chunk, query string) ([]Chunk, error)
}

// ToolLayer manages tool execution and safety boundaries
type ToolLayer interface {
	RegisterTool(ctx context.Context, tool ToolRegistry) error
	ExecuteReadTool(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error)
	ExecuteDraftTool(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error)
	ExecuteWriteTool(ctx context.Context, toolName string, args map[string]interface{}) (map[string]interface{}, error)
}

// FSMStateHandler manages session state transitions
type FSMStateHandler interface {
	GetSession(ctx context.Context, sessionID string) (*Session, error)
	UpdateSessionState(ctx context.Context, sessionID string, newState SessionState, version int64) error
	TransitionToHuman(ctx context.Context, sessionID string, reason string) error
	AutoExpireSession(ctx context.Context, sessionID string) error
}

// RiverWorker manages background job processing
type RiverWorker interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	QueueJob(ctx context.Context, job *river.Job) error
}

// APIServer handles incoming requests and routing
type APIServer interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
	RegisterHandler(pattern string, handler func(http.ResponseWriter, *http.Request)) error
}

// Config holds application configuration
type Config struct {
	DatabaseURL      string
	RedisURL         string
	APIPort          int
	WorkerMode       bool
	OfflineMode      bool
	EmbeddingModel   string
	LLMProvider      ModelProvider
	RateLimit        int
	SessionTimeout   time.Duration
}

// Service represents the main application service
type Service struct {
	Config           *Config
	DBPool           *pgxpool.Pool
	RiverClient      *river.Client
	ChannelAdapters  map[string]ChannelAdapter
	ModelRouter      ModelRouter
	PromptRegistry   PromptRegistry
	StructuredParser StructuredOutputParser
	Guardrail        GuardrailEvaluator
	RAGPipeline      RAGPipeline
	ToolLayer        ToolLayer
	FSMHandler       FSMStateHandler
	Worker           RiverWorker
	APIServer        APIServer
}

// NewService creates a new Ragivka service instance
func NewService(config *Config) (*Service, error) {
	return &Service{}, nil
}

// Start initializes and starts the service
func (s *Service) Start(ctx context.Context) error {
	return nil
}

// Stop shuts down the service
func (s *Service) Stop(ctx context.Context) error {
	return nil
}

// GetSession retrieves a session by ID
func (s *Service) GetSession(ctx context.Context, sessionID string) (*Session, error) {
	return &Session{}, nil
}

// CreateSession creates a new conversation session
func (s *Service) CreateSession(ctx context.Context, tenantID string, userID string) (*Session, error) {
	return &Session{}, nil
}

// ProcessMessage handles user messages and orchestrates responses
func (s *Service) ProcessMessage(ctx context.Context, sessionID string, message string) error {
	return nil
}

// ExecuteTool executes a registered tool with given arguments
func (s *Service) ExecuteTool(ctx context.Context, sessionID string, toolName string, args map[string]interface{}) (map[string]interface{}, error) {
	return map[string]interface{}{}, nil
}

// IngestDocument processes document ingestion into knowledge base
func (s *Service) IngestDocument(ctx context.Context, tenantID string, document Document) error {
	return nil
}

// RetrieveContext fetches relevant context for a query
func (s *Service) RetrieveContext(ctx context.Context, tenantID string, query string, limit int) ([]Chunk, error) {
	return []Chunk{}, nil
}

// EvaluateGuardrail checks if LLM output is acceptable
func (s *Service) EvaluateGuardrail(ctx context.Context, answer string, retrievedChunks []string) (bool, string, error) {
	return true, "", nil
}

// RegisterTool registers a new tool in the system
func (s *Service) RegisterTool(ctx context.Context, tool ToolRegistry) error {
	return nil
}

// GetPrompt retrieves a prompt by name and version
func (s *Service) GetPrompt(ctx context.Context, name string, version int64) (string, error) {
	return "", nil
}

// ParseStructuredOutput parses LLM output into structured format
func (s *Service) ParseStructuredOutput(ctx context.Context, response string, schema interface{}) (interface{}, error) {
	return nil, nil
}
