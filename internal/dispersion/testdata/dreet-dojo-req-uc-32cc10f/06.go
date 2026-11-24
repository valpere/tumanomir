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
	StatusPass    AttemptStatus = "pass"
	StatusGiveup  AttemptStatus = "giveup"
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
	CheckPass         CheckStatus = "pass"
	CheckCompileError CheckStatus = "compile_error"
	CheckTestFail     CheckStatus = "test_fail"
	CheckTimeout      CheckStatus = "timeout"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
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
	DurationMS  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Grade int

type Stats struct {
	QueueToday         int
	ProcessedToday     int
	CleanAttemptRate   float64
	WeakTags           []TagStat
}

type TagStat struct {
	Tag           string
	AverageGrade  float64
}

// --- Interfaces & Services ---

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

type Loader interface {
	LoadAll(rootDir string) ([]CardMeta, error)
}

type Store interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, attempt Attempt, state CardState) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

type NextOptions struct {
	NewPerDay int
	PracticeAhead bool
}

type AttemptInput struct {
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        AttemptStatus
}

type CardView struct {
	Meta        CardMeta
	CurrentCode string
	HintsShown  []string
	StartedAt   time.Time
	Attempt     *Attempt
}

// --- Function Signatures ---

func EvaluateGrade(in AttemptInput) Grade {}
func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}

type SessionService struct{}

func (s *SessionService) Next(ctx context.Context) (CardView, error) {}
func (s *SessionService) Current(ctx context.Context) (CardView, error) {}
func (s *SessionService) Run(ctx context.Context, cardID string, code string) (CheckResult, error) {}
func (s *SessionService) Hint(ctx context.Context) (string, error) {}
func (s *SessionService) Giveup(ctx context.Context, cardID string, code string) error {}
func (s *SessionService) Diff(ctx context.Context) (userCode, solutionCode string, err error) {}

type SandboxManager struct{}

func (m *SandboxManager) Prepare(card CardMeta, files CardFiles) (string, error) {}
func (m *SandboxManager) WriteCode(sessionID, entryFile, code string) error {}
