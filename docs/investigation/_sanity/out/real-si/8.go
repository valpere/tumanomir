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
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Date        time.Time `json:"date"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Deferred    bool      `json:"deferred"`
}

type SearchResult struct {
	ChunkID   int64     `json:"chunk_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Snippet   string    `json:"snippet"`
	Date      time.Time `json:"date"`
	Score     float32   `json:"score"`
}

type MineResult struct {
	ChunksIndexed int     `json:"chunks_indexed"`
	Embeddings    int     `json:"embeddings"`
	Skipped       int     `json:"skipped"`
	Deferred      int     `json:"deferred"`
}

type Stats struct {
	Sessions     int `json:"sessions"`
	Chunks       int `json:"chunks"`
	PendingEmbed int `json:"pending_embed"`
	DBSize       int `json:"db_size"`
}

func (s *SessionIndexer) Mine(ctx context.Context, jsonlPath, dbPath string) (*MineResult, error) {
	return &MineResult{}, nil
}

func (s *SessionIndexer) Search(ctx context.Context, query, dbPath string, limit int, jsonOutput bool) ([]SearchResult, error) {
	return []SearchResult{}, nil
}

func (s *SessionIndexer) Embed(ctx context.Context, dbPath string) error {
	return nil
}

func (s *SessionIndexer) Stats(ctx context.Context, dbPath string) (*Stats, error) {
	return &Stats{}, nil
}

type JSONLRecord struct {
	Type      string `json:"type"`
	IsMeta    bool   `json:"isMeta"`
	Content   string `json:"content"`
	Date      string `json:"date"`
	SessionID string `json:"sessionId"`
}

func ParseJSONL(path string) ([]JSONLRecord, error) {
	return []JSONLRecord{}, nil
}

func ExtractChunks(records []JSONLRecord, sessionID string) []*Chunk {
	return []*Chunk{}
}

func (s *SessionIndexer) OpenDB(dbPath string) (*sql.DB, error) {
	return &sql.DB{}, nil
}

func (s *SessionIndexer) CloseDB(db *sql.DB) error {
	return nil
}

func (s *SessionIndexer) EnsureSchema(db *sql.DB) error {
	return nil
}
