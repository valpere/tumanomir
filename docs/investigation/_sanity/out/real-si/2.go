package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type SessionIndexer struct{}

type Chunk struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	ProjectRoot string    `json:"project_root"`
	Content     string    `json:"content"`
	Role        string    `json:"role"`
	Timestamp   time.Time `json:"timestamp"`
	Embedded    bool      `json:"embedded"`
}

type SearchResult struct {
	ChunkID   int64     `json:"chunk_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Snippet   string    `json:"snippet"`
	Score     float64   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

type MineResult struct {
	ChunksIndexed int `json:"chunks_indexed"`
	Embeddings    int `json:"embeddings"`
	Skipped       int `json:"skipped"`
	Deferred      int `json:"deferred"`
}

func (s *SessionIndexer) Mine(ctx context.Context, jsonlPath, dbPath string) (*MineResult, error) {}
func (s *SessionIndexer) Search(ctx context.Context, query, dbPath string, limit int, jsonOutput bool) ([]SearchResult, error) {}
func (s *SessionIndexer) Embed(ctx context.Context, dbPath string) error {}
func (s *SessionIndexer) Stats(ctx context.Context, dbPath string) (map[string]interface{}, error) {}

type SessionRecord struct {
	Type      string `json:"type"`
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
	IsMeta    bool   `json:"isMeta"`
}

func parseJSONL(path string) ([]SessionRecord, error) {}
func extractChunks(records []SessionRecord, sessionID string) []Chunk {}
func storeChunks(db *sql.DB, chunks []Chunk) error {}
func generateEmbeddings(db *sql.DB, chunks []Chunk, ctx context.Context) (*MineResult, error) {}
func cosineSimilarity(embedding1, embedding2 []float32) float64 {}
func fts5Search(db *sql.DB, query string, limit int) ([]SearchResult, error) {}
func getProjectRoot(jsonlPath string) (string, error) {}
