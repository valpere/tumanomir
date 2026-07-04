package main

import (
	"context"
	"database/sql"
	"time"
)

type Chunk struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	Content     string    `json:"content"`
	Role        string    `json:"role"`
	Timestamp   time.Time `json:"timestamp"`
	Embedding   []float32 `json:"embedding,omitempty"`
	Deferred    bool      `json:"deferred"`
	ProjectRoot string    `json:"project_root"`
}

type SearchResult struct {
	ChunkID   int64     `json:"chunk_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Snippet   string    `json:"snippet"`
	Score     float32   `json:"score"`
	Timestamp time.Time `json:"timestamp"`
}

type MineResult struct {
	ChunksIndexed int `json:"chunks_indexed"`
	ChunksEmbedded int `json:"chunks_embedded"`
	ChunksDeferred int `json:"chunks_deferred"`
	ChunksSkipped  int `json:"chunks_skipped"`
}

type Stats struct {
	Sessions     int `json:"sessions"`
	Chunks       int `json:"chunks"`
	PendingEmbed int `json:"pending_embed"`
	DBSize       int `json:"db_size"`
}

type DB struct {
	*sql.DB
}

func (db *DB) Mine(ctx context.Context, jsonlPath string, projectRoot string) (*MineResult, error) {}
func (db *DB) Search(ctx context.Context, query string, limit int, useJSON bool) ([]SearchResult, error) {}
func (db *DB) Embed(ctx context.Context, projectRoot string) error {}
func (db *DB) Stats(projectRoot string) (*Stats, error) {}
func (db *DB) Close() error {}
