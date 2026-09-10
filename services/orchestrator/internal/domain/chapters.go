package domain

import (
	"fmt"
	"strings"
)

// Chapter is one YouTube chapter, marked by a `# CHAPTER: "..."` comment in the
// Creator's script (CR-006 FR15.1).
//
// SceneIndex ties it to a narration marker rather than to a timestamp, because
// the real timestamp is only known after Rendering measures where that line
// actually falls (CR-002). Storing a time here would mean guessing.
type Chapter struct {
	SceneIndex int    `json:"scene_index"`
	Title      string `json:"title"`
}

// YouTube's own rules for chapters to appear at all: the first must start at
// 00:00, there must be at least three, and each must run for at least ten
// seconds. A list that breaks any of these is not partially honoured — YouTube
// silently shows no chapters at all, so it is better to emit none than to emit
// a list that will be thrown away.
const (
	MinChaptersForYouTube = 3
	MinChapterSeconds     = 10.0
)

// BuildChapterTimestamps turns chapter markers into the "0:00 Title" lines that
// go at the top of a YouTube description (FR15.2).
//
// offsets are the real per-narration start times from Rendering. videoSeconds
// must already include intro/outro (CR-023 D6) — the caller adds those before
// calling in, since only it knows whether they were attached at assembly.
//
// introDuration is the length of the channel intro sting prepended ahead of
// the rendered video (CR-023 D2/D6), or 0 when this project has no intro.
//   - introDuration == 0: unchanged from before CR-023 — chapter[0] is pulled
//     to 0:00 because it genuinely is the start of the video.
//   - introDuration > 0: a synthetic "Intro" chapter is inserted at 0:00, and
//     every other chapter's start is pushed out by introDuration (not forced
//     to 0 — only the true first frame is 0:00).
//
// Returns nil when the result would not satisfy YouTube's rules, so the
// caller can leave the description without a chapter block rather than
// shipping one that will be ignored.
func BuildChapterTimestamps(chapters []Chapter, offsets []float64, videoSeconds float64, introDuration float64) []string {
	if len(chapters) < MinChaptersForYouTube {
		return nil
	}

	times := make([]float64, 0, len(chapters)+1)
	titles := make([]string, 0, len(chapters)+1)
	for _, chapter := range chapters {
		if chapter.SceneIndex < 0 || chapter.SceneIndex >= len(offsets) {
			return nil
		}
		title := strings.TrimSpace(chapter.Title)
		if title == "" {
			return nil
		}
		start := offsets[chapter.SceneIndex]
		if introDuration > 0 {
			start += introDuration
		}
		times = append(times, start)
		titles = append(titles, title)
	}

	if introDuration > 0 {
		// A synthetic chapter for the channel intro itself — it is real screen
		// time the viewer sees, so it earns its own entry rather than being
		// folded silently into whatever the Creator's first marker was.
		times = append([]float64{0}, times...)
		titles = append([]string{"Intro"}, titles...)
	} else {
		// The first chapter must open the video. A Creator who marked their
		// first chapter a little way in still gets chapters — the opening is
		// simply pulled back to zero, which is what they meant.
		times[0] = 0
	}

	for i := range times {
		if i > 0 && times[i] <= times[i-1] {
			// Out of order or duplicated: the markers do not describe a
			// timeline, so there is nothing sensible to publish.
			return nil
		}
		end := videoSeconds
		if i+1 < len(times) {
			end = times[i+1]
		}
		if end-times[i] < MinChapterSeconds {
			return nil
		}
	}

	lines := make([]string, 0, len(times))
	for i, t := range times {
		lines = append(lines, fmt.Sprintf("%s %s", FormatTimestamp(t), titles[i]))
	}
	return lines
}

// FormatTimestamp renders seconds the way YouTube parses them in a description:
// M:SS below an hour, H:MM:SS at or above it.
func FormatTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	total := int(seconds)
	hours := total / 3600
	minutes := (total % 3600) / 60
	secs := total % 60
	if hours > 0 {
		return fmt.Sprintf("%d:%02d:%02d", hours, minutes, secs)
	}
	return fmt.Sprintf("%d:%02d", minutes, secs)
}

// ComposeDescription assembles the four-part description of FR18.2: the
// model's summary, the chapter list, a call to action, and hashtags.
//
// Each part is omitted when empty rather than leaving a blank heading, so a
// project with no chapters still gets a clean description.
func ComposeDescription(summary string, chapterLines []string, callToAction string, tags []string) string {
	var sections []string

	if s := strings.TrimSpace(summary); s != "" {
		sections = append(sections, s)
	}
	if len(chapterLines) > 0 {
		sections = append(sections, strings.Join(chapterLines, "\n"))
	}
	if cta := strings.TrimSpace(callToAction); cta != "" {
		sections = append(sections, cta)
	}
	if hashtags := buildHashtags(tags); hashtags != "" {
		sections = append(sections, hashtags)
	}

	return strings.Join(sections, "\n\n")
}

// buildHashtags renders up to three tags as hashtags. Three because YouTube
// only surfaces the first three above the title, and more in the description
// reads as keyword stuffing.
func buildHashtags(tags []string) string {
	var out []string
	for _, tag := range tags {
		cleaned := strings.ReplaceAll(strings.TrimSpace(tag), " ", "")
		cleaned = strings.TrimPrefix(cleaned, "#")
		if cleaned == "" {
			continue
		}
		out = append(out, "#"+cleaned)
		if len(out) == 3 {
			break
		}
	}
	return strings.Join(out, " ")
}
