package dreet

import (
	"context"
	"time"
)

// --- Domain Types ---

type Format string

const (
	FormatCloze       Format = "cloze"
	FormatSpec        Format = "spec"
	FormatProgressive Format = "progressive"
	FormatBugfix      Format = "bugfix"
	FormatSkeleton    Format = "skeleton"
)

type CardMeta struct {
	ID        string
	Lang      string
	Title     string
	Format    Format
	Prereq    string
	Order     int
	TargetSec int
	Tags      []string
	EntryFile string
	Hints     []string
}

type CardFiles struct {
	Meta     string
	Prompt   string
	Module   string
	Stub     string
	Tests    []string
	Solution string
}

type CardState struct {
	CardID         string
	DeckID         string
	PrereqCardID   string
	SortOrder      int
	Tags           []string
	Reps           int
	Lapses         int
	IntervalDays   float64
	EaseFactor     float64
	DueAt          time.Time
	LastReviewedAt *time.Time
	Active         bool
}

type AttemptStatus string

const (
	AttemptPass    AttemptStatus = "pass"
	AttemptGiveup  AttemptStatus = "giveup"
)

type Attempt struct {
	ID            int
	CardID        string
	StartedAt     time.Time
	FinishedAt    time.Time
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        AttemptStatus
	Grade         int
}

type CheckStatus string

const (
	StatusPass         CheckStatus = "pass"
	StatusCompileError CheckStatus = "compile_error"
	StatusTestFail     CheckStatus = "test_fail"
	StatusTimeout      CheckStatus = "timeout"
)

type Severity string

const (
	SevError   Severity = "error"
	SevWarning Severity = "warning"
)

type Diagnostic struct {
	File     string
	Line     int
	Col      int
	Severity Severity
	Message  string
}

type CheckResult struct {
	Status      CheckStatus
	DurationMs  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Grade int

type Stats struct {
	QueueToday      int
	ProcessedToday  int
	PerfectRatio    float64
	WeakTags        []TagStat
}

type TagStat struct {
	Tag           string
	AverageGrade  float64
}

type CardView struct {
	Meta        CardMeta
	CurrentCode string
	Prompt      string
	HintsShown  []string
	StartedAt   time.Time
	TargetSec   int
}

// --- Interface Definitions ---

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

// --- Service Signatures ---

type Loader struct{}

func (l *Loader) LoadAll(rootDir string) ([]CardMeta, error) {
	return nil, nil
}

type SRSStore struct{}

func (s *SRSStore) NextDue(ctx context.Context, now time.Time, newPerDay int) (CardState, error) {
	return CardState{}, nil
}

func (s *SRSStore) RecordAttempt(ctx context.Context, attempt Attempt) error {
	return nil
}

func (s *SRSStore) Stats(ctx context.Context, now time.Time) (Stats, error) {
	return Stats{}, nil
}

func EvaluateGrade(elapsed, target, compileErrors, testFails, hintsUsed int, status AttemptStatus) Grade {
	return 0
}

func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {
	return CardState{}
}

type SandboxManager struct{}

func (m *SandboxManager) Prepare(cardID string, meta CardMeta) (string, error) {
	return "", nil
}

func (m *SandboxManager) WriteCode(workDir, entryFile, code string) error {
	return nil
}

type SessionService struct{}

func (s *SessionService) Next(ctx context.Context) (CardView, error) {
	return CardView{}, nil
}

func (s *SessionService) Current(ctx context.Context) (CardView, error) {
	return CardView{}, nil
}

func (s *SessionService) Run(ctx context.Context, cardID, code string) (CheckResult, error) {
	return CheckResult{}, nil
}

func (s *SessionService) Hint(ctx context.Context) (string, error) {
	return "", nil
}

func (s *SessionService) Giveup(ctx context.Context, cardID, code string) error {
	return nil
}

func (s *SessionService) Diff(ctx context.Context, cardID string) (userCode, solutionCode string, err error) {
	return "", "", nil
}
