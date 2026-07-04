package sessionindexer

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"time"

	_ "modernc.org/sqlite"
)

type SessionID string

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Chunk struct {
	ID        int64
	SessionID SessionID
	Timestamp time.Time
	Role      Role
	Content   string
}

type SearchResult struct {
	Chunk   Chunk
	Score   float64
	Snippet string
	Date    time.Time
}

type MineResult struct {
	Stored   int
	Embedded int
	Deferred int
	Skipped  int
}

type EmbedResult struct {
	Processed int
	Skipped   int
}

type Stats struct {
	Sessions           int
	Chunks             int
	PendingEmbeddings  int
	DBSizeBytes        int64
	SchemaVersion      int
}

type JSONLRecord struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Timestamp int64  `json:"timestamp"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	IsMeta    bool   `json:"isMeta"`
}

type Indexer struct {
	db     *sql.DB
	ollama OllamaClient
}

type OllamaClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Healthy(ctx context.Context) bool
}

type Config struct {
	DBPath        string
	OllamaURL     string
	SchemaVersion int
}

func Open(cfg Config) (*Indexer, error) {
	return nil, errors.New("not implemented")
}

func (idx *Indexer) Close() error {
	return nil
}

func (idx *Indexer) Mine(ctx context.Context, jsonlPath string) (MineResult, error) {
	return MineResult{}, nil
}

func (idx *Indexer) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	return nil, nil
}

func (idx *Indexer) Embed(ctx context.Context) (EmbedResult, error) {
	return EmbedResult{}, nil
}

func (idx *Indexer) Stats(ctx context.Context) (Stats, error) {
	return Stats{}, nil
}

func ParseJSONL(r io.Reader, fallbackSessionID SessionID) ([]Chunk, error) {
	return nil, nil
}

func CleanContent(content string) string {
	return ""
}

func TruncateSnippet(content string, maxLen int) string {
	return ""
}

func CosineSimilarity(a, b []float32) float32 {
	return 0
}

func FindProjectRoot(startDir string) (string, error) {
	return "", nil
}

func RunCLI(args []string) error {
	return nil
}
