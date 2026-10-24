// Command mangabind reorganizes a chapter-by-chapter manga download into one .cbz per volume.
package main

import (
	"context"
	"errors"
	"flag"
	"io"
	"os"
	"os/signal"
	"syscall"
)

// version is set at build/release time via -ldflags "-X main.version=...";
// "dev" is what a plain `go build`/`go run` produces.
var version = "dev"

// exitInterrupted is the exit code of a run that was stopped by Ctrl-C or by
// SIGTERM: 128 plus the number of SIGINT, as shells report it.
const exitInterrupted = 130

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// The first Ctrl-C asks the run to stop and clean up. The handler is then
	// let go, so that a second one is not caught and ends the program at once,
	// for the case where cleaning up is itself taking too long.
	go func() {
		<-ctx.Done()
		stop()
	}()
	os.Exit(runCLIContext(ctx, os.Args[1:], os.Stdout, os.Stderr))
}

func runCLI(args []string, stdout, stderr io.Writer) int {
	return runCLIContext(context.Background(), args, stdout, stderr)
}

// exitCode is what a run that ended with err exits with: 130 when it was
// interrupted, 1 for any other failure.
func exitCode(err error) int {
	if errors.Is(err, context.Canceled) {
		return exitInterrupted
	}
	return 1
}

// runCLIContext is the whole program. ctx is done when the person interrupts
// it: the volume being written is then removed, nothing further is started, and
// the exit code is 130.
func runCLIContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	machineRequested := flagRequested(args, "json") || flagRequested(args, "protocol-version")
	cfg, fs, err := parseFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			// Help that was asked for is the output, so it goes to stdout (and
			// can be piped into a pager); help that was not asked for is an
			// error, and goes to stderr below.
			printUsage(fs, stdout)
			return 0
		}
		if machineRequested {
			// The report on stdout is the whole answer; stderr stays free for
			// the progress stream a consumer may have asked for.
			report := newMachineReport("configuration", false)
			value := issue("error", "invalid_arguments", "configuration", "The command-line arguments are invalid.")
			value.Diagnostic = err.Error()
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else {
			printUsageError(stderr, fs, err)
		}
		return 2
	}

	if cfg.showProtocol {
		if err := writeMachineJSON(stdout, protocolInfo{
			ProtocolVersion: machineProtocolVersion,
			Tool:            "mangabind",
			ToolVersion:     version,
			Capabilities:    []string{"report", "progress-json", "chapter-versions", "chapter-labels", "unit-first-page"},
		}); err != nil {
			printlnTo(stderr, "mangabind: writing protocol information:", err)
			return 1
		}
		return 0
	}

	if cfg.showVersion {
		printlnTo(stdout, "mangabind", version)
		return 0
	}
	if cfg.progressJSON && !cfg.json {
		printlnTo(stderr, "mangabind: -progress-json requires -json")
		return 2
	}

	if cfg.input == "" {
		if cfg.json {
			report := newMachineReport(modeFor(cfg.dryRun), cfg.batch)
			value := issue("error", "missing_input", "configuration", "Choose a manga, CBZ, or library folder to process.")
			value.Recoverable = true
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else if len(args) == 0 {
			// Run with nothing at all: show how to use it.
			printUsage(fs, stderr)
		} else {
			printlnTo(stderr, "mangabind: no -input given")
			printlnTo(stderr, "Usage: mangabind -input <dir> [-output <dir>]")
			printlnTo(stderr, hintForHelp)
		}
		return 2
	}
	if cfg.batch && cfg.metadataFile != "" {
		message := "-metadata-file can't be combined with -batch - place a mangabind.json file inside each manga folder instead"
		if cfg.json {
			report := newMachineReport(modeFor(cfg.dryRun), true)
			value := issue("error", "metadata_file_with_batch", "configuration", "A library batch cannot use one shared metadata file.")
			value.Recoverable = true
			value.Diagnostic = message
			report.addIssue(value)
			_ = writeMachineJSON(stdout, report)
		} else {
			printlnTo(stderr, "mangabind:", message)
		}
		return 2
	}

	output := cfg.output
	if output == "" {
		output = defaultOutputDir(cfg.input)
		if !cfg.quiet && !cfg.json {
			printfTo(stderr, "mangabind: no -output given, writing to %s\n", output)
		}
	}

	if cfg.json {
		var progress progressSink
		if cfg.progressJSON {
			progress = newProgressSink(stderr)
		}
		report, err := runMachine(ctx, cfg, output, progress)
		if writeErr := writeMachineJSON(stdout, report); writeErr != nil {
			printlnTo(stderr, "mangabind: writing JSON report:", writeErr)
			return 1
		}
		if err != nil {
			return exitCode(err)
		}
		return 0
	}

	opts := runOptions{
		output:       output,
		metadataFile: cfg.metadataFile,
		quiet:        cfg.quiet,
		dryRun:       cfg.dryRun,
		combine:      cfg.combine,
		labels:       cfg.labels,
		human:        true,
		stdout:       stdout,
		stderr:       stderr,
	}
	if cfg.batch {
		err = runBatch(ctx, cfg.input, opts)
	} else {
		_, _, err = processManga(ctx, cfg.input, opts)
	}
	if err != nil {
		// An interruption has already been said, where it happened.
		if !errors.Is(err, context.Canceled) {
			printlnTo(stderr, "mangabind:", err)
		}
		return exitCode(err)
	}
	return 0
}

func modeFor(dryRun bool) string {
	if dryRun {
		return "plan"
	}
	return "execute"
}
