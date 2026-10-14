package main

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/scanner"
)

// runBatch treats libraryInput as a folder of manga folders, processing each
// immediate subdirectory independently: one manga failing (or producing no
// volumes) is reported but never stops the rest of the batch, the same way
// one bad chapter never stops the rest of a single manga's volumes.
func runBatch(ctx context.Context, libraryInput string, opts runOptions) error {
	stdout, stderr := opts.stdout, opts.stderr
	mangaNames, links, err := scanner.Library(libraryInput)
	if err != nil {
		return fmt.Errorf("scanning library %s: %w", libraryInput, err)
	}
	// A link to a manga folder is not a manga of this run: say so, rather than
	// leave a folder out without a word.
	for _, link := range links {
		printlnTo(stderr, "warning:", libraryLinkMessage(link))
	}

	var mangaCount, errCount, totalVolumes, totalPages int

	for _, name := range mangaNames {
		// An interruption ends the batch: the manga that are left are not tried.
		if err := ctx.Err(); err != nil {
			printlnTo(stderr, "mangabind:", interruptedBeforeNextManga)
			return fmt.Errorf("interrupted: %w", err)
		}
		mangaCount++
		mangaPath := filepath.Join(libraryInput, name)
		if !opts.quiet {
			printfTo(stdout, "== %s ==\n", name)
		}

		// No explicit override in batch mode - each manga only picks up its
		// own mangabind.json convention file, if it has one.
		summary, _, err := processManga(ctx, mangaPath, opts.forManga())
		if errors.Is(err, context.Canceled) {
			return err
		}
		// What a manga wrote counts, even when it also had an error: the volumes
		// that were bound before one failed are on disk, and the tally is of the
		// disk.
		totalVolumes += summary.volumes
		totalPages += summary.pages
		if err != nil {
			errCount++
			printfTo(stderr, "mangabind: %s: %v\n", name, err)
		}
	}

	printfTo(stdout, "mangabind: processed %d manga, %d volume(s), %d page(s) total\n", mangaCount, totalVolumes, totalPages)
	if errCount > 0 {
		return fmt.Errorf("%d of %d manga had errors, see above", errCount, mangaCount)
	}
	return nil
}

// interruptedBeforeNextManga is said when a batch is stopped between two manga.
const interruptedBeforeNextManga = "Interrupted: the manga that were left were not started."
