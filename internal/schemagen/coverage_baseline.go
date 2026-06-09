package schemagen

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"unicode"
)

type coverageBaseline struct {
	FormatVersion int                     `json:"formatVersion"`
	Entries       []coverageBaselineEntry `json:"entries"`
}

type coverageBaselineEntry struct {
	Release    string      `json:"release"`
	APIVersion string      `json:"apiVersion,omitempty"`
	Kind       string      `json:"kind,omitempty"`
	Path       string      `json:"path,omitempty"`
	Category   gapCategory `json:"category"`
	Reason     string      `json:"reason"`
	Note       string      `json:"note"`
}

func loadCoverageBaseline(path string) (coverageBaseline, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return coverageBaseline{}, fmt.Errorf("read coverage baseline: %w", err)
	}
	var baseline coverageBaseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return coverageBaseline{}, fmt.Errorf("parse coverage baseline: %w", err)
	}
	if baseline.FormatVersion != coverageFormatVersion {
		return coverageBaseline{}, fmt.Errorf("unsupported coverage baseline format %d", baseline.FormatVersion)
	}
	seen := map[string]struct{}{}
	for i, entry := range baseline.Entries {
		if entry.Release == "" || entry.Category == "" || entry.Reason == "" || entry.Note == "" {
			return coverageBaseline{}, fmt.Errorf("baseline entry %d is missing release, category, reason, or note", i)
		}
		key := entry.key()
		if _, ok := seen[key]; ok {
			return coverageBaseline{}, fmt.Errorf("duplicate baseline entry %s", key)
		}
		seen[key] = struct{}{}
	}
	return baseline, nil
}

func (entry coverageBaselineEntry) key() string {
	return entry.Release + "\x00" + entry.APIVersion + "\x00" + entry.Kind + "\x00" + entry.Path + "\x00" + string(entry.Category)
}

func (baseline coverageBaseline) match(gap observedGap) (coverageBaselineEntry, bool) {
	entries := baseline.matchingEntries(gap)
	if len(entries) == 0 {
		return coverageBaselineEntry{}, false
	}
	return entries[0], true
}

func validateCoverageBaselineUse(baseline coverageBaseline, gaps []observedGap) []coverageProblem {
	matched := map[string]struct{}{}
	for _, gap := range gaps {
		for _, entry := range baseline.matchingEntries(gap) {
			matched[entry.key()] = struct{}{}
		}
	}
	var problems []coverageProblem
	for _, entry := range baseline.Entries {
		if _, ok := matched[entry.key()]; ok {
			continue
		}
		problems = append(problems, coverageProblem{Message: formatObsoleteBaselineEntry(entry)})
	}
	return problems
}

func formatObsoleteBaselineEntry(entry coverageBaselineEntry) string {
	return fmt.Sprintf(
		"obsolete baseline entry release=%s apiVersion=%s kind=%s path=%s category=%s reason=%s note=%s",
		strconv.Quote(entry.Release),
		strconv.Quote(entry.APIVersion),
		strconv.Quote(entry.Kind),
		strconv.Quote(entry.Path),
		strconv.Quote(string(entry.Category)),
		strconv.Quote(entry.Reason),
		strconv.Quote(entry.Note),
	)
}

func (baseline coverageBaseline) matchingEntries(gap observedGap) []coverageBaselineEntry {
	var entries []coverageBaselineEntry
	for _, entry := range baseline.Entries {
		if entry.Release != "*" && entry.Release != gap.Release {
			continue
		}
		if entry.APIVersion != gap.APIVersion || entry.Kind != gap.Kind || entry.Path != gap.Path || entry.Category != gap.Category {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

func validateCoverageRatchet(state coverageState, baseline coverageBaseline) []coverageProblem {
	var problems []coverageProblem
	for _, gap := range state.Gaps {
		if _, ok := baseline.match(gap); ok {
			continue
		}
		problems = append(problems, coverageProblem{Message: fmt.Sprintf(
			"unclassified coverage gap release=%s apiVersion=%s kind=%s path=%s category=%s reason=%s",
			gap.Release, gap.APIVersion, gap.Kind, gap.Path, gap.Category, gap.Reason,
		)})
	}
	problems = append(problems, validateCoverageBaselineUse(baseline, state.Gaps)...)
	return problems
}

func formatCoverageProblems(problems []coverageProblem) string {
	if len(problems) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("schema coverage ratchet failed: ")
	for i, problem := range problems {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(escapeControlCharacters(problem.Message))
	}
	return b.String()
}

func escapeControlCharacters(s string) string {
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			if r <= 0xff {
				fmt.Fprintf(&b, "\\x%02x", r)
				continue
			}
			fmt.Fprintf(&b, "\\u%04x", r)
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
