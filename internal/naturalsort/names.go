package naturalsort

import (
	"sort"
	"strings"
	"unicode/utf8"
)

// This file puts the names of the files of a folder in the order KCC (Kindle
// Comic Converter) puts them: its natsort library's "operating system" order.
// A person who runs KCC on the same folder gets the pages in the same order, which
// is what the book Mangabind makes should have too. What it does to a name:
//
//   - The name is split in a stem and its extensions, and the stem is compared
//     first: "p01.png" before "p01 (2).png", and "1.png" before "1.5.png". Up to
//     two extensions are split off, each of at most five characters, none that
//     starts with a digit ("1.5" is a stem, ".png" an extension).
//   - A number is a run of decimal digits of any script, or one character that
//     stands for a digit ("²", "①"). It can be any size.
//   - Case is ignored. Text between numbers is compared by code point, which is
//     what natsort gets on a system whose locale collates by code point; on any
//     other it follows the locale, so the order of punctuation and accents is the
//     one thing that can differ from KCC there.
//
// The order is checked against the real library: testdata/natsort_vectors.txt
// holds the order natsort 8.4.0 gives for 1,657 names, written by
// tools/parity/natsort_vectors.py in the mangapress repository.

// part is one piece of a key: text or a number, taking turns, starting with text
// (empty when the name starts with a number).
type part struct {
	number bool
	text   string // for a number, its digits without leading zeros
}

func comparePart(a, b part) int {
	if a.number != b.number {
		if !a.number {
			return -1
		}
		return 1
	}
	if a.number && len(a.text) != len(b.text) {
		if len(a.text) < len(b.text) {
			return -1
		}
		return 1
	}
	return strings.Compare(a.text, b.text)
}

func number(digits []byte) part {
	trimmed := strings.TrimLeft(string(digits), "0")
	if trimmed == "" {
		trimmed = "0"
	}
	return part{number: true, text: trimmed}
}

// pieceKey is the key of one piece of a name: its text and numbers, in turn.
func pieceKey(piece string) []part {
	var (
		parts []part
		text  strings.Builder
		run   []byte
	)
	endRun := func() {
		if len(run) > 0 {
			parts = append(parts, part{text: text.String()}, number(run))
			text.Reset()
			run = run[:0]
		}
	}
	for _, r := range strings.ToLower(piece) {
		if value, ok := decimalValue(r); ok {
			run = append(run, byte('0'+value))
			continue
		}
		endRun()
		if value, ok := singleValue(r); ok {
			parts = append(parts, part{text: text.String()}, number([]byte{byte('0' + value)}))
			text.Reset()
			continue
		}
		text.WriteRune(r)
	}
	endRun()
	if text.Len() > 0 {
		parts = append(parts, part{text: text.String()})
	}
	return parts
}

// extensions are the extensions of a name that are split from its stem: the
// last two at most, each of at most five characters, stopping at one that starts
// with a digit.
func extensions(name string) []string {
	var all []string
	if fields := strings.Split(strings.TrimLeft(name, "."), "."); len(fields) > 1 {
		for _, field := range fields[1:] {
			all = append(all, "."+field)
		}
	}
	var kept []string
	for position := range all {
		extension := all[len(all)-1-position]
		startsWithDigit := false
		if _, size := utf8.DecodeRuneInString(extension); size < len(extension) {
			next, _ := utf8.DecodeRuneInString(extension[size:])
			_, startsWithDigit = decimalValue(next)
		}
		if startsWithDigit || position > 1 || utf8.RuneCountInString(extension) > 5 {
			break
		}
		kept = append([]string{extension}, kept...)
	}
	return kept
}

// nameKey is the key of one file or folder name (not a path): its stem, then
// each extension.
func nameKey(name string) [][]part {
	exts := extensions(name)
	stem := name
	if joined := strings.Join(exts, ""); joined != "" {
		stem = strings.ReplaceAll(name, joined, "")
	}
	var key [][]part
	for _, piece := range append([]string{stem}, exts...) {
		if piece != "" {
			key = append(key, pieceKey(piece))
		}
	}
	return key
}

func compareKeys(a, b [][]part) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		for j := 0; j < len(a[i]) && j < len(b[i]); j++ {
			if c := comparePart(a[i][j], b[i][j]); c != 0 {
				return c
			}
		}
		if c := len(a[i]) - len(b[i]); c != 0 {
			return sign(c)
		}
	}
	return sign(len(a) - len(b))
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	}
	return 0
}

// LessName reports whether the file or folder name a comes before b in KCC's
// order. A name is not a path: use LessPath for entries of an archive.
func LessName(a, b string) bool {
	return compareKeys(nameKey(a), nameKey(b)) < 0
}

// LessPath reports whether the path a comes before b in KCC's order, component
// by component (separated by "/"), so that a number in one folder name is never
// compared with a number in another level. Where one path has a file and the
// other a folder at the same level, the file comes first: a folder's own pages
// come before the folders inside it.
func LessPath(a, b string) bool {
	left, right := strings.Split(a, "/"), strings.Split(b, "/")
	for level := 0; level < len(left) && level < len(right); level++ {
		leftKey, rightKey := nameKey(left[level]), nameKey(right[level])
		c := compareKeys(leftKey, rightKey)
		if c == 0 {
			continue
		}
		leftIsFile, rightIsFile := level+1 == len(left), level+1 == len(right)
		switch {
		case leftIsFile && !rightIsFile:
			return true
		case !leftIsFile && rightIsFile:
			return false
		}
		return c < 0
	}
	return len(left) < len(right)
}

// Names sorts file names in place in KCC's order. Names that are the same to it
// ("A.png" and "a.png") keep the order they had.
func Names(s []string) {
	sort.SliceStable(s, func(i, j int) bool { return LessName(s[i], s[j]) })
}

// Paths sorts paths in place in KCC's order, as LessPath orders them.
func Paths(s []string) {
	sort.SliceStable(s, func(i, j int) bool { return LessPath(s[i], s[j]) })
}
