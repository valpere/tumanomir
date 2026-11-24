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
	FormatSkeleton   Format = "skeleton"
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
	DurationMs  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Grade int

type AttemptInput struct {
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        AttemptStatus
}

type Stats struct {
	QueueToday          int
	ProcessedToday      int
	PerfectAttemptRatio float64
	WeakTags            []TagStat
}

type TagStat struct {
	Tag           string
	AverageGrade  float64
}

type NextOptions struct {
	NewPerDay int
}

// --- Service Interfaces & Signatures ---

type Runner interface {
	Check(ctx context.Context, workDir string) (CheckResult, error)
}

type Loader interface {
	LoadAll(rootDir string) ([]CardMeta, error)
}

type SRSStore interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, att Attempt) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

type SandboxManager interface {
	Prepare(cardID string, files CardFiles) (string, error)
	WriteCode(workDir, entryFile, code string) error
	Cleanup(workDir string) error
}

type SessionService interface {
	Next(ctx context.Context) (CardView, error)
	Current(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID string, code string) (CheckResult, error)
	Hint(ctx context.Context) (string, error)
	GiveUp(ctx context.Context, cardID string, code string) error
	Diff(ctx context.Context) (DiffView, error)
}

type CardView struct {
	Meta      CardMeta
	Prompt    string
	Code      string
	StartedAt time.Time
	HintsUsed int
}

type DiffView struct {
	UserCode     string
	SolutionCode string
}

// --- Logic Functions ---

func EvaluateGrade(in AttemptInput) Grade {
	return 0
}

func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {
	return CardState{}
}

func ValidateDecks(dir string) (bool, error) {
	return false, nil
}
