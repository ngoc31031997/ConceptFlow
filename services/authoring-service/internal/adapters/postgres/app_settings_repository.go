package postgres

import (
	"context"
	"strings"

	"authoring/internal/domain"
)

// stepModelKeyPrefix starts the app_settings key of a step's default model;
// the step name follows it.
const stepModelKeyPrefix = "default_model."

// GetStepDefaultModels returns the default model of every configurable step:
// the one saved for it, else the first model of the catalog.
func (r *PromptTemplateRepository) GetStepDefaultModels(ctx context.Context) (map[string]string, error) {
	rows, err := r.pool.Query(ctx, `SELECT key, value FROM app_settings WHERE key LIKE $1`, stepModelKeyPrefix+"%")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	saved := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		saved[strings.TrimPrefix(key, stepModelKeyPrefix)] = value
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return domain.WithStepModelFallback(saved), nil
}

// SaveStepDefaultModel stores the default model of one step.
func (r *PromptTemplateRepository) SaveStepDefaultModel(ctx context.Context, step, model string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, stepModelKeyPrefix+step, model)
	return err
}
