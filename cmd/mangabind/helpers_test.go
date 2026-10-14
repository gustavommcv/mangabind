package main

import (
	"context"
	"io"
	"os"
)

// The tests that check what a run writes call these with the few things they care about; the options they
// build are those of a run on a terminal.

func testOptions(output, metadataFile string, quiet, dryRun, combine bool, stdout, stderr io.Writer) runOptions {
	return runOptions{
		output:       output,
		metadataFile: metadataFile,
		quiet:        quiet,
		dryRun:       dryRun,
		combine:      combine,
		human:        true,
		stdout:       stdout,
		stderr:       stderr,
	}
}

func processMangaWithOutput(ctx context.Context, input, output, metadataFile string, quiet, dryRun, combine bool, stdout, stderr io.Writer) (mangaSummary, error) {
	summary, _, err := processManga(ctx, input, testOptions(output, metadataFile, quiet, dryRun, combine, stdout, stderr))
	return summary, err
}

func processMangaOnTerminal(input, output, metadataFile string, quiet, dryRun bool) (mangaSummary, error) {
	return processMangaWithOutput(context.Background(), input, output, metadataFile, quiet, dryRun, false, os.Stdout, os.Stderr)
}

func runBatchOnTerminal(libraryInput, output string, quiet, dryRun bool) error {
	return runBatch(context.Background(), libraryInput, testOptions(output, "", quiet, dryRun, false, os.Stdout, os.Stderr))
}
