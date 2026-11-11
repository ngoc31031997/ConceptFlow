// Package domain holds visual-service's rules: the visual-concept document of
// step 8, its parts and checks, the review findings and the inputs read from
// the locked script.
package domain

import "errors"

var (
	// ErrScriptNotLocked: step 8 reads the locked script, and there is none yet.
	ErrScriptNotLocked = errors.New("Chốt kịch bản ở bước 7 trước")
	// ErrUnknownUnit: the key names no part of the visual concept.
	ErrUnknownUnit = errors.New("không có phần này trong ngôn ngữ hình ảnh")
	// ErrUnitNotWritten: the part has no content yet.
	ErrUnitNotWritten = errors.New("phần này chưa có nội dung")
	// ErrUnitRunning: the AI is writing the part.
	ErrUnitRunning = errors.New("AI đang viết phần này")
	// ErrUnitInvalid: the part does not pass its checks and cannot be approved.
	ErrUnitInvalid = errors.New("phần này chưa đạt các điều kiện kiểm")
	// ErrInvalidEdit: an edit does not fit the part's schema.
	ErrInvalidEdit = errors.New("bản sửa không đúng khuôn của phần")
	// ErrUnknownFinding: no review finding or machine check has the id.
	ErrUnknownFinding = errors.New("không có vấn đề này")
	// ErrFindingNotApplicable: the action does not fit the finding.
	ErrFindingNotApplicable = errors.New("không áp dụng được cho vấn đề này")
	// ErrLockNotReady: the conditions to lock do not hold.
	ErrLockNotReady = errors.New("chưa đủ điều kiện chốt")
	// ErrNothingToLock: the parts are the same as in the latest locked version.
	ErrNothingToLock = errors.New("không có thay đổi so với phiên bản đã chốt")
	// ErrVersionExists: another lock took the version number first.
	ErrVersionExists = errors.New("phiên bản này đã được chốt")
	// ErrNoVersion: the version asked for is not locked.
	ErrNoVersion = errors.New("chưa có phiên bản đã chốt này")
	// ErrUnknownReference: a picked reference is not in the library.
	ErrUnknownReference = errors.New("mô tả tham chiếu không có trong thư viện")
	// ErrUnreadableReply: the AI's answer cannot be used; nothing was stored.
	ErrUnreadableReply = errors.New("câu trả lời của AI không dùng được")
	// ErrUnknownStep: no later step reads the locked visual concept by this name.
	ErrUnknownStep = errors.New("chưa có bước nào đọc ngôn ngữ hình ảnh theo tên này")
	// ErrConceptNotLocked: step 9 reads the locked visual concept, and there is none yet.
	ErrConceptNotLocked = errors.New("Chốt Ngôn ngữ hình ảnh ở bước 8 trước")
	// ErrUnknownPlanUnit: the id names no unit of the visual plan.
	ErrUnknownPlanUnit = errors.New("không có đơn vị hình này trong kế hoạch")
	// ErrUnknownQuestion: the unit has no open question with the id.
	ErrUnknownQuestion = errors.New("đơn vị hình không có câu hỏi này")
	// ErrBeatStale: the script or the visual concept changed since the beat was planned.
	ErrBeatStale = errors.New("kịch bản hoặc ngôn ngữ hình ảnh đã đổi từ lúc lập kế hoạch beat này")
	// ErrNotSplittable: the unit cannot be cut at the line given.
	ErrNotSplittable = errors.New("không tách được unit tại câu này")
	// ErrNotMergeable: the two units are not neighbours of the same beat.
	ErrNotMergeable = errors.New("chỉ gộp được hai unit liền nhau trong cùng một beat")
	// ErrUnknownPlanBeat: the key names no beat of the locked script.
	ErrUnknownPlanBeat = errors.New("kịch bản chốt không có beat này")
	// ErrNoNarrationPace: the speaking pace of the video cannot be worked out.
	ErrNoNarrationPace = errors.New("không tính được mức tối thiểu của lời thoại: creative brief chưa có thời lượng hoặc kịch bản chưa có chữ")
)
