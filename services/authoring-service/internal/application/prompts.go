package application

import "errors"

// ErrPromptNotFound — llm-service holds no prompt for the role asked.
var ErrPromptNotFound = errors.New("prompt not found")
