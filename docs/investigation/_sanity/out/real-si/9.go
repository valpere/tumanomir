package main

import (
	"context"
	"database/sql"
	"time"
)

type Chunk struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	ProjectRoot string    `json:"project_root"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Date        time.Time `json:"date"`
	Embedded    bool      `json:"embedded"`
	Deferred    bool      `json:"deferred"`
}

type SearchResult struct {
	ID          int64     `json:"id"`
	SessionID   string    `json:"session_id"`
	ProjectRoot string    `json:"project_root"`
	Role        string    `json:"role"`
	Content     string    `json:"content"`
	Date        time.Time `json:"date"`
	Score       float64   `json:"score"`
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

type SessionIndexer struct{}

func (s *SessionIndexer) Mine(ctx context.Context, jsonlPath string, dbPath string) (*MineResult, error) {}
func (s *SessionIndexer) Search(ctx context.Context, query string, dbPath string, limit int, jsonOutput bool) ([]SearchResult, error) {}
func (s *SessionIndexer) Embed(ctx context.Context, dbPath string) error {}
func (s *SessionIndexer) Stats(ctx context.Context, dbPath string) (*Stats, error) {}
func (s *SessionIndexer) OpenDB(dbPath string) (*sql.DB, error) {}
func (s *SessionIndexer) CloseDB(db *sql.DB) error {}
