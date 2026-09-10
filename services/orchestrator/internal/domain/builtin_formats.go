package domain

// Built-in formats, seeded into the database on first start (CR-019 FR51.2).
//
// These are a **starting point to edit**, not a fixed shape. The Creator clones
// and adjusts them (FR51.5); the numbers below came from reasoning about a new
// channel, not from data. CR-022 replaces them with measured retention once the
// channel has enough of it.
//
// Neither format targets the 8-minute mid-roll threshold. That mark only pays
// off after monetisation is switched on (1.000 subscriber + 4.000 giờ xem), and
// until then the thing worth optimising is retention — a loose 12-minute video
// does worse than a tight 7-minute one.

// FormatVisualFirst7Min is the channel's main shape: a worked example before
// any definition.
//
// The order is the substance here, not the labels. `concrete` is required to
// come before `pattern` because that ordering is what turns "video nhiều ví dụ
// minh hoạ trực quan thay vì text đơn điệu" from a wish in the prompt into
// something the validator can actually check.
var FormatVisualFirst7Min = VideoFormat{
	ID:         "visual_first_7min",
	Name:       "Ví dụ trước — 6 đến 8 phút",
	Version:    1,
	MinSeconds: 360,
	MaxSeconds: 480,
	Beats: []FormatBeat{
		{ID: "hook", Role: "hook", MinSeconds: 8, MaxSeconds: 12, Required: true, MaxRepeat: 1},
		{ID: "concrete", Role: "example", MinSeconds: 40, MaxSeconds: 70, Required: true, MaxRepeat: 1},
		{ID: "pattern", Role: "explain", MinSeconds: 60, MaxSeconds: 100, Required: true, MaxRepeat: 1},
		{ID: "variation", Role: "example", MinSeconds: 50, MaxSeconds: 90, Required: false, MaxRepeat: 2},
		{ID: "edge", Role: "caveat", MinSeconds: 30, MaxSeconds: 60, Required: false, MaxRepeat: 1},
		{ID: "recap", Role: "summary", MinSeconds: 20, MaxSeconds: 30, Required: true, MaxRepeat: 1},
		{ID: "cta", Role: "cta", MinSeconds: 10, MaxSeconds: 15, Required: true, MaxRepeat: 1},
	},
}

// FormatQuickExplainer3Min is for a single idea that does not need building up.
var FormatQuickExplainer3Min = VideoFormat{
	ID:         "quick_explainer_3min",
	Name:       "Giải thích nhanh — 3 đến 5 phút",
	Version:    1,
	MinSeconds: 180,
	MaxSeconds: 300,
	Beats: []FormatBeat{
		{ID: "hook", Role: "hook", MinSeconds: 6, MaxSeconds: 10, Required: true, MaxRepeat: 1},
		{ID: "concrete", Role: "example", MinSeconds: 30, MaxSeconds: 60, Required: true, MaxRepeat: 1},
		{ID: "pattern", Role: "explain", MinSeconds: 40, MaxSeconds: 80, Required: true, MaxRepeat: 1},
		{ID: "recap", Role: "summary", MinSeconds: 15, MaxSeconds: 25, Required: true, MaxRepeat: 1},
		{ID: "cta", Role: "cta", MinSeconds: 8, MaxSeconds: 12, Required: true, MaxRepeat: 1},
	},
}

func BuiltinFormats() []VideoFormat {
	return []VideoFormat{FormatVisualFirst7Min, FormatQuickExplainer3Min}
}

// DefaultVideoFormatID is what a project gets when the Creator did not choose,
// including every project created before formats existed.
const DefaultVideoFormatID = "visual_first_7min"

// BeatBudget is how long one beat actually came out, against what the format
// asked for (CR-019 FR52.4).
type BeatBudget struct {
	BeatID  string  `json:"beat_id"`
	Seconds float64 `json:"seconds"`
	Min     float64 `json:"min_seconds"`
	Max     float64 `json:"max_seconds"`
	Status  string  `json:"status"` // "ok" | "short" | "long"
}

// MeasureBeatBudgets attributes each narration's duration to the beat that was
// open at the time, then compares the totals against the format.
//
// Every result is advisory. Durations here are either estimates (before TTS) or
// real measurements (after), and the caller cannot always tell which — so this
// never produces a blocking issue. FR52.5 draws the line: block on facts
// (a missing required beat), warn on numbers.
func (f VideoFormat) MeasureBeatBudgets(observed []BeatOccurrence, sceneDurations []float64) []BeatBudget {
	if len(observed) == 0 {
		return nil
	}

	defined := make(map[string]FormatBeat, len(f.Beats))
	for _, beat := range f.Beats {
		defined[beat.ID] = beat
	}

	totals := make(map[string]float64)
	seen := make([]string, 0, len(observed))
	for i, occurrence := range observed {
		end := len(sceneDurations)
		if i+1 < len(observed) {
			end = observed[i+1].SceneIndex
		}
		total := 0.0
		for scene := occurrence.SceneIndex; scene < end && scene < len(sceneDurations); scene++ {
			if scene >= 0 {
				total += sceneDurations[scene]
			}
		}
		if _, ok := totals[occurrence.ID]; !ok {
			seen = append(seen, occurrence.ID)
		}
		totals[occurrence.ID] += total
	}

	budgets := make([]BeatBudget, 0, len(seen))
	for _, id := range seen {
		beat, known := defined[id]
		budget := BeatBudget{BeatID: id, Seconds: totals[id], Status: "ok"}
		if known {
			budget.Min, budget.Max = beat.MinSeconds, beat.MaxSeconds
			// A beat allowed to repeat gets its budget multiplied, otherwise
			// two legitimate `variation` sections always read as over budget.
			repeats := beat.MaxRepeat
			if repeats < 1 {
				repeats = 1
			}
			switch {
			case totals[id] < beat.MinSeconds:
				budget.Status = "short"
			case totals[id] > beat.MaxSeconds*float64(repeats):
				budget.Status = "long"
			}
		}
		budgets = append(budgets, budget)
	}
	return budgets
}
