package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/trace"
)

type TenantID string
type SessionID string
type DocumentID string
type ArtifactID string
type ToolName string
type PromptVersion string
type LLMProvider string
type JobID string

type FSMState string

const (
	FSMStateActive         FSMState = "active"
	FSMStateWaitingForHuman FSMState = "waiting_for_human"
	FSMStateCompleted      FSMState = "completed"
	FSMStateExpired        FSMState = "expired"
)

type ToolPermission string

const (
	ToolPermissionRead   ToolPermission = "read"
	ToolPermissionDraft  ToolPermission = "draft"
	ToolPermissionWrite  ToolPermission = "write"
)

type Tool struct {
	Name    ToolName
	Perms   []ToolPermission
	Handler func(context.Context, any) (any, error)
}

type ModelRouter interface {
	Route(context.Context, string) (LLMProvider, error)
}

type PromptRegistry interface {
	GetPrompt(ctx context.Context, name string, version PromptVersion) (string, error)
}

type ToolRegistry interface {
	Register(tool Tool) error
	Get(name ToolName) (*Tool, error)
	Permissions(name ToolName) []ToolPermission
}

type SessionManager interface {
	CreateSession(ctx context.Context, tenantID TenantID) (SessionID, error)
	GetSession(ctx context.Context, sessionID SessionID) (*Session, error)
	UpdateSession(ctx context.Context, sessionID SessionID, state FSMState, version int64) error
	ExpireSession(ctx context.Context, sessionID SessionID) error
}

type Session struct {
	ID        SessionID
	TenantID  TenantID
	State     FSMState
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
}

type RAGService interface {
	Retrieve(ctx context.Context, tenantID TenantID, query string) ([]*Chunk, error)
	GenerateResponse(ctx context.Context, tenantID TenantID, sessionID SessionID, query string) (*Response, error)
}

type Chunk struct {
	ID           string
	DocumentID   DocumentID
	Content      string
	Metadata     map[string]interface{}
	Vector       []float32
	Ordinal      int
	Source       string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Response struct {
	Text         string
	Citations    []*Citation
	Usage        *TokenUsage
	GeneratedAt  time.Time
}

type Citation struct {
	DocumentName string
	Ordinal      int
	Content      string
}

type TokenUsage struct {
	PromptTokens   int
	CompletionTokens int
	TotalTokens    int
	CostUSD        float64
}

type IngestionService interface {
	IngestDocument(ctx context.Context, tenantID TenantID, content []byte, filename string) (DocumentID, error)
	GetDocument(ctx context.Context, tenantID TenantID, docID DocumentID) (*Document, error)
}

type Document struct {
	ID          DocumentID
	TenantID    TenantID
	Name        string
	Size        int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Chunks      []*Chunk
	Version     int
	Status      string
}

type ArtifactService interface {
	GeneratePDF(ctx context.Context, tenantID TenantID, content any) (ArtifactID, error)
	GetArtifact(ctx context.Context, tenantID TenantID, artifactID ArtifactID) (*Artifact, error)
}

type Artifact struct {
	ID         ArtifactID
	TenantID   TenantID
	DocumentID DocumentID
	Name       string
	Size       int64
	CreatedAt  time.Time
	UpdatedAt  time.Time
	URL        string
}

type ChannelAdapter interface {
	HandleMessage(ctx context.Context, tenantID TenantID, message any) error
	SendResponse(ctx context.Context, tenantID TenantID, sessionID SessionID, response *Response) error
}

type TelegramAdapter struct {
	BotToken string
	Redis    *redis.Client
}

type WebWidgetAdapter struct {
	AuthKey  string
	Redis    *redis.Client
}

type AuditLogger interface {
	LogWriteTool(ctx context.Context, tenantID TenantID, operationKey string, toolName ToolName, request, response any) error
	LogFSMTransition(ctx context.Context, tenantID TenantID, sessionID SessionID, from, to FSMState) error
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID) (bool, error)
}

type RiverWorker struct {
	Pool   *pgxpool.Pool
	Queue  *river.Queue
	Tracer trace.Tracer
}

func (r *RiverWorker) Start(ctx context.Context) error { return nil }
func (r *RiverWorker) Stop(ctx context.Context) error   { return nil }

type APIServer struct {
	Router      *river.Queue
	Tracer      trace.Tracer
	RateLimiter RateLimiter
}

func (a *APIServer) Start(ctx context.Context) error { return nil }
func (a *APIServer) Stop(ctx context.Context) error  { return nil }

type Configuration struct {
	DatabaseURL       string
	RedisURL          string
	TelemetryEndpoint string
	DeploymentMode    string
}

func NewFramework(config Configuration) (*Framework, error) { return nil, nil }

type Framework struct {
	SessionManager SessionManager
	RAGService     RAGService
	Ingestion      IngestionService
	Artifact       ArtifactService
	ToolRegistry   ToolRegistry
	ModelRouter    ModelRouter
	PromptRegistry PromptRegistry
	AuditLogger    AuditLogger
	ChannelAdapters map[string]ChannelAdapter
	RateLimiter    RateLimiter
	Worker         *RiverWorker
	API            *APIServer
}

func (f *Framework) Run(ctx context.Context) error { return nil }
