package domain

import "errors"

// Errors of step 11.
var (
	// ErrSequenceNotLocked: step 11 reads the locked visual sequence, and there is none yet.
	ErrSequenceNotLocked = errors.New("Chốt Chuỗi hình ở bước 10 trước")
	// ErrSequenceOutdated: the locked visual sequence was written from an older
	// script, visual concept or visual plan than the latest locked ones.
	ErrSequenceOutdated = errors.New("Chuỗi hình dựa trên bản cũ của bước trước, chốt lại bước 10")
	// ErrNoKit: the video picked no reference style at step 8, or its style has
	// no drawing kit, so nothing can be drawn.
	ErrNoKit = errors.New("dự án chưa chọn phong cách tham chiếu có bộ thông số vẽ ở bước 8")
	// ErrAssetsMissing: the shots of a unit are written only over approved assets.
	ErrAssetsMissing = errors.New("tài sản của video chưa đủ hoặc chưa duyệt hết")
	// ErrAssetNoDetail: an asset is drawn only once it has its detail (a portrayal).
	ErrAssetNoDetail = errors.New("tài sản chưa có chi tiết, viết chi tiết trước khi vẽ")
	// ErrUnknownAsset: the video has no asset with the id.
	ErrUnknownAsset = errors.New("không có tài sản này")
	// ErrUnknownShot: no shot of the unit has the id.
	ErrUnknownShot = errors.New("không có shot này")
	// ErrAssetInUse: shots use the asset, so it cannot be deleted.
	ErrAssetInUse = errors.New("tài sản đang được shot dùng")
	// ErrLastShot: a unit keeps at least one shot.
	ErrLastShot = errors.New("mỗi unit phải còn ít nhất một shot")
)
