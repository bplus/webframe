package demo

import (
	"fmt"
	"strings"

	"github.com/bplus/webframe/components/diffview"
)

// The History tab's diff is demo data: there is no stored history, so the
// "previous version" of an item's notes is derived from the item (the
// seed's first line, a status line for the other status, and a TODO line
// that the current version has dropped) and diffed against the current
// notes plus a status line. The diff itself is real: a line-level longest
// common subsequence, so whatever the user types into the notes shows up as
// additions against the derived previous version.

// notesVersions returns the previous and current text the History tab
// diffs for it.
func notesVersions(it Item) (previous, current string) {
	first := strings.SplitN(seedNotes, "\n", 2)[0]
	otherStatus := "draft"
	if it.Status == "draft" {
		otherStatus = "published"
	}
	previous = first + "\nStatus: " + otherStatus + "\nTODO: describe where it drops."
	current = strings.TrimSpace(it.Notes)
	if current != "" {
		current += "\n"
	}
	current += "Status: " + it.Status
	return previous, current
}

// notesDiffProps declares the History tab's diffview: one file,
// items/<id>/notes.txt, one hunk holding the whole line diff, in the
// inline layout (the panel is narrow enough that side by side would only
// show from a wide desktop; the showcase shows both layouts).
func notesDiffProps(it Item) diffview.Props {
	previous, current := notesVersions(it)
	lines := diffLines(splitLines(previous), splitLines(current))
	return diffview.Props{
		ID: "item-notes-diff",
		Files: []diffview.File{{
			Path:  "items/" + it.ID + "/notes.txt",
			Hunks: []diffview.Hunk{{Header: hunkHeader(lines), Lines: lines}},
		}},
	}
}

// splitLines splits text on newlines; empty text is no lines (not one
// empty line).
func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// diffLines computes a line diff of old against new by longest common
// subsequence: common lines are context, lines only in old are deletions,
// lines only in new are additions, each numbered on its own side.
func diffLines(old, new []string) []diffview.Line {
	n, m := len(old), len(new)
	// lcs[i][j] is the LCS length of old[i:] and new[j:].
	lcs := make([][]int, n+1)
	for i := range lcs {
		lcs[i] = make([]int, m+1)
	}
	for i := n - 1; i >= 0; i-- {
		for j := m - 1; j >= 0; j-- {
			if old[i] == new[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	var out []diffview.Line
	i, j := 0, 0
	for i < n || j < m {
		switch {
		case i < n && j < m && old[i] == new[j]:
			out = append(out, diffview.Line{Kind: diffview.KindContext, OldNo: i + 1, NewNo: j + 1, Text: old[i]})
			i++
			j++
		case i < n && (j >= m || lcs[i+1][j] >= lcs[i][j+1]):
			// Deletions before the additions that replace them, as diff
			// writes a changed line.
			out = append(out, diffview.Line{Kind: diffview.KindDel, OldNo: i + 1, Text: old[i]})
			i++
		default:
			out = append(out, diffview.Line{Kind: diffview.KindAdd, NewNo: j + 1, Text: new[j]})
			j++
		}
	}
	return out
}

// hunkHeader writes the "@@ -1,a +1,b @@" header for a hunk that starts at
// line 1 on both sides, counting the lines present on each.
func hunkHeader(lines []diffview.Line) string {
	oldCount, newCount := 0, 0
	for _, l := range lines {
		if l.Kind != diffview.KindAdd {
			oldCount++
		}
		if l.Kind != diffview.KindDel {
			newCount++
		}
	}
	return fmt.Sprintf("@@ -1,%d +1,%d @@", oldCount, newCount)
}
