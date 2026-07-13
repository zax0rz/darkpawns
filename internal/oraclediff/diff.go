package oraclediff

import (
	"fmt"
	"strings"
)

type diffLine struct {
	prefix byte
	text   string
}

// UnifiedDiff returns a line-oriented unified diff with three lines of context.
// Transcript sizes are small, so a straightforward LCS keeps this dependency-free.
func UnifiedDiff(fromName, toName, from, to string) string {
	if from == to {
		return ""
	}
	a := splitDiffLines(from)
	b := splitDiffLines(to)
	ops := lcsDiff(a, b)

	var out strings.Builder
	fmt.Fprintf(&out, "--- %s\n+++ %s\n", fromName, toName)
	for _, hunk := range diffHunks(ops, 3) {
		aStart, bStart, aCount, bCount := hunkRange(ops, hunk[0], hunk[1])
		fmt.Fprintf(&out, "@@ -%d,%d +%d,%d @@\n", aStart, aCount, bStart, bCount)
		for _, op := range ops[hunk[0]:hunk[1]] {
			fmt.Fprintf(&out, "%c%s\n", op.prefix, op.text)
		}
	}
	return out.String()
}

func splitDiffLines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func lcsDiff(a, b []string) []diffLine {
	lcs := make([][]int, len(a)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(b)+1)
	}
	for i := len(a) - 1; i >= 0; i-- {
		for j := len(b) - 1; j >= 0; j-- {
			if a[i] == b[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else if lcs[i+1][j] >= lcs[i][j+1] {
				lcs[i][j] = lcs[i+1][j]
			} else {
				lcs[i][j] = lcs[i][j+1]
			}
		}
	}

	ops := make([]diffLine, 0, len(a)+len(b))
	for i, j := 0, 0; i < len(a) || j < len(b); {
		switch {
		case i < len(a) && j < len(b) && a[i] == b[j]:
			ops = append(ops, diffLine{' ', a[i]})
			i++
			j++
		case j == len(b) || (i < len(a) && lcs[i+1][j] >= lcs[i][j+1]):
			ops = append(ops, diffLine{'-', a[i]})
			i++
		default:
			ops = append(ops, diffLine{'+', b[j]})
			j++
		}
	}
	return ops
}

func diffHunks(ops []diffLine, context int) [][2]int {
	var hunks [][2]int
	for i := 0; i < len(ops); {
		for i < len(ops) && ops[i].prefix == ' ' {
			i++
		}
		if i == len(ops) {
			break
		}
		start := max(0, i-context)
		end := i + 1
		lastChange := i
		for end < len(ops) {
			if ops[end].prefix != ' ' {
				lastChange = end
			}
			if end-lastChange > context*2 {
				break
			}
			end++
		}
		end = min(len(ops), lastChange+context+1)
		hunks = append(hunks, [2]int{start, end})
		i = end
	}
	return hunks
}

func hunkRange(ops []diffLine, start, end int) (int, int, int, int) {
	aStart, bStart := 1, 1
	for _, op := range ops[:start] {
		if op.prefix != '+' {
			aStart++
		}
		if op.prefix != '-' {
			bStart++
		}
	}
	var aCount, bCount int
	for _, op := range ops[start:end] {
		if op.prefix != '+' {
			aCount++
		}
		if op.prefix != '-' {
			bCount++
		}
	}
	return aStart, bStart, aCount, bCount
}
