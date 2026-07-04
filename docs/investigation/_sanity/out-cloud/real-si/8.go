package main

import (
	"context"
	"fmt"
	"io"
	"time"
)

const SchemaVersion = 1
const EmbeddingDim = 1024
const SnippetLen = 200
const ShortMessageThreshold = 30
const DefaultSearchLimit = 5
const MineDeadline = 50 * time.Second

type Record struct {
	Type       string `json:"type"`
	IsMeta     bool   `json:"isMeta"`
	Content    string `json:"content,omitempty"`
	Text       string `json:"text,omitempty"`
	Timestamp  int64  `json:"timestamp,omitempty"`
	Timestamps struct {
		Message int64 `json:"message"`
	} `json:"timestamps,omitempty"`
	Message struct {
		Content string `json:"content"`
	} `json:"message,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

type Chunk struct {
	ID        int64
	SessionID string
	MessageID string
	Role      string
	Content   string
	CreatedAt time.Time
}

type Embedding struct {
	ChunkID int64
	Vector  []float32
}

type SearchResult struct {
	ChunkID   int64
	SessionID string
	Role      string
	Date      time.Time
	Snippet   string
	Score     float64
}

type MineResult struct {
	SessionID string
	Inserted  int
	Deferred  int
	Skipped   int
}

type SearchOptions struct {
	Limit int
	JSON  bool
}

type Indexer struct{}

func NewIndexer(dbPath string) (*Indexer, error) { return nil, nil }
func (idx *Indexer) Close() error                 { return nil }

type CLI struct{}

func NewCLI() *CLI { return nil }

func (c *CLI) Run(args []string, stdout io.Writer, stderr io.Writer) int { return 0 }

func main() {}

func initDB(dbPath string, expectedVersion int) error { return nil }
func parseJSONL(path string) ([]Record, error)        { return nil, nil }
func extractChunks(records []Record, sessionID string) []Chunk {
	return nil
}
func cleanContent(content string) string { return "" }
func isValidChunk(chunk Chunk) bool      { return false }

func (idx *Indexer) Mine(ctx context.Context, jsonlPath string) (*MineResult, error) {
	return nil, nil
}
func (idx *Indexer) Embed(ctx context.Context) error         { return nil }
func (idx *Indexer) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	return nil, nil
}
func (idx *Indexer) Stats() (Stats, error) { return Stats{}, nil }

type Stats struct {
	Sessions         int
	Chunks           int
	PendingEmbeddings int
	DBSizeBytes      int64
}

func embedText(ctx context.Context, text string) ([]float32, error) { return nil, nil }
func cosineSimilarity(a, b []float32) float64                      { return 0 }
func truncateSnippet(content string, maxLen int) string            { return "" }
func ollamaAvailable(ctx context.Context) bool                     { return false }

func projectRootFromPath(jsonlPath string) (string, error) { return "", nil }
func dbPathFromProjectRoot(root string) string             { return "" }
func sessionIDFromPath(path string) (string, error)        { return "", nil }

type wrappedError struct {
	Op      string
	Err     error
	Details string
}

func (e *wrappedError) Error() string { return "" }
func (e *wrappedError) Unwrap() error { return nil }

func wrapError(op string, err error, details string) error { return nil }

func fmtSchemaMismatchError(dbVersion, expectedVersion int) string { return "" }
