package application

import (
	"context"
	"time"

	"orchestrator/internal/domain"
)

// NormalizeChannelAssetInput is the shape POST /v1/channel-assets/{kind}
// hands to ChannelAssetsUseCase.Normalize — a Creator-uploaded file api-
// gateway already wrote to the shared volume.
type NormalizeChannelAssetInput struct {
	Kind          string // "intro" or "outro"
	FilePath      string // path on the shared volume api-gateway just wrote
	SourceHash    string // content hash api-gateway computed, for video-assembly's cache
	RenderQuality domain.RenderQuality
	// AssetRole says what kind of file FilePath is: AssetRoleVideo (the
	// sting clip itself) or AssetRoleMusic (its audio bed). Empty
	// means video.
	AssetRole string
}

// Asset roles carried on normalize_channel_asset.
const (
	AssetRoleVideo = "video"
	AssetRoleMusic = "music"
)

// ChannelAssetsUseCase handles the two synchronous-looking but AMQP-backed
// channel asset operations: triggering a normalize run
// for an uploaded file, and reading back the current projection for preview.
// There is no HTTP call to video-assembly anywhere in here — Normalize only
// publishes a command, and Preview only reads Orchestrator's own
// channel_asset_pointers table.
type ChannelAssetsUseCase struct {
	commands domain.CommandPublisherPort
	pointers domain.ChannelAssetPort
}

// NewChannelAssetsUseCase constructs the use case.
func NewChannelAssetsUseCase(commands domain.CommandPublisherPort, pointers domain.ChannelAssetPort) *ChannelAssetsUseCase {
	return &ChannelAssetsUseCase{commands: commands, pointers: pointers}
}

// Normalize publishes the normalize_channel_asset command to video-assembly
// (routing key "video_assembly", same queue assemble_video commands use —
// video-assembly's consumer.go dispatches on payload event_type). This is not
// part of any render/publish saga — there is no saga_id to thread through, so
// a fresh message_id stands alone, mirroring how rendering's admin-triggered
// render_channel_asset command has no real project behind it
// either.
func (uc *ChannelAssetsUseCase) Normalize(ctx context.Context, in NormalizeChannelAssetInput) error {
	role := in.AssetRole
	if role == "" {
		role = AssetRoleVideo
	}
	envelope := domain.Envelope{
		MessageID: newUUID(),
		SagaID:    newUUID(),
		ProjectID: "channel-asset-upload",
		EventType: "normalize_channel_asset",
		Payload: map[string]interface{}{
			"event_type":     "normalize_channel_asset",
			"kind":           in.Kind,
			"file_path":      in.FilePath,
			"source_hash":    in.SourceHash,
			"render_quality": string(in.RenderQuality),
			"asset_role":     role,
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}
	return uc.commands.PublishCommand(ctx, "video_assembly", envelope)
}

// Preview returns every (kind, render_quality) pointer Orchestrator currently
// knows about (GET /v1/channel-assets/preview).
func (uc *ChannelAssetsUseCase) Preview(ctx context.Context) ([]domain.ChannelAssetPointer, error) {
	return uc.pointers.ListChannelAssetPointers(ctx)
}
