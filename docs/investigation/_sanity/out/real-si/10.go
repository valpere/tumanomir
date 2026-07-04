package main

import (
	"context"
	"database/sql"
	"time"
)

type Chunk struct {
	ID         int64     `json:"id"`
	SessionID  string    `json:"session_id"`
	ProjectID  string    `json:"project_id"`
	Role       string    `json:"role"`
	Content    string    `json:"content"`
	Date       time.Time `json:"date"`
	WordCount  int       `json:"word_count"`
	Embedding  []float32 `json:"embedding,omitempty"`
	Deferred   bool      `json:"deferred"`
	IsMeta     bool      `json:"is_meta"`
}

type Session struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Date      time.Time `json:"date"`
	Chunks    []Chunk   `json:"chunks"`
}

type MineResult struct {
	NewChunks     int `json:"new_chunks"`
	DeferredChunks int `json:"deferred_chunks"`
	SkippedChunks int `json:"skipped_chunks"`
}

type SearchResult struct {
	ChunkID   int64     `json:"chunk_id"`
	SessionID string    `json:"session_id"`
	Role      string    `json:"role"`
	Date      time.Time `json:"date"`
	Snippet   string    `json:"snippet"`
	Score     float32   `json:"score"`
}

type Stats struct {
	Sessions       int64 `json:"sessions"`
	Chunks         int64 `json:"chunks"`
	PendingEmbeds  int64 `json:"pending_embeds"`
	DBSizeBytes    int64 `json:"db_size_bytes"`
	SchemaVersion  int   `json:"schema_version"`
}

type DB struct {
	path string
	db   *sql.DB
}

func (db *DB) Mine(ctx context.Context, jsonlPath string, projectID string) (*MineResult, error) {}
func (db *DB) Search(ctx context.Context, query string, limit int, useJSON bool) ([]SearchResult, error) {}
func (db *DB) Embed(ctx context.Context) error {}
func (db *DB) Stats() (*Stats, error) {}
func (db *DB) Close() error {}

func NewDB(path string) (*DB, error) {}
func OpenDB(path string) (*DB, error) {}
