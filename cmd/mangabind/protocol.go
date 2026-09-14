package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	Name         string             `json:"name"`
	InputPath    string             `json:"input_path"`
	MetadataFile string             `json:"metadata_file,omitempty"`
	Status       string             `json:"status"`
	Units        []unitReport       `json:"units"`
	Volumes      []volumeReport     `json:"volumes"`
	Issues       []machineIssue     `json:"issues"`
	Summary      mangaReportSummary `json:"summary"`
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

func runMachine(cfg cliConfig, output string) (machineReport, error) {
	report := newMachineReport(modeFor(cfg.dryRun), cfg.batch)
	report.InputPath = absolutePath(cfg.input)
	report.OutputPath = absolutePath(output)

	if !cfg.batch {
		_, manga, err := processMangaDetailed(
			cfg.input,
			output,
			cfg.metadataFile,
			true,
			cfg.dryRun,
			false,
			io.Discard,
			io.Discard,
		)
		report.addManga(manga)
		return report, err
	}

	entries, err := os.ReadDir(cfg.input)
	if err != nil {
		wrapped := fmt.Errorf("scanning library %s: %w", cfg.input, err)
		value := issue("error", "library_scan_failed", "inspect", "Couldn't inspect the library folder.")
		value.Path = absolutePath(cfg.input)
		value.Recoverable = true
		value.Diagnostic = wrapped.Error()
		report.addIssue(value)
		return report, wrapped
	}

	var errorCount int
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		mangaPath := filepath.Join(cfg.input, entry.Name())
		_, manga, mangaErr := processMangaDetailed(
			mangaPath,
			output,
			"",
			true,
			cfg.dryRun,
			false,
			io.Discard,
			io.Discard,
		)
		report.addManga(manga)
		if mangaErr != nil {
			errorCount++
		}
	}

	if errorCount > 0 {
		return report, fmt.Errorf("%d of %d manga had errors", errorCount, report.Summary.Manga)
	}
	return report, nil
}
