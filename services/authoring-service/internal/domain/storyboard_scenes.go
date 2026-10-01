package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// StoryboardShot is the part of one storyboard shot the post-1b checks read.
// The full shape is visual_director_ai's JSON output, parsed
// and canonicalised by llm-service (app/storyboard.py).
type StoryboardShot struct {
	ID        string `json:"id"`
	Visual    string `json:"visual"`
	Narration string `json:"narration"`
}

// StoryboardScene is one scene; its ID is the Story Architect's beat id.
type StoryboardScene struct {
	ID string `json:"id"`
	// Setting is the place the scene happens in; empty in a storyboard
	// written before the director was asked for it.
	Setting string           `json:"setting"`
	Shots   []StoryboardShot `json:"shots"`
}

// CheckSceneSettings warns about every scene the director gave no setting:
// its shots are drawn on a flat colour instead of in a place.
func CheckSceneSettings(scenes []StoryboardScene) []string {
	var warnings []string
	for _, sc := range scenes {
		if strings.TrimSpace(sc.Setting) == "" {
			warnings = append(warnings, fmt.Sprintf(
				"Cảnh %s chưa có bối cảnh (setting): các shot của cảnh sẽ dựng trên nền màu thay vì một nơi chốn kín khung.", sc.ID))
		}
	}
	return warnings
}

// ParseStoryboardScenes reads the scenes out of a canonical storyboard.
//
// Only the fields the checks need are decoded; anything else in the document
// is ignored. An unreadable document is an error for the caller to report, not
// an empty list — "no scenes" would read as "nothing to warn about".
func ParseStoryboardScenes(content string) ([]StoryboardScene, error) {
	var doc struct {
		Scenes []StoryboardScene `json:"scenes"`
	}
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("storyboard trống")
	}
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		return nil, fmt.Errorf("storyboard không phải JSON hợp lệ: %w", err)
	}
	if len(doc.Scenes) == 0 {
		return nil, errors.New("storyboard không có cảnh nào")
	}
	return doc.Scenes, nil
}
