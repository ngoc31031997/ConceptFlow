package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
)

const defaultModelKey = "default_model"

// GetDefaultModel returns the model every AI call runs on when nothing more
// specific was chosen, or "" when none was set (the server's own default then
// applies).
func (r *PromptTemplateRepository) GetDefaultModel(ctx context.Context) (string, error) {
	var value string
	err := r.pool.QueryRow(ctx, `SELECT value FROM app_settings WHERE key = $1`, defaultModelKey).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return value, err
}

// SaveDefaultModel stores the default model; "" clears it.
func (r *PromptTemplateRepository) SaveDefaultModel(ctx context.Context, model string) error {
	if model == "" {
		_, err := r.pool.Exec(ctx, `DELETE FROM app_settings WHERE key = $1`, defaultModelKey)
		return err
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO app_settings (key, value) VALUES ($1, $2)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`, defaultModelKey, model)
	return err
}
