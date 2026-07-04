package ragivka

import (
	"context"
	"database/sql"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type TenantID string
type SessionID string
type JobID string
type OperationKey string
type ToolName string
type PromptName string
type PromptVersion string
type DocumentID string
type ChunkID string
type ArtifactID string
type TraceID string
type RequestID string
type ModelProvider string
type ToolPermission string

const (
	StateActive          string = "Active"
	StateWaitingForHuman string = "WaitingForHuman"
	StateCompleted       string = "Completed"
	StateExpired         string = "Expired"

	PermissionRead  ToolPermission = "Read"
	PermissionDraft ToolPermission = "Draft"
	PermissionWrite ToolPermission = "Write"

	ProviderOpenAI    ModelProvider = "openai"
	ProviderAnthropic ModelProvider = "anthropic"
	ProviderOpenRouter ModelProvider = "openrouter"
	ProviderGemini    ModelProvider = "gemini"
	ProviderOllama    ModelProvider = "ollama"
)

type Config struct {
	MaxSyncLatency      time.Duration
	ConcurrencyTarget   int
	InactivityTimeout   time.Duration
	ContextWindowTurns  int
	DefaultChunkSize    int
	DefaultChunkOverlap float64
	TopK                int
}

type Tenant struct {
	ID        TenantID
	Name      string
	CreatedAt time.Time
}

type Session struct {
	ID          SessionID
	TenantID    TenantID
	State       string
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	LastActivityAt time.Time
	Metadata    map[string]string
}

type Message struct {
	ID        string
	SessionID SessionID
	Role      string
	Content   string
	Turn      int
	CreatedAt time.Time
}

type Document struct {
	ID         DocumentID
	TenantID   TenantID
	ObjectKey  string
	SourceURI  string
	Version    int
	RawHash    string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type Chunk struct {
	ID           ChunkID
	TenantID     TenantID
	DocumentID   DocumentID
	Ordinal      int
	SourceLocation string
	Content      string
	Embedding    []float32
	Tsvector     string
	Metadata     map[string]string
}

type Citation struct {
	DocumentName string
	ChunkOrdinal int
	SourceLocation string
}

type RetrievalResult struct {
	Chunk     Chunk
	Score     float64
	Citation  Citation
}

type Artifact struct {
	ID         ArtifactID
	TenantID   TenantID
	ObjectKey  string
	Kind       string
	CreatedAt  time.Time
}

type LLMRequest struct {
	Provider      ModelProvider
	Model         string
	Messages      []Message
	Temperature   float64
	MaxTokens     int
	ResponseSchema any
}

type LLMResponse struct {
	Content      string
	Structured   map[string]any
	Usage        TokenUsage
	Provider     ModelProvider
	Model        string
	DurationMs   int64
}

type TokenUsage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
	EstimatedCostUSD float64
}

type Prompt struct {
	Name    PromptName
	Version PromptVersion
	Template string
	Variables []string
}

type Tool interface {
	Name() ToolName
	Permission() ToolPermission
	Schema() map[string]any
	Execute(ctx context.Context, input map[string]any) (map[string]any, error)
}

type ToolRegistry interface {
	Register(tool Tool) error
	Get(name ToolName) (Tool, bool)
	List() []Tool
}

type ModelRouter interface {
	Route(ctx context.Context, task TaskComplexity, req LLMRequest) (LLMResponse, error)
}

type TaskComplexity int

const (
	TaskL0Deterministic TaskComplexity = iota
	TaskL1ToolAssistant
	TaskL2Workflow
	TaskL3MultiAgent
)

type KnowledgeStore interface {
	Ingest(ctx context.Context, doc Document, chunks []Chunk) error
	HybridSearch(ctx context.Context, tenantID TenantID, query string, topK int) ([]RetrievalResult, error)
	ReRank(ctx context.Context, results []RetrievalResult, query string) ([]RetrievalResult, error)
	GetDocument(ctx context.Context, tenantID TenantID, id DocumentID) (Document, error)
}

type SessionStore interface {
	Create(ctx context.Context, tenantID TenantID) (Session, error)
	Get(ctx context.Context, tenantID TenantID, id SessionID) (Session, error)
	Update(ctx context.Context, session Session) (Session, error)
	Transition(ctx context.Context, tx *sql.Tx, session Session, newState string) error
	ListActive(ctx context.Context, tenantID TenantID, before time.Time) ([]Session, error)
	AppendMessage(ctx context.Context, sessionID SessionID, msg Message) error
	GetMessages(ctx context.Context, sessionID SessionID, limit int) ([]Message, error)
}

type AuditLogEntry struct {
	ID             string
	TenantID       TenantID
	IdempotencyKey OperationKey
	ToolName       ToolName
	RequestHash    string
	ResponseHash   string
	FSMFromState   string
	FSMToState     string
	CreatedAt      time.Time
}

type AuditStore interface {
	Log(ctx context.Context, entry AuditLogEntry) error
}

type JobStore interface {
	Insert(ctx context.Context, job Job) (JobID, error)
	Claim(ctx context.Context, queue string) (Job, error)
	Complete(ctx context.Context, jobID JobID) error
	Fail(ctx context.Context, jobID JobID, err error, retryable bool) error
}

type Job struct {
	ID          JobID
	TenantID    TenantID
	Queue       string
	Payload     []byte
	Attempt     int
	MaxAttempts int
	CreatedAt   time.Time
	ScheduledAt time.Time
}

type RateLimiter interface {
	Allow(ctx context.Context, tenantID TenantID, key string, limit int, window time.Duration) (bool, time.Duration, error)
}

type ObjectStore interface {
	Put(ctx context.Context, key string, data []byte, metadata map[string]string) error
	Get(ctx context.Context, key string) ([]byte, map[string]string, error)
	Delete(ctx context.Context, key string) error
}

type IngestionConnector interface {
	Fetch(ctx context.Context, uri string) ([]byte, map[string]string, error)
}

type Parser interface {
	Parse(ctx context.Context, contentType string, data []byte) (string, []ChunkCandidate, error)
}

type ChunkCandidate struct {
	Content        string
	SourceLocation string
	Metadata       map[string]string
}

type FSM interface {
	CanTransition(from, to string) bool
	Transition(ctx context.Context, session *Session, to string) error
}

type Orchestrator interface {
	RunL0(ctx context.Context, tenantID TenantID, req L0Request) (L0Response, error)
	RunL1(ctx context.Context, tenantID TenantID, sessionID SessionID, req L1Request) (L1Response, error)
	EnqueueL2(ctx context.Context, tenantID TenantID, req L2Request) (JobID, error)
	EnqueueL3(ctx context.Context, tenantID TenantID, req L3Request) (JobID, error)
}

type L0Request struct {
	Prompt Prompt
	Input  map[string]any
}

type L0Response struct {
	Output     string
	Usage      TokenUsage
	TraceID    TraceID
}

type L1Request struct {
	Message  string
	SessionID SessionID
}

type L1Response struct {
	Message    string
	Citations  []Citation
	State      string
	RequiresHuman bool
	Usage      TokenUsage
	TraceID    TraceID
}

type L2Request struct {
	Name    string
	Payload []byte
}

type L2Response struct {
	JobID JobID
}

type L3Request struct {
	Name string
	DAG  DAGNode
}

type L3Response struct {
	JobID JobID
}

type DAGNode struct {
	ID       string
	Agent    string
	Timeout  time.Duration
	Depends  []string
	Reviewers []string
}

type ChannelAdapter interface {
	Send(ctx context.Context, tenantID TenantID, recipient string, message ChannelMessage) error
	Receive(ctx context.Context, payload []byte) (ChannelMessage, error)
}

type ChannelMessage struct {
	Recipient   string
	Text        string
	Markup      string
	Attachments []string
}

type TelegramAdapter struct{}

type WebWidgetAdapter struct{}

type Evaluator interface {
	ComputeRecallAtK(ctx context.Context, expected []ChunkID, actual []ChunkID, k int) (float64, error)
	ComputeCitationCoverage(ctx context.Context, answer string, citations []Citation) (float64, error)
	RecordGroundednessHook(ctx context.Context, sessionID SessionID, score float64) error
}

type Tracer interface {
	Start(ctx context.Context, name string) (context.Context, Span)
}

type Span interface {
	End()
	RecordError(err error)
	SetAttributes(attrs map[string]string)
}

type MetricsCollector interface {
	RecordLLMUsage(ctx context.Context, usage TokenUsage)
	RecordRetrievalLatency(ctx context.Context, p50, p95 time.Duration)
	RecordQueueDepth(ctx context.Context, queue string, depth int)
	RecordError(ctx context.Context, kind string)
	Register() *prometheus.Registry
}

type Authenticator interface {
	Authenticate(ctx context.Context, token string) (TenantID, map[string]string, error)
}

type APIError struct {
	Code    string
	Message string
	Details map[string]any
}

func (e *APIError) Error() string { return "" }

type Server struct {
	Pool         *pgxpool.Pool
	Orchestrator Orchestrator
	Auth         Authenticator
	RateLimiter  RateLimiter
	Tracer       Tracer
	Metrics      MetricsCollector
}

func (s *Server) HandleL0(ctx context.Context, req L0Request) (L0Response, error) {
	return L0Response{}, nil
}

func (s *Server) HandleL1(ctx context.Context, sessionID SessionID, req L1Request) (L1Response, error) {
	return L1Response{}, nil
}

func (s *Server) HandleL2Enqueue(ctx context.Context, req L2Request) (L2Response, error) {
	return L2Response{}, nil
}

func (s *Server) HandleL3Enqueue(ctx context.Context, req L3Request) (L3Response, error) {
	return L3Response{}, nil
}

type Worker struct {
	Pool        *pgxpool.Pool
	JobStore    JobStore
	Orchestrator Orchestrator
	Tracer      Tracer
	Metrics     MetricsCollector
}

func (w *Worker) Run(ctx context.Context, queues []string) error {
	return nil
}

func (w *Worker) ProcessJob(ctx context.Context, job Job) error {
	return nil
}

type IngestionPipeline struct {
	Connectors []IngestionConnector
	Parsers    []Parser
	Chunker    Chunker
	Store      KnowledgeStore
	Object     ObjectStore
	PIIHook    func([]byte) []byte
}

type Chunker interface {
	Chunk(ctx context.Context, text string, config ChunkConfig) ([]ChunkCandidate, error)
}

type ChunkConfig struct {
	ChunkSize int
	Overlap   float64
}

func (p *IngestionPipeline) Process(ctx context.Context, tenantID TenantID, uri string, contentType string) (DocumentID, error) {
	return "", nil
}

type RAGPipeline struct {
	Store      KnowledgeStore
	Reranker   KnowledgeStore
	LLM        ModelRouter
	Prompts    PromptRegistry
}

type PromptRegistry interface {
	Load(ctx context.Context, name PromptName, version PromptVersion) (Prompt, error)
}

func (p *RAGPipeline) Answer(ctx context.Context, tenantID TenantID, query string) (string, []Citation, TokenUsage, error) {
	return "", nil, TokenUsage{}, nil
}

type ToolExecutor struct {
	Registry     ToolRegistry
	Audit        AuditStore
	RateLimiter  RateLimiter
	Cache        Cache
}

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
}

func (e *ToolExecutor) Execute(ctx context.Context, tenantID TenantID, opKey OperationKey, toolName ToolName, input map[string]any) (map[string]any, error) {
	return nil, nil
}

type ArtifactGenerator struct {
	Object ObjectStore
	Store  ArtifactStore
}

type ArtifactStore interface {
	Create(ctx context.Context, artifact Artifact) error
	Get(ctx context.Context, tenantID TenantID, id ArtifactID) (Artifact, error)
}

func (g *ArtifactGenerator) GeneratePDF(ctx context.Context, tenantID TenantID, data map[string]any) (ArtifactID, error) {
	return "", nil
}

func (g *ArtifactGenerator) GenerateExcel(ctx context.Context, tenantID TenantID, data map[string]any) (ArtifactID, error) {
	return "", nil
}

type CostTracker interface {
	LogRequest(ctx context.Context, tenantID TenantID, usage TokenUsage) error
	GetTenantSpend(ctx context.Context, tenantID TenantID, since time.Time) (float64, error)
}

type PIIStripper interface {
	Strip(ctx context.Context, text string) (string, error)
}
