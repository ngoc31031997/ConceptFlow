package domain

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// StoryboardShot is the part of one storyboard shot the post-1b checks read
// (CR-048 T8/T9). The full shape is visual_director_ai's JSON output, parsed
// and canonicalised by llm-service (app/storyboard.py).
type StoryboardShot struct {
	ID        string `json:"id"`
	Visual    string `json:"visual"`
	Narration string `json:"narration"`
}

// StoryboardScene is one scene; its ID is the Story Architect's beat id.
type StoryboardScene struct {
	ID    string           `json:"id"`
	Shots []StoryboardShot `json:"shots"`
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
