package postgres

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"orchestrator/internal/domain"
)

// ChannelAssetPointerRepository implements domain.ChannelAssetPort against
// Orchestrator's own channel_asset_pointers projection (CR-023 correction —
// there is no HTTP server on video-assembly for Orchestrator to call; this
// table is kept current by subscribing to channel_asset_rendered /
// channel_asset_normalized events, see application/handle_step_event.go).
type ChannelAssetPointerRepository struct {
	pool *pgxpool.Pool
}

// NewChannelAssetPointerRepository constructs the repository over an
// already-open pool.
func NewChannelAssetPointerRepository(pool *pgxpool.Pool) *ChannelAssetPointerRepository {
	return &ChannelAssetPointerRepository{pool: pool}
}

// LatestChannelAsset returns the currently pointed-at asset_id for
// (kind, quality), or ("", nil) when the channel never had one rendered or
// uploaded for that combination.
func (r *ChannelAssetPointerRepository) LatestChannelAsset(ctx context.Context, kind string, quality domain.RenderQuality) (string, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT asset_id FROM channel_asset_pointers WHERE kind = $1 AND render_quality = $2`, kind, string(quality))

	var assetID string
	if err := row.Scan(&assetID); err != nil {
		if err == pgx.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return assetID, nil
}

// UpsertChannelAssetPointer records the asset newly active for (kind,
// quality). Called from the channel_asset_rendered/channel_asset_normalized
// event subscriber, never from an HTTP handler directly.
func (r *ChannelAssetPointerRepository) UpsertChannelAssetPointer(ctx context.Context, kind string, quality domain.RenderQuality, assetID string, version int) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO channel_asset_pointers (kind, render_quality, asset_id, version, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (kind, render_quality) DO UPDATE
		SET asset_id = EXCLUDED.asset_id, version = EXCLUDED.version, updated_at = now()
		WHERE channel_asset_pointers.version <= EXCLUDED.version`,
		kind, string(quality), assetID, version)
	return err
}

// ListChannelAssetPointers backs GET /v1/channel-assets/preview.
func (r *ChannelAssetPointerRepository) ListChannelAssetPointers(ctx context.Context) ([]domain.ChannelAssetPointer, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT kind, render_quality, asset_id, version FROM channel_asset_pointers ORDER BY kind, render_quality`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []domain.ChannelAssetPointer
	for rows.Next() {
		var p domain.ChannelAssetPointer
		var quality string
		if err := rows.Scan(&p.Kind, &quality, &p.AssetID, &p.Version); err != nil {
			return nil, err
		}
		p.RenderQuality = domain.RenderQuality(quality)
		out = append(out, p)
	}
	return out, rows.Err()
}
