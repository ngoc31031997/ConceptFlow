package domain

import (
	"fmt"
	"math"
	"strings"
)

// NarrationBudgetTolerance is how far a scene's estimated narration may fall
// outside its beat's budget before the Creator is warned (CR-048 T8). Wider
// than the ±15% the outline prompt allows, because this compares an estimate
// (words at a speaking rate) with a budget, and the estimate carries its own
// error.
const NarrationBudgetTolerance = 0.20

// CheckNarrationBudgets estimates how long each storyboard scene's narration
// takes to read and compares it with the budget the video format gives that
// scene's beat (CR-048 T8). It returns one Creator-facing warning per scene
// that is off by more than NarrationBudgetTolerance, and one per scene whose id
// is not a beat of the format. Warnings only: the numbers are estimates.
//
// The rate is BeatSheetWordsPerMinute(language, calibratedWPM) — the same one
// the outline prompt converted the beat budgets into words with — and words
// are counted the way EstimateNarrationDuration counts them.
//
// Narration is grouped by scene id. A beat the format lets repeat (`variation`)
// may appear as several scenes with the same id; their narration is summed and
// the budget is multiplied by the number of scenes carrying the id, so two
// legitimate variations do not read as one over-long one.
func CheckNarrationBudgets(scenes []StoryboardScene, format VideoFormat, language string, calibratedWPM float64) []string {
	wpm := BeatSheetWordsPerMinute(language, calibratedWPM)

	beats := make(map[string]FormatBeat, len(format.Beats))
	for _, beat := range format.Beats {
		beats[beat.ID] = beat
	}

	type group struct {
		id     string
		scenes int
		words  int
	}
	var order []*group
	byID := map[string]*group{}
	for _, scene := range scenes {
		id := strings.TrimSpace(scene.ID)
		g := byID[id]
		if g == nil {
			g = &group{id: id}
			byID[id] = g
			order = append(order, g)
		}
		g.scenes++
		for _, shot := range scene.Shots {
			g.words += countWords(shot.Narration)
		}
	}

	var warnings []string
	for _, g := range order {
		seconds := float64(g.words) / wpm * 60.0
		label := sceneLabel(g.id, g.scenes)
		beat, known := beats[g.id]
		if !known {
			warnings = append(warnings, fmt.Sprintf(
				"%s: không có beat này trong format %q — không so được với ngân sách (~%d giây lời thoại)",
				label, format.Name, roundSeconds(seconds)))
			continue
		}
		low := beat.MinSeconds * float64(g.scenes)
		high := beat.MaxSeconds * float64(g.scenes)
		switch {
		case high > 0 && seconds > high*(1+NarrationBudgetTolerance):
			warnings = append(warnings, fmt.Sprintf("%s: ~%d giây, ngân sách %s (+%d%%)",
				label, roundSeconds(seconds), budgetRange(low, high), percent((seconds-high)/high)))
		case low > 0 && seconds < low*(1-NarrationBudgetTolerance):
			warnings = append(warnings, fmt.Sprintf("%s: ~%d giây, ngân sách %s (-%d%%)",
				label, roundSeconds(seconds), budgetRange(low, high), percent((low-seconds)/low)))
		}
	}
	return warnings
}

func sceneLabel(id string, count int) string {
	name := "Cảnh " + id
	if id == "" {
		name = "Cảnh không có id"
	}
	if count > 1 {
		name += fmt.Sprintf(" (%d cảnh)", count)
	}
	return name
}

func budgetRange(low, high float64) string {
	if roundSeconds(low) == roundSeconds(high) {
		return fmt.Sprintf("%d giây", roundSeconds(high))
	}
	return fmt.Sprintf("%d–%d giây", roundSeconds(low), roundSeconds(high))
}

func roundSeconds(s float64) int { return int(math.Round(s)) }

func percent(f float64) int { return int(math.Round(f * 100)) }
