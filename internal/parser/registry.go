package parser

// Registry tries a sequence of ChapterNameParser strategies, most specific
// first, and reports which one (if any) matched.
type Registry struct {
	parsers []ChapterNameParser
}

// NewRegistry builds a Registry that tries parsers in the given order.
func NewRegistry(parsers ...ChapterNameParser) *Registry {
	return &Registry{parsers: parsers}
}

// DefaultRegistry returns the built-in parsers, ordered from most to least
// specific.
func DefaultRegistry() *Registry {
	return NewRegistry(
		VolChTitleParser{},
		ChapterOnlyParser{},
	)
}

// Parse tries each registered parser in order and returns the first
// confident match, along with the name of the parser that produced it.
// ok is false when no parser recognized dirName; callers should report such
// folders to the user rather than guess.
func (r *Registry) Parse(dirName string) (result ParsedChapter, parserName string, ok bool) {
	for _, p := range r.parsers {
		if parsed, matched := p.Parse(dirName); matched {
			return parsed, p.Name(), true
		}
	}
	return ParsedChapter{}, "", false
}
