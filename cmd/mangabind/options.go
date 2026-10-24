package main

import (
	"io"

	"github.com/gustavommcv/mangabind/internal/grouper"
)

// runOptions is what a run was asked to do and where it says things, carried from the command line down to
// each manga. The context travels apart from it, as the first argument.
type runOptions struct {
	// output is the folder the volumes are written to.
	output string
	// metadataFile is the mapping file named on the command line; empty, each manga looks for its own
	// mangabind.json.
	metadataFile string

	quiet, dryRun, combine bool

	// labels is how the chapters are named inside a volume, and so in the table of contents of its book.
	labels grouper.LabelStyle

	// human is whether the run says things for people: the lines on stdout and stderr. A run that answers
	// with a JSON report keeps them off, and its writers are io.Discard.
	human          bool
	stdout, stderr io.Writer

	// progress, when not nil, hears of each stage and of the pages as they are written.
	progress progressSink
}

// forManga is the options for one manga of a library: no mapping file is named for it, so it looks for
// the mangabind.json inside its own folder.
func (o runOptions) forManga() runOptions {
	o.metadataFile = ""
	return o
}
