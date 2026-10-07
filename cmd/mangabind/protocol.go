package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"

	"github.com/gustavommcv/mangabind/internal/scanner"
)

// machineProtocolVersion changes only when a consumer must change how it
// interprets Mangabind's structured output. It is deliberately independent
// from the executable's release version.
const machineProtocolVersion = 1

type protocolInfo struct {
	ProtocolVersion int      `json:"protocol_version"`
	Tool            string   `json:"tool"`
	ToolVersion     string   `json:"tool_version"`
	Capabilities    []string `json:"capabilities"`
}

// Progress is an opt-in stderr side channel. It never changes the version 1
// stdout report or the default human-readable CLI behavior.
type machineProgress struct {
	ProtocolVersion int      `json:"protocol_version"`
	Tool            string   `json:"tool"`
	ToolVersion     string   `json:"tool_version"`
	Kind            string   `json:"kind"`
	Stage           string   `json:"stage"`
	State           string   `json:"state"`
	Manga           string   `json:"manga"`
	VolumeIndex     int      `json:"volume_index,omitempty"`
	VolumeCount     int      `json:"volume_count,omitempty"`
	VolumeNumber    *float64 `json:"volume_number,omitempty"`
	CompletedPages  *int     `json:"completed_pages,omitempty"`
	TotalPages      int      `json:"total_pages,omitempty"`
}

type progressSink func(machineProgress)

func newProgressSink(writer io.Writer) progressSink {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)
	return func(event machineProgress) {
		event.ProtocolVersion = machineProtocolVersion
		event.Tool = "mangabind"
		event.ToolVersion = version
		event.Kind = "progress"
		// Progress is advisory; the stdout report and exit code remain authoritative.
		_ = encoder.Encode(event)
	}
}

type machineReport struct {
	ProtocolVersion int            `json:"protocol_version"`
	Tool            string         `json:"tool"`
	ToolVersion     string         `json:"tool_version"`
	Kind            string         `json:"kind"`
	Mode            string         `json:"mode"`
	Status          string         `json:"status"`
	InputPath       string         `json:"input_path,omitempty"`
	OutputPath      string         `json:"output_path,omitempty"`
	Batch           bool           `json:"batch"`
	Manga           []mangaReport  `json:"manga"`
	Issues          []machineIssue `json:"issues"`
	Summary         machineSummary `json:"summary"`
}

type machineSummary struct {
	Manga    int `json:"manga"`
	Volumes  int `json:"volumes"`
	Pages    int `json:"pages"`
	Errors   int `json:"errors"`
	Warnings int `json:"warnings"`
}

type mangaReport struct {
	Name      string `json:"name"`
	InputPath string `json:"input_path"`
	// CombinedOutputPath is set instead of each volume having its own written
	// file when -combine was used: every volumeReport below still describes
	// its own chapters, page count, and number, but its content physically
	// lives inside this one file. Additive field, no protocol version bump -
	// see docs/adr/0012-combine-series-into-one-volume.md.
	CombinedOutputPath string             `json:"combined_output_path,omitempty"`
	MetadataFile       string             `json:"metadata_file,omitempty"`
	Status             string             `json:"status"`
	Units              []unitReport       `json:"units"`
	Volumes            []volumeReport     `json:"volumes"`
	Issues             []machineIssue     `json:"issues"`
	Summary            mangaReportSummary `json:"summary"`
}

type mangaReportSummary struct {
	Units    int `json:"units"`
	Volumes  int `json:"volumes"`
	Pages    int `json:"pages"`
	Warnings int `json:"warnings"`
	Errors   int `json:"errors"`
}

type unitReport struct {
	Name               string                   `json:"name"`
	Path               string                   `json:"path"`
	Kind               string                   `json:"kind"`
	PageCount          int                      `json:"page_count"`
	Parser             parserReport             `json:"parser"`
	MetadataAssignment metadataAssignmentReport `json:"metadata_assignment"`
	EffectiveVolume    *float64                 `json:"effective_volume,omitempty"`
	Disposition        string                   `json:"disposition"`
}

type parserReport struct {
	Matched  bool     `json:"matched"`
	Name     string   `json:"name,omitempty"`
	Volume   *float64 `json:"volume,omitempty"`
	Chapter  *float64 `json:"chapter,omitempty"`
	Special  string   `json:"special,omitempty"`
	Title    string   `json:"title,omitempty"`
	Group    string   `json:"group,omitempty"`
	Language string   `json:"language,omitempty"`
}

type metadataAssignmentReport struct {
	Status         string   `json:"status"`
	MetadataVolume *float64 `json:"metadata_volume,omitempty"`
}

type volumeReport struct {
	Number     float64  `json:"number"`
	OutputPath string   `json:"output_path"`
	PageCount  int      `json:"page_count"`
	Chapters   []string `json:"chapters"`
	Written    bool     `json:"written"`
}

// machineIssue is shared by planning and execution reports. Stable code and
// context fields are for software; message and diagnostic are for people.
type machineIssue struct {
	Tool         string   `json:"tool"`
	Severity     string   `json:"severity"`
	Code         string   `json:"code"`
	Stage        string   `json:"stage"`
	Manga        string   `json:"manga,omitempty"`
	Volume       *float64 `json:"volume,omitempty"`
	Chapter      *float64 `json:"chapter,omitempty"`
	Special      string   `json:"special,omitempty"`
	Path         string   `json:"path,omitempty"`
	RelatedPaths []string `json:"related_paths,omitempty"`
	Recoverable  bool     `json:"recoverable"`
	Message      string   `json:"message"`
	Diagnostic   string   `json:"diagnostic,omitempty"`
}

func newMachineReport(mode string, batch bool) machineReport {
	return machineReport{
		ProtocolVersion: machineProtocolVersion,
		Tool:            "mangabind",
		ToolVersion:     version,
		Kind:            "report",
		Mode:            mode,
		Status:          "completed",
		Batch:           batch,
		Manga:           []mangaReport{},
		Issues:          []machineIssue{},
	}
}

func newMangaReport(name, input string) mangaReport {
	return mangaReport{
		Name:      name,
		InputPath: input,
		Status:    "completed",
		Units:     []unitReport{},
		Volumes:   []volumeReport{},
		Issues:    []machineIssue{},
	}
}

func issue(severity, code, stage, message string) machineIssue {
	return machineIssue{
		Tool:        "mangabind",
		Severity:    severity,
		Code:        code,
		Stage:       stage,
		Recoverable: severity != "error",
		Message:     message,
	}
}

func (r *mangaReport) addIssue(value machineIssue) {
	r.Issues = append(r.Issues, value)
	switch value.Severity {
	case "error":
		r.Summary.Errors++
		r.Status = "failed"
	case "warning":
		r.Summary.Warnings++
		if r.Status == "completed" {
			r.Status = "completed_with_warnings"
		}
	}
}

func (r *machineReport) addIssue(value machineIssue) {
	r.Issues = append(r.Issues, value)
	switch value.Severity {
	case "error":
		r.Summary.Errors++
		r.Status = "failed"
	case "warning":
		r.Summary.Warnings++
		if r.Status == "completed" {
			r.Status = "completed_with_warnings"
		}
	}
}

func (r *machineReport) addManga(value mangaReport) {
	r.Manga = append(r.Manga, value)
	r.Summary.Manga++
	r.Summary.Volumes += value.Summary.Volumes
	r.Summary.Pages += value.Summary.Pages
	r.Summary.Errors += value.Summary.Errors
	r.Summary.Warnings += value.Summary.Warnings
	if value.Status == "failed" {
		r.Status = "failed"
	} else if value.Status == "completed_with_warnings" && r.Status == "completed" {
		r.Status = "completed_with_warnings"
	}
}

func writeMachineJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func finalizeMangaReport(report *mangaReport, summary mangaSummary) {
	report.Summary.Units = len(report.Units)
	report.Summary.Volumes = summary.volumes
	report.Summary.Pages = summary.pages
}

func runMachine(ctx context.Context, cfg cliConfig, output string, progress progressSink) (machineReport, error) {
	report := newMachineReport(modeFor(cfg.dryRun), cfg.batch)
	report.InputPath = absolutePath(cfg.input)
	report.OutputPath = absolutePath(output)

	if !cfg.batch {
		_, manga, err := processMangaDetailed(
			ctx,
			cfg.input,
			output,
			cfg.metadataFile,
			true,
			cfg.dryRun,
			cfg.combine,
			false,
			io.Discard,
			io.Discard,
			progress,
		)
		report.addManga(manga)
		return report, err
	}

	mangaNames, links, err := scanner.Library(cfg.input)
	if err != nil {
		wrapped := fmt.Errorf("scanning library %s: %w", cfg.input, err)
		value := issue("error", "library_scan_failed", "inspect", "Couldn't inspect the library folder.")
		value.Path = absolutePath(cfg.input)
		value.Recoverable = true
		value.Diagnostic = wrapped.Error()
		report.addIssue(value)
		return report, wrapped
	}

	// A link to a manga folder is not a manga of this run: report it, rather
	// than leave a folder out without a word.
	for _, link := range links {
		value := issue("warning", "link_skipped", "inspect", libraryLinkMessage(link))
		value.Path = absolutePath(link.Path)
		report.addIssue(value)
	}

	var errorCount int
	for _, name := range mangaNames {
		// An interruption ends the batch: the manga that are left are not tried.
		if cause := ctx.Err(); cause != nil {
			value := issue("error", "interrupted", "inspect", interruptedBeforeNextManga)
			value.Path = absolutePath(cfg.input)
			value.Recoverable = true
			report.addIssue(value)
			return report, fmt.Errorf("interrupted: %w", cause)
		}
		mangaPath := filepath.Join(cfg.input, name)
		_, manga, mangaErr := processMangaDetailed(
			ctx,
			mangaPath,
			output,
			"",
			true,
			cfg.dryRun,
			cfg.combine,
			false,
			io.Discard,
			io.Discard,
			progress,
		)
		report.addManga(manga)
		if errors.Is(mangaErr, context.Canceled) {
			return report, mangaErr
		}
		if mangaErr != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		return report, fmt.Errorf("%d of %d manga had errors", errorCount, report.Summary.Manga)
	}
	return report, nil
}
