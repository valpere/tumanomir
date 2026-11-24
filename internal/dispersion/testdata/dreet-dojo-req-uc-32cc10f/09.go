package dreet

import (
	"context"
	"time"
)

// --- Deck Types ---

type Format string

const (
	FormatCloze       Format = "cloze"
	FormatSpec        Format = "spec"
	FormatProgressive Format = "progressive"
	FormatBugfix      Format = "bugfix"
	FormatSkeleton    Format = "skeleton"
)

type CardMeta struct {
	ID        string   `yaml:"id"`
	Lang      string   `yaml:"lang"`
	Title     string   `yaml:"title"`
	Format    Format   `yaml:"format"`
	Prereq    string   `yaml:"prereq,omitempty"`
	Order     int      `yaml:"order"`
	TargetSec int      `yaml:"target_sec"`
	Tags      []string `yaml:"tags"`
	EntryFile string   `yaml:"entry_file"`
	Hints     []string `yaml:"hints"`
}

type CardFiles struct {
	Meta     CardMeta
	Prompt   string
	Module   string
	Stub     string
	Tests    []string
	Solution string
}

type Loader interface {
	LoadAll(rootDir string) ([]CardFiles, error)
}

// --- SRS Types ---

type Grade int

const (
	GradeAgain Grade = 1
	GradeHard  Grade = 2
	GradeGood  Grade = 3
	GradeEasy  Grade = 4
)

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
	Status        string // "pass", "giveup"
	Grade         Grade
}

type AttemptInput struct {
	ElapsedSec    int
	TargetSec     int
	CompileErrors int
	TestFails     int
	HintsUsed     int
	Status        string
}

type NextOptions struct {
	NewPerDay int
}

type Stats struct {
	QueueToday       int
	ReviewedToday    int
	CleanPassRate    float64
	WeakTags         []TagStat
}

type TagStat struct {
	Tag            string
	AverageGrade   float64
}

type Store interface {
	NextDue(ctx context.Context, now time.Time, opts NextOptions) (CardState, error)
	RecordAttempt(ctx context.Context, att Attempt, state CardState) error
	Stats(ctx context.Context, now time.Time, newPerDay int) (Stats, error)
}

// --- Runner Types ---

type Status string

const (
	StatusPass         Status = "pass"
	StatusCompileError Status = "compile_error"
	StatusTestFail     Status = "test_fail"
	StatusTimeout      Status = "timeout"
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

type Result struct {
	Status      Status
	DurationMs  int
	Diagnostics []Diagnostic
	RawOutput   string
}

type Runner interface {
	Check(ctx context.Context, workDir string) (Result, error)
}

type Registry interface {
	GetRunner(lang string) (Runner, bool)
}

// --- Sandbox Types ---

type Workspace struct {
	Dir       string
	EntryFile string
}

type Manager interface {
	Prepare(card CardFiles) (Workspace, error)
	WriteCode(ws Workspace, code string) error
	Cleanup(ws Workspace) error
}

// --- Session Types ---

type CardView struct {
	Meta          CardMeta
	Prompt        string
	CurrentCode   string
	StartedAt     time.Time
	HintsUsed     int
	ShownHints    []string
	LastResult    *Result
	Grade         *Grade
	IntervalDays   *float64
}

type Service interface {
	Next(ctx context.Context) (CardView, error)
	Current(ctx context.Context) (CardView, error)
	Run(ctx context.Context, cardID string, code string) (Result, error)
	Hint(ctx context.Context, cardID string) (string, error)
	GiveUp(ctx context.Context, cardID string, code string) error
	Diff(ctx context.Context, cardID string) (userCode, solutionCode string, err error)
}

// --- Logic Signatures ---

func EvaluateGrade(in AttemptInput) Grade {}
func ScheduleSM2(st CardState, g Grade, now time.Time, rnd func() float64) CardState {}
