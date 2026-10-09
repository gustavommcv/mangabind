package naturalsort

import "unicode"

// otherDigits are the characters that stand for a digit without being decimal
// digits (superscripts, circled and parenthesized digits and the like), as
// {first, last, value of first}: each next character is worth one more. They
// are the ones the natsort library counts as numbers, one at a time. The
// decimal digits of every script come from the unicode package.
var otherDigits = [...][3]rune{
	{0xb2, 0xb3, 2}, {0xb9, 0xb9, 1}, {0x1369, 0x1371, 1}, {0x19da, 0x19da, 1},
	{0x2070, 0x2070, 0}, {0x2074, 0x2079, 4}, {0x2080, 0x2089, 0},
	{0x2460, 0x2468, 1}, {0x2474, 0x247c, 1}, {0x2488, 0x2490, 1},
	{0x24ea, 0x24ea, 0}, {0x24f5, 0x24fd, 1}, {0x24ff, 0x24ff, 0},
	{0x2776, 0x277e, 1}, {0x2780, 0x2788, 1}, {0x278a, 0x2792, 1},
	{0x10a40, 0x10a43, 1}, {0x10e60, 0x10e68, 1}, {0x11052, 0x1105a, 1},
	{0x1f100, 0x1f100, 0}, {0x1f101, 0x1f10a, 0},
}

// decimalValue is the value of r when it is a decimal digit of any script
// (Unicode category Nd). Every block of decimal digits is ten characters in a
// row starting at zero, and the unicode tables join neighbouring blocks, so a
// digit is worth its distance from the start of its range, modulo ten.
func decimalValue(r rune) (int, bool) {
	if r >= '0' && r <= '9' {
		return int(r - '0'), true
	}
	if r < 0x80 || !unicode.Is(unicode.Nd, r) {
		return 0, false
	}
	for _, rg := range unicode.Nd.R16 {
		if r >= rune(rg.Lo) && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	for _, rg := range unicode.Nd.R32 {
		if r >= rune(rg.Lo) && r <= rune(rg.Hi) {
			return int(r-rune(rg.Lo)) % 10, true
		}
	}
	return 0, false
}

// singleValue is the value of r when it stands for a digit without being one.
func singleValue(r rune) (int, bool) {
	for _, rg := range otherDigits {
		if r >= rg[0] && r <= rg[1] {
			return int(rg[2] + r - rg[0]), true
		}
	}
	return 0, false
}
