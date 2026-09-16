// Package parser turns raw chapter folder names into structured
// data. Naming conventions vary even within a single manga (see
// docs/adr/0003-pluggable-chapter-parsing.md), so parsing is a chain of
// strategies rather than one fixed pattern.
package parser

// ParsedChapter is the structured result of successfully parsing a chapter
// folder name.
type ParsedChapter struct {
	Volume  *float64 // nil when the folder name carries no volume number
	Chapter float64
	Special string // "x1", "y2", ... for bonus/special chapters; empty otherwise
	Title   string
	Group   string // scanlation group/source, informational only
	Lang    string // language tag, informational only; never drives grouping
}

// ChapterNameParser recognizes one chapter-folder naming convention. Parse
// returns false when it doesn't recognize dirName with enough confidence to
// trust - callers should try the next parser rather than accept a
// low-confidence guess (see docs/adr/0003-pluggable-chapter-parsing.md).
type ChapterNameParser interface {
	// Name identifies the parser/convention, used in diagnostics.
	Name() string
	Parse(dirName string) (ParsedChapter, bool)
}
