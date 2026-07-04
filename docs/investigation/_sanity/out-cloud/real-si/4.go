package main

import (
	"context"
	"io"
	"time"
)

type RecordType string

const (
	RecordTypeUser      RecordType = "user"
	RecordTypeAssistant RecordType = "assistant"
)

type JSONLRecord struct {
	Type       string `json:"type"`
	Timestamp  int64  `json:"timestamp"`
	Message    string `json:"message,omitempty"`
	Content    string `json:"content,omitempty"`
	IsMeta     bool   `json:"isMeta,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
	Command    string `json:"command,omitempty"`
	ToolName   string `json:"toolName,omitempty"`
	ToolResult string `json:"toolResult,omitempty"`
}

type Chunk struct {
	SessionID string
	MessageID string
	Role      string
	Timestamp time.Time
	Content   string
	Trimmed   bool
	Deferred  bool
}

type MineResult struct {
	Sessions       int
	ChunksStored   int
	ChunksEmbedded int
	Skipped        int
	Deferred       int
	Elapsed        time.Duration
}

type SearchResult struct {
	SessionID string    `json:"session_id"`
	MessageID string    `json:"message_id"`
	Role      string    `json:"role"`
	Timestamp time.Time `json:"timestamp"`
	Snippet   string    `json:"snippet"`
	Score     float64   `json:"score"`
}

type Stats struct {
	Sessions           int
	Chunks             int
	Embeddings         int
	PendingEmbeddings  int
	DBSizeBytes        int64
	SchemaVersion      int
}

type Index interface {
	Open(dbPath string, expectedSchema int) error
	Close() error
	Begin() error
	Commit() error
	Rollback() error
	StoreSession(sessionID string, minedAt time.Time) error
	StoreChunk(chunk Chunk) (chunkID int64, err error)
	MarkChunkDeferred(chunkID int64) error
	GetEmbedding(sessionID, messageID string) ([]float32, error)
	StoreEmbedding(chunkID int64, vector []float32) error
	SearchEmbeddings(queryVector []float32, limit int) ([]SearchResult, error)
	SearchFTS5(query string, limit int) ([]SearchResult, error)
	HasEmbeddings() (bool, error)
	PendingEmbeddingChunks() ([]int64, error)
	Stats() (Stats, error)
	SchemaVersion() (int, error)
}

type OllamaClient interface {
	Embed(ctx context.Context, model string, texts []string) ([][]float32, error)
	Healthy(ctx context.Context) error
}

type Config struct {
	DBPath      string
	Model       string
	Dim         int
	Limit       int
	JSONOutput  bool
	OllamaURL   string
	ProjectRoot string
}

type CLI struct {
	Out io.Writer
	Err io.Writer
}

func main() {}

func NewCLI(out io.Writer, err io.Writer) *CLI {
	return nil
}

func (c *CLI) Run(args []string) int {
	return 0
}

func Mine(ctx context.Context, db Index, ollama OllamaClient, jsonlPath string, cfg Config) (MineResult, error) {
	return MineResult{}, nil
}

func Search(ctx context.Context, db Index, ollama OllamaClient, query string, cfg Config) ([]SearchResult, error) {
	return nil, nil
}

func Embed(ctx context.Context, db Index, ollama OllamaClient, cfg Config) error {
	return nil
}

func StatsCmd(db Index) (Stats, error) {
	return Stats{}, nil
}

func ParseJSONL(r io.Reader, sessionID string) ([]Chunk, error) {
	return nil, nil
}

func NormalizeContent(rec JSONLRecord) (string, bool) {
	return "", false
}

func ProjectRoot(path string) (string, error) {
	return "", nil
}

func DefaultDBPath(projectRoot string) string {
	return ""
}

func Snippet(content string, maxLen int) string {
	return ""
}

func Cosine(a, b []float32) float64 {
	return 0
}

func OpenIndex(dbPath string, schemaVersion int) (Index, error) {
	return nil, nil
}

func NewOllamaClient(baseURL string) OllamaClient {
	return nil
}

func EmbeddingModel() string {
	return ""
}

func EmbeddingDim() int {
	return 0
}

func SchemaVersion() int {
	return 0
}
