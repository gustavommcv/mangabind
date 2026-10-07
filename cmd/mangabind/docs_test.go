package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// repositoryRoot is the folder of go.mod, from the test's own folder.
func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// TestTheProtocolDocumentListsEveryIssueCodeAndOnlyThose keeps the document the
// consumers read from drifting from the code that emits the codes: a code that
// is emitted and not documented, or documented and never emitted, fails here.
func TestTheProtocolDocumentListsEveryIssueCodeAndOnlyThose(t *testing.T) {
	root := repositoryRoot(t)

	emitted := map[string]bool{}
	issueCall := regexp.MustCompile(`issue\(\s*"(?:error|warning)"\s*,\s*"([a-z_]+)"`)
	codeAssignment := regexp.MustCompile(`\.Code = "([a-z_]+)"`)
	sources, err := filepath.Glob(filepath.Join(root, "cmd", "mangabind", "*.go"))
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range sources {
		if strings.HasSuffix(source, "_test.go") {
			continue
		}
		text := readText(t, source)
		for _, match := range issueCall.FindAllStringSubmatch(text, -1) {
			emitted[match[1]] = true
		}
		for _, match := range codeAssignment.FindAllStringSubmatch(text, -1) {
			emitted[match[1]] = true
		}
	}
	if len(emitted) < 20 {
		t.Fatalf("found only %d issue codes in the source (%v): the pattern no longer matches how they are written", len(emitted), sortedKeys(emitted))
	}

	documented := map[string]bool{}
	row := regexp.MustCompile("(?m)^\\| `([a-z_]+)` \\| ")
	for _, match := range row.FindAllStringSubmatch(readText(t, filepath.Join(root, "docs", "machine-protocol-v1.md")), -1) {
		documented[match[1]] = true
	}

	for _, code := range sortedKeys(emitted) {
		if !documented[code] {
			t.Errorf("the code %q is emitted but is not in the table of docs/machine-protocol-v1.md", code)
		}
	}
	for _, code := range sortedKeys(documented) {
		if !emitted[code] {
			t.Errorf("the code %q is in the table of docs/machine-protocol-v1.md but nothing emits it", code)
		}
	}
}

// TestMarkdownLinksPointAtFiles catches the links that a rename or a rewrite
// leaves behind: every relative link in the documents must name a file or a
// folder that exists. Links to other sites and to anchors are not checked.
func TestMarkdownLinksPointAtFiles(t *testing.T) {
	root := repositoryRoot(t)
	var documents []string
	for _, pattern := range []string{"*.md", "docs/*.md", "docs/adr/*.md"} {
		matches, err := filepath.Glob(filepath.Join(root, filepath.FromSlash(pattern)))
		if err != nil {
			t.Fatal(err)
		}
		documents = append(documents, matches...)
	}
	if len(documents) < 20 {
		t.Fatalf("found only %d documents", len(documents))
	}

	link := regexp.MustCompile(`\]\(([^)\s]+)\)`)
	for _, document := range documents {
		for _, match := range link.FindAllStringSubmatch(readText(t, document), -1) {
			target := match[1]
			if strings.Contains(target, "://") || strings.HasPrefix(target, "#") || strings.HasPrefix(target, "mailto:") {
				continue
			}
			target, _, _ = strings.Cut(target, "#")
			if target == "" {
				continue
			}
			resolved := filepath.Join(filepath.Dir(document), filepath.FromSlash(target))
			if _, err := os.Stat(resolved); err != nil {
				relative, _ := filepath.Rel(root, document)
				t.Errorf("%s links to %q, which does not exist", filepath.ToSlash(relative), match[1])
			}
		}
	}
}
