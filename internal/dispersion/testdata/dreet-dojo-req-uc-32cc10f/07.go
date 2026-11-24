package dreetdojo

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

type Status string

const (
	StatusPass         Status = "pass"
	StatusCompileError Status = "compile_error"
	StatusTestFail     Status = "test_fail"
	StatusTimeout      Status = "timeout"
	StatusGiveup       Status = "giveup"
)

type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

type Grade int

const (
	Grade1 Grade = 1
	Grade2 Grade = 2
	Grade3 Grade = 3
	Grade4 Grade = 4
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

type Attempt struct {
	ID            int
	CardID        string
	StartedAt     time.Time
	FinishedAt   time.Time
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        Status
	Grade         Grade
}

type Diagnostic struct {
	File     string
	Line     int
	Col      int
	Severity Severity
	Message  string
}

type CheckResult struct {
	Status      Status
	DurationMS  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Stats struct {
	QueueToday           int
	ProcessedToday       int
	CleanAttemptFraction float64
	WeakTags             []TagStat
}

type TagStat struct {
	Tag         string
	AverageGrade float64
}

type CardView struct {
	Meta      CardMeta
	Code      string
	Prompt    string
	StartedAt time.Time
	HintsUsed []string
	Grade     *Grade
}

// --- Interfaces & Services ---

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

type SRSStore interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, attempt Attempt) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

type NextOptions struct {
	NewPerDay int
}

type Loader interface {
	LoadAll(rootDir string) ([]CardMeta, error)
}

type SessionService interface {
	Next(ctx context.Context) (CardView, error)
	Current(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID string, code string) (CheckResult, error)
	Hint(ctx context.Context) (string, error)
	Giveup(ctx context.Context, cardID string, code string) error
	Diff(ctx context.Context) (userCode, solutionCode string, err error)
}

type SandboxManager interface {
	Prepare(ctx context.Context, cardID string) (string, error)
	WriteCode(ctx context.Context, workDir, fileName, code string) error
	Cleanup(ctx context.Context, workDir string) error
}

// --- Function Signatures ---

func EvaluateGrade(compileErrors, testFails, hintsUsed, elapsedSec, targetSec int, status Status) Grade {}

func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}

func ValidateDecks(decksDir string) (int, error) {}
