package ragivka

import (
	"context"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/riverqueue/river"
	"go.opentelemetry.io/otel/trace"
)

type TenantID string
type SessionID string
type DocumentID string
type ArtifactID string
type ToolName string
type ModelProvider string
type State string

const (
	StateActive        State = "active"
	StateWaitingForHuman State = "waiting_for_human"
	StateCompleted     State = "completed"
	StateExpired       State = "expired"
)

type FSMTransition struct {
	SessionID SessionID
	From      State
	To        State
	Timestamp time.Time
}

type AuditLog struct {
	ID            int64
	SessionID     SessionID
	IdempotencyKey string
	ToolName      ToolName
	RequestHash   string
	ResponseHash  string
	Timestamp     time.Time
}

type ToolPermission string

const (
	ToolRead   ToolPermission = "read"
	ToolDraft  ToolPermission = "draft"
	ToolWrite  ToolPermission = "write"
)

type Tool struct {
	Name        ToolName
	Permissions []ToolPermission
	Schema      any
	Handler     func(context.Context, any) (any, error)
}

type ModelRouter interface {
	Route(context.Context, string) (string, error)
	GetModelInfo(string) (map[string]any, error)
}

type EmbeddingModel interface {
	Embed(context.Context, []string) ([][]float32, error)
}

type RAGPipeline interface {
	Ingest(context.Context, any) error
	Retrieve(context.Context, string, TenantID) ([]any, error)
	GenerateResponse(context.Context, string, []any, TenantID) (string, error)
}

type SessionManager interface {
	GetSession(context.Context, SessionID, TenantID) (*Session, error)
	CreateSession(context.Context, TenantID) (*Session, error)
	UpdateSession(context.Context, *Session) error
	TransitionState(context.Context, SessionID, State) error
	ExpireInactiveSessions(context.Context, time.Duration) error
}

type Session struct {
	ID        SessionID
	TenantID  TenantID
	State     State
	Version   int64
	CreatedAt time.Time
	UpdatedAt time.Time
	ExpiresAt time.Time
}

type Message struct {
	ID         int64
	SessionID  SessionID
	Content    string
	Role       string
	Timestamp  time.Time
}

type ToolRegistry interface {
	RegisterTool(Tool) error
	GetTool(ToolName) (*Tool, error)
	ListTools(TenantID) ([]ToolName, error)
}

type ChannelAdapter interface {
	HandleMessage(context.Context, any) error
	ValidateRequest(context.Context, any) error
}

type TelegramAdapter struct{}

type WebWidgetAdapter struct{}

type APIService struct {
	tracer trace.Tracer
}

type WorkerService struct {
	riverClient *river.Client
	dbPool      *pgxpool.Pool
}

type CostTracker struct {
}

type MetricsCollector struct {
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func (s *SessionManager) GetSession(ctx context.Context, id SessionID, tenant TenantID) (*Session, error) {}
func (s *SessionManager) CreateSession(ctx context.Context, tenant TenantID) (*Session, error)         {}
func (s *SessionManager) UpdateSession(ctx context.Context, session *Session) error                   {}
func (s *SessionManager) TransitionState(ctx context.Context, id SessionID, to State) error           {}
func (s *SessionManager) ExpireInactiveSessions(ctx context.Context, timeout time.Duration) error     {}

func (r *RAGPipeline) Ingest(ctx context.Context, document any) error                              {}
func (r *RAGPipeline) Retrieve(ctx context.Context, query string, tenant TenantID) ([]any, error)   {}
func (r *RAGPipeline) GenerateResponse(ctx context.Context, query string, chunks []any, tenant TenantID) (string, error) {}

func (t *ToolRegistry) RegisterTool(tool Tool) error                    {}
func (t *ToolRegistry) GetTool(name ToolName) (*Tool, error)            {}
func (t *ToolRegistry) ListTools(tenant TenantID) ([]ToolName, error)  {}

func (a *APIService) HandleRequest(ctx context.Context, req any) (any, error) {}
func (a *APIService) ValidateAuth(ctx context.Context, token string) (bool, error) {}

func (w *WorkerService) ProcessJob(ctx context.Context, job *river.Job) error {}
func (w *WorkerService) StartWorkers(ctx context.Context) error              {}

func (c *CostTracker) TrackCost(ctx context.Context, cost float64, tenant TenantID) error {}
func (m *MetricsCollector) CollectLLMUsage(ctx context.Context, promptTokens, completionTokens int) {}
