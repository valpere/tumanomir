package ragivka

import (
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type TenantID uuid.UUID
type SessionID uuid.UUID
type DocumentID uuid.UUID
type ChunkID uuid.UUID
type ArtifactID uuid.UUID
type ToolName string
type PromptName string
type PromptVersion string
type IdempotencyKey string
type OperationHash string
type ModelProvider string

const (
	ModelProviderOpenAI     ModelProvider = "openai"
	ModelProviderAnthropic  ModelProvider = "anthropic"
	ModelProviderOpenRouter ModelProvider = "openrouter"
	ModelProviderGemini     ModelProvider = "gemini"
	ModelProviderOllama     ModelProvider = "ollama"
)

type Tier string

const (
	TierL0 Tier = "L0"
	TierL1 Tier = "L1"
	TierL2 Tier = "L2"
	TierL3 Tier = "L3"
)

type FSMState string

const (
	FSMStateActive         FSMState = "Active"
	FSMStateWaitingForHuman FSMState = "WaitingForHuman"
	FSMStateCompleted      FSMState = "Completed"
	FSMStateExpired        FSMState = "Expired"
)

type ToolPermission string

const (
	ToolPermissionRead  ToolPermission = "Read"
	ToolPermissionDraft ToolPermission = "Draft"
	ToolPermissionWrite ToolPermission = "Write"
)

type IngestionSourceType string

const (
	IngestionSourceTypePDF IngestionSourceType = "pdf"
	IngestionSourceTypeURL IngestionSourceType = "url"
	IngestionSourceTypeDB  IngestionSourceType = "db"
)

type DeploymentMode string

const (
	DeploymentModeSingleBinary DeploymentMode = "single-binary"
	DeploymentModeDockerCompose DeploymentMode = "docker-compose"
	DeploymentModeSplit         DeploymentMode = "split"
	DeploymentModeOffline       DeploymentMode = "offline"
)

type ErrorResponse struct {
	Code    string          `json:"code"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

type Tenant struct {
	ID              TenantID
	Name            string
	RateLimitRPM    int
	EmbeddingModel  string
	LanguageWhitelist []string
	CreatedAt       time.Time
}

type Session struct {
	ID           SessionID
	TenantID     TenantID
	State        FSMState
	Context      json.RawMessage
	Version      int
	LastActivity time.Time
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

type Message struct {
	ID        uuid.UUID
	SessionID SessionID
	Role      string
	Content   string
	Turn      int
	CreatedAt time.Time
}

type Document struct {
	ID          DocumentID
	TenantID    TenantID
	URI         string
	SourceType  IngestionSourceType
	Metadata    json.RawMessage
	Version     int
	IngestedAt  time.Time
}

type Chunk struct {
	ID           ChunkID
	DocumentID   DocumentID
	TenantID     TenantID
	Ordinal      int
	Content      string
	Embedding    pgtype.Vector
	Tsvector     string
	SourceOffset int
}

type Artifact struct {
	ID         ArtifactID
	TenantID   TenantID
	SessionID  SessionID
	URI        string
	ContentHash OperationHash
	CreatedAt  time.Time
}

type Prompt struct {
	Name        PromptName
	Version     PromptVersion
	Content     string
	Variables   []string
	Active      bool
	CreatedAt   time.Time
}

type ToolSchema struct {
	Name        ToolName
	Permission  ToolPermission
	Description string
	InputSchema json.RawMessage
	CacheTTL    time.Duration
}

type ToolRegistry interface {
	Register(ctx context.Context, tool ToolSchema) error
	Get(ctx context.Context, name ToolName) (ToolSchema, bool)
	List(ctx context.Context) []ToolSchema
}

type ToolInvocation struct {
	Name      ToolName
	Input     json.RawMessage
	IdempotencyKey IdempotencyKey
}

type ToolResult struct {
	Output    json.RawMessage
	Error     string
	FromCache bool
}

type LLMRequest struct {
	TenantID      TenantID
	ModelProvider ModelProvider
	ModelName     string
	Messages      []Message
	Temperature   float64
	JSONSchema    json.RawMessage
	MaxTokens     int
}

type LLMResponse struct {
	Content      string
	Structured   json.RawMessage
	Provider     ModelProvider
	Model        string
	PromptTokens int
	CompTokens   int
	CostUSD      float64
}

type ModelRouter interface {
	Route(ctx context.Context, req LLMRequest) (ModelProvider, string)
	Complete(ctx context.Context, req LLMRequest) (LLMResponse, error)
}

type Citation struct {
	DocumentName string
	ChunkOrdinal int
	Excerpt      string
}

type RetrievalResult struct {
	Chunk     Chunk
	Score     float64
	Citation  Citation
}

type SearchQuery struct {
	TenantID TenantID
	Text     string
	TopK     int
}

type Retriever interface {
	HybridSearch(ctx context.Context, q SearchQuery) ([]RetrievalResult, error)
	Rerank(ctx context.Context, results []RetrievalResult) ([]RetrievalResult, error)
}

type AuditLogEntry struct {
	ID             uuid.UUID
	TenantID       TenantID
	IdempotencyKey IdempotencyKey
	Actor          string
	Action         string
	ToolName       ToolName
	RequestHash    OperationHash
	ResponseHash   OperationHash
	Metadata       json.RawMessage
	CreatedAt      time.Time
}

type JobArgs interface {
	Kind() string
}

type RiverJob[T JobArgs] struct {
	ID         uuid.UUID
	Args       T
	Queue      string
	Attempt    int
	MaxRetries int
	RunAt      time.Time
}

type Worker[T JobArgs] interface {
	Work(ctx context.Context, job RiverJob[T]) error
}

type FSM interface {
	Transition(ctx context.Context, sessionID SessionID, from, to FSMState) (*Session, error)
	Expire(ctx context.Context) error
}

type Orchestrator interface {
	ExecuteL0(ctx context.Context, tenantID TenantID, prompt Prompt, input json.RawMessage) (LLMResponse, error)
	ExecuteL1(ctx context.Context, sessionID SessionID, input string) (*Session, []ToolInvocation, []Citation, error)
	EnqueueL2(ctx context.Context, tenantID TenantID, args JobArgs) error
	ExecuteL3(ctx context.Context, tenantID TenantID, dag DAG) error
}

type DAG struct {
	ID        uuid.UUID
	TenantID  TenantID
	Nodes     []DAGNode
	Edges     []DAGEdge
	Timeout   time.Duration
}

type DAGNode struct {
	ID       uuid.UUID
	Kind     string
	Args     JobArgs
	Timeout  time.Duration
	Critics  []uuid.UUID
}

type DAGEdge struct {
	From uuid.UUID
	To   uuid.UUID
}

type ChannelAdapter interface {
	HandleWebhook(w http.ResponseWriter, r *http.Request) error
	SendMessage(ctx context.Context, tenantID TenantID, sessionID SessionID, message string, citations []Citation) error
}

type IngestionPipeline interface {
	Ingest(ctx context.Context, tenantID TenantID, source IngestionSourceType, uri string, reader io.Reader) (*Document, error)
	Parse(ctx context.Context, doc *Document) ([]Chunk, error)
	Chunk(ctx context.Context, doc *Document, text string) ([]Chunk, error)
}

type ArtifactGenerator interface {
	GeneratePDF(ctx context.Context, data json.RawMessage) (*Artifact, error)
	GenerateExcel(ctx context.Context, data json.RawMessage) (*Artifact, error)
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID, key string) (bool, time.Duration, error)
}

type AuthMiddleware interface {
	Authenticate(r *http.Request) (TenantID, error)
}

type Evaluator interface {
	RecallAtK(ctx context.Context, results []RetrievalResult, expected []ChunkID, k int) float64
	CitationCoverage(ctx context.Context, answer string, citations []Citation) float64
	Groundedness(ctx context.Context, answer string, context []string) (float64, error)
}

func NewTenant(id TenantID, name string) *Tenant { return nil }
func NewSession(tenantID TenantID) *Session { return nil }
func NewDocument(tenantID TenantID, sourceType IngestionSourceType, uri string) *Document { return nil }
func NewChunk(tenantID TenantID, documentID DocumentID, ordinal int, content string) *Chunk { return nil }
func NewArtifact(tenantID TenantID, sessionID SessionID, uri string) *Artifact { return nil }
func NewPrompt(name PromptName, version PromptVersion, content string) *Prompt { return nil }
func NewToolSchema(name ToolName, permission ToolPermission, description string) ToolSchema { return ToolSchema{} }
func NewLLMRequest(tenantID TenantID, provider ModelProvider, model string, messages []Message) LLMRequest { return LLMRequest{} }
func NewDAG(tenantID TenantID, timeout time.Duration) *DAG { return nil }
func NewAuditLogEntry(tenantID TenantID, idempotencyKey IdempotencyKey, actor, action string) *AuditLogEntry { return nil }

func (t *Tenant) ValidateLanguage(lang string) bool { return false }
func (s *Session) BumpVersion() {}
func (d *Document) UpdateVersion() {}
func (c *Chunk) ComputeEmbedding(ctx context.Context) error { return nil }
func (a *Artifact) ComputeHash() (OperationHash, error) { return "", nil }
func (p *Prompt) Render(vars map[string]string) (string, error) { return "", nil }
func (ts ToolSchema) Validate(input json.RawMessage) error { return nil }
func (r *RetrievalResult) RelevanceScore() float64 { return 0 }
func (dag *DAG) DetectDeadlock() error { return nil }

func EncodeErrorResponse(code, message string, details json.RawMessage) []byte { return nil }
func HashRequest(payload json.RawMessage) OperationHash { return "" }
func HashResponse(payload json.RawMessage) OperationHash { return "" }
func MaskPII(text string) string { return "" }
func TokenCount(text string, model string) int { return 0 }
func SlidingWindowKey(tenantID TenantID, window time.Duration) string { return "" }
func GenerateIdempotencyKey() IdempotencyKey { return "" }
func ProviderFromString(s string) (ModelProvider, error) { return "", nil }
func TierFromString(s string) (Tier, error) { return "", nil }
