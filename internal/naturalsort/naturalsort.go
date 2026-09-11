// Package naturalsort provides human-friendly ("natural") ordering for
// strings that mix text and numbers, e.g. "page2.jpg" before "page10.jpg".
package naturalsort

import (
	"sort"
	"strconv"
)

// Less reports whether a should sort before b in natural order: runs of
// digits are compared numerically, everything else is compared as text.
func Less(a, b string) bool {
	ai, bi := 0, 0
	for ai < len(a) && bi < len(b) {
		ac, bc := a[ai], b[bi]

		if isDigit(ac) && isDigit(bc) {
			aStart, bStart := ai, bi
			for ai < len(a) && isDigit(a[ai]) {
				ai++
			}
			for bi < len(b) && isDigit(b[bi]) {
				bi++
			}
			aDigits, bDigits := a[aStart:ai], b[bStart:bi]

			an, aErr := strconv.ParseUint(aDigits, 10, 64)
			bn, bErr := strconv.ParseUint(bDigits, 10, 64)
			if aErr == nil && bErr == nil {
				if an != bn {
					return an < bn
				}
				continue
			}
			// Numeric run too large for uint64 (unrealistic for manga
			// volume/chapter/page numbers, but handled rather than panicking).
			if aDigits != bDigits {
				return aDigits < bDigits
			}
			continue
		}

		if ac != bc {
			return ac < bc
		}
		ai++
		bi++
	}
	return len(a)-ai < len(b)-bi
}

func isDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// Strings sorts s in place using natural order.
func Strings(s []string) {
	sort.Slice(s, func(i, j int) bool { return Less(s[i], s[j]) })
}
