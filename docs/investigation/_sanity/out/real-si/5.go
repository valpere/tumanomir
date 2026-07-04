package main

import (
	"context"
	"database/sql"
	"time"
)

type SessionIndexer struct{}

type Chunk struct {
	ID          int64
	SessionID   string
	ProjectRoot string
	Content     string
	Role        string
	Timestamp   time.Time
	Embedded    bool
}

type SearchResult struct {
	ChunkID   int64
	Score     float64
	Content   string
	SessionID string
	Role      string
	Timestamp time.Time
}

type MineResult struct {
	ChunksIndexed int
	ChunksEmbedded int
	ChunksDeferred int
	EmbedErrors    int
}

type Stats struct {
	Sessions       int
	Chunks         int
	PendingEmbeds  int
	DBSizeBytes    int64
	SchemaVersion  int
}

func (s *SessionIndexer) Mine(ctx context.Context, jsonlPath, dbPath string) (*MineResult, error) {}
func (s *SessionIndexer) Search(ctx context.Context, query, dbPath string, limit int, useJSON bool) ([]SearchResult, error) {}
func (s *SessionIndexer) Embed(ctx context.Context, dbPath string) error {}
func (s *SessionIndexer) Stats(ctx context.Context, dbPath string) (*Stats, error) {}
func (s *SessionIndexer) OpenDB(dbPath string) (*sql.DB, error) {}
func (s *SessionIndexer) CloseDB(db *sql.DB) error {}
