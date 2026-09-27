package domain

import (
	_ "embed"
	"fmt"
	"regexp"
	"strings"
)

// The channel's illustration style rules, the building blocks a drawing may
// use, and three reference drawings (CR-044). The rules are what the AI drawer
// is held to and what web-gui shows; rendering/domain/illustration_style.py
// checks the [S..] rules it can check mechanically, and the rendering test
// suite holds the reference drawings to zero findings.
//
//go:embed prompts/illustration_style_vi.txt
var illustrationStyleVI string

//go:embed prompts/illustration_helpers_vi.txt
var illustrationHelpersVI string

//go:embed prompts/illustration_exemplars_vi.txt
var illustrationExemplarsVI string

// IllustrationStyleGuide is the rule text, as shown to the Creator and the model.
func IllustrationStyleGuide() string { return strings.TrimSpace(illustrationStyleVI) }

// IllustrationHelpers documents the kit's building blocks for a new drawing.
func IllustrationHelpers() string { return strings.TrimSpace(illustrationHelpersVI) }

// CodeFinding is one problem found in a drawing's code: a check failure that
// blocks saving, or a style warning that does not (CR-044).
type CodeFinding struct {
	Message string `json:"message"`
	Line    *int   `json:"line"`
}

// CR-044: the illustration library. Every drawing the Remotion Engineer may
// place in a shot is a row here, filed in exactly one folder so the Creator and
// the matcher find it again instead of drawing it twice.
//
// Built-in rows are the CR-043 kit: their code lives in
// rendering/remotion_project/src/conceptflow-mini/illustration.tsx and ships
// with the image, so the row carries only what is needed to find and describe
// them. Library rows carry their own TSX, drawn by the model or by hand.

// IllustrationFolder is one shelf of the library ("con-nguoi", "phuong-tien"...).
type IllustrationFolder struct {
	ID          string `json:"id"` // slug, stable
	Name        string `json:"name"`
	Description string `json:"description"`
	Position    int    `json:"position"`
	IsSystem    bool   `json:"is_system"`
}

// IllustrationStatus is where a library drawing is in review.
type IllustrationStatus string

const (
	IllustrationDraft    IllustrationStatus = "draft"    // drawn, waiting for the Creator
	IllustrationApproved IllustrationStatus = "approved" // offered to the Remotion Engineer
)

// Illustration is one drawing of the library.
type Illustration struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"` // the exported component, PascalCase
	Title       string   `json:"title"`
	FolderID    string   `json:"folder_id"`
	Tags        []string `json:"tags"`
	Description string   `json:"description"` // what it looks like and when to use it
	Usage       string   `json:"usage"`       // one-line API: props and box aspect
	Code        string   `json:"code,omitempty"`
	// Builtin rows are read-only: the CR-043 kit (code ships in the image,
	// Code is empty) and the style exemplars (Exemplar, code in the row).
	Builtin    bool               `json:"builtin"`
	Exemplar   bool               `json:"exemplar"`
	Warnings   []CodeFinding      `json:"warnings"`
	Status     IllustrationStatus `json:"status"`
	Version    int                `json:"version"`
	HasPreview bool               `json:"has_preview"`
	CreatedAt  string             `json:"created_at"`
	UpdatedAt  string             `json:"updated_at"`
}

var illustrationNameRe = regexp.MustCompile(`^[A-Z][A-Za-z0-9]{1,40}$`)
var folderIDRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// ValidateIllustrationName is the same rule rendering applies to the code
// (domain/illustration_asset.py); checked here too so a bad name is refused
// before a preview round-trip.
func ValidateIllustrationName(name string) error {
	if !illustrationNameRe.MatchString(name) {
		return fmt.Errorf("tên component %q phải là PascalCase, 2–41 ký tự chữ/số (vd SchoolBus)", name)
	}
	return nil
}

// ValidateFolderID accepts a lowercase ASCII slug such as "phuong-tien".
func ValidateFolderID(id string) error {
	if !folderIDRe.MatchString(id) {
		return fmt.Errorf("mã thư mục %q phải là chữ thường không dấu, nối bằng gạch ngang (vd do-choi)", id)
	}
	return nil
}

// NormalizeTags trims, lowercases and de-duplicates tags, keeping order.
func NormalizeTags(tags []string) []string {
	out := make([]string, 0, len(tags))
	seen := map[string]bool{}
	for _, t := range tags {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" || seen[t] {
			continue
		}
		seen[t] = true
		out = append(out, t)
	}
	return out
}

// SystemIllustrationFolders are the shelves the library starts with (CR-044,
// agreed with the Creator 2026-09-27). The Creator can add more.
func SystemIllustrationFolders() []IllustrationFolder {
	rows := []IllustrationFolder{
		{ID: "con-nguoi", Name: "Con người", Description: "người, nghề nghiệp, nhóm người"},
		{ID: "dong-vat", Name: "Động vật", Description: "thú, chim, cá, côn trùng"},
		{ID: "thuc-vat", Name: "Thực vật", Description: "cây, hoa, lá, chậu cây"},
		{ID: "co-the-suc-khoe", Name: "Cơ thể & sức khoẻ", Description: "bộ phận cơ thể, vi khuẩn, thuốc, dụng cụ y tế"},
		{ID: "do-an-thuc-uong", Name: "Đồ ăn & thức uống", Description: "món ăn, bánh kẹo, đồ uống, trái cây"},
		{ID: "phuong-tien", Name: "Phương tiện", Description: "xe, tàu, máy bay, xe đạp"},
		{ID: "do-vat", Name: "Đồ vật", Description: "đồ gia dụng, nội thất, dụng cụ, thiết bị"},
		{ID: "cong-trinh-noi-chon", Name: "Công trình & nơi chốn", Description: "nhà, trường, bệnh viện, cửa hàng, căn phòng"},
		{ID: "thien-nhien-thoi-tiet", Name: "Thiên nhiên & thời tiết", Description: "mặt trời, mây, mưa, núi, sông, biển"},
		{ID: "tien-kinh-te", Name: "Tiền & kinh tế", Description: "tiền, ví, biểu đồ giá, cửa hàng, thẻ"},
		{ID: "khoa-hoc-cong-nghe", Name: "Khoa học & công nghệ", Description: "máy tính, điện thoại, ống nghiệm, nguyên tử"},
		{ID: "bieu-tuong", Name: "Biểu tượng", Description: "dấu tích/sai, tim, bóng đèn, lấp lánh, bong bóng thoại"},
		{ID: "boi-canh", Name: "Bối cảnh", Description: "nền màu, mảng màu, màn chia đôi"},
	}
	for i := range rows {
		rows[i].Position = i + 1
		rows[i].IsSystem = true
	}
	return rows
}

// BuiltinIllustrations are the 34 figures of the CR-043 kit, filed into the
// system folders. Name must match an export of illustration.tsx; the rendering
// test suite checks the kit against the prompt, and illustration_test.go
// checks this list against the same prompt text.
func BuiltinIllustrations() []Illustration {
	b := func(name, title, folder, tags, desc, usage string) Illustration {
		return Illustration{
			ID: "builtin-" + name, Name: name, Title: title, FolderID: folder,
			Tags: NormalizeTags(strings.Split(tags, ",")), Description: desc, Usage: usage,
			Builtin: true, Status: IllustrationApproved, Version: 1,
		}
	}
	return []Illustration{
		b("Backdrop", "Nền màu phẳng", "boi-canh", "nền,sàn,phòng", "Nền màu phẳng phủ cả khung, tuỳ chọn một dải sàn.", "<Backdrop color floor floorY />"),
		b("Panel", "Mảng màu", "boi-canh", "chia đôi,mảng,cửa sổ", "Mảng màu chữ nhật: nửa khung, vạch ngăn, ô nhìn sang nơi khác. x, y là góc trên-trái.", "<Panel x y w h color radius>...</Panel>"),
		b("Person", "Người", "con-nguoi", "người,trẻ em,người già,bác sĩ", "Người nhìn thẳng, 9 dáng, 7 nét mặt, 3 lứa tuổi, 3 kiểu áo, nói được.", "<Person pose mood age outfit hairStyle talking glasses shirt pants skin hair /> — 260×420"),
		b("Tooth", "Chiếc răng", "co-the-suc-khoe", "răng,sâu răng,nha khoa", "Răng hàm có mặt, sâu dần theo decay 0→1.", "<Tooth decay mood shine face /> — 200×220"),
		b("Germ", "Vi khuẩn", "co-the-suc-khoe", "vi khuẩn,vi trùng,virus,bệnh", "Vi khuẩn lúc nhúc có gai, mặt ác/vui/buồn.", "<Germ color mood variant /> — 1:1"),
		b("OpenMouth", "Miệng há", "co-the-suc-khoe", "miệng,răng,hàm", "Miệng há thấy hai hàm răng, chỉ định răng sâu.", "<OpenMouth decayed decay lips /> — 400×260"),
		b("Drop", "Giọt chất lỏng", "thien-nhien-thoi-tiet", "giọt,nước,axit,mồ hôi", "Giọt nước/axit/mồ hôi, tuỳ chọn có mặt.", "<Drop color face /> — 140×180"),
		b("Shield", "Khiên", "bieu-tuong", "bảo vệ,an toàn,miễn dịch", "Khiên bảo vệ có dấu tích.", "<Shield color check /> — 160×190"),
		b("Heart", "Trái tim", "bieu-tuong", "tim,yêu,sức khoẻ", "Trái tim, đập nhịp được.", "<Heart color beat /> — 200×180"),
		b("Candy", "Kẹo", "do-an-thuc-uong", "kẹo,đường,ngọt", "Kẹo gói xoắn hai đầu.", "<Candy color /> — 240×110"),
		b("Lollipop", "Kẹo mút", "do-an-thuc-uong", "kẹo,đường,ngọt", "Kẹo mút xoắn trên que.", "<Lollipop color /> — 140×260"),
		b("Soda", "Ly nước ngọt", "do-an-thuc-uong", "nước ngọt,đồ uống,đường", "Ly nước ngọt có nắp và ống hút.", "<Soda color /> — 160×260"),
		b("Donut", "Bánh donut", "do-an-thuc-uong", "bánh,đường,ngọt", "Bánh donut phủ đường, rắc cốm.", "<Donut color /> — 1:1"),
		b("Apple", "Quả táo", "do-an-thuc-uong", "trái cây,táo,lành mạnh", "Quả táo có lá.", "<Apple color /> — 1:1"),
		b("Clock", "Đồng hồ treo tường", "do-vat", "thời gian,giờ,chờ đợi", "Đồng hồ vuông, kim quay theo hour/minute.", "<Clock hour minute frameColor handColor /> — 1:1"),
		b("Table", "Cái bàn", "do-vat", "bàn,nội thất,phòng", "Bàn nhìn chính diện.", "<Table color /> — 400×200"),
		b("Chair", "Cái ghế", "do-vat", "ghế,nội thất,phòng", "Ghế tựa.", "<Chair color /> — 200×260"),
		b("Window", "Cửa sổ", "cong-trinh-noi-chon", "cửa sổ,ngày,đêm,phòng", "Cửa sổ có trời ngày hoặc đêm.", `<Window time="day"|"night" /> — 220×260`),
		b("Plant", "Chậu cây", "thuc-vat", "cây,chậu,phòng", "Chậu cây lá đung đưa.", "<Plant color pot /> — 160×240"),
		b("House", "Ngôi nhà", "cong-trinh-noi-chon", "nhà,gia đình", "Ngôi nhà nhỏ mái đỏ.", "<House color roof /> — 240×220"),
		b("Tree", "Cái cây", "thuc-vat", "cây,rừng,thiên nhiên", "Cây tán tròn.", "<Tree color /> — 180×260"),
		b("Sun", "Mặt trời", "thien-nhien-thoi-tiet", "mặt trời,nắng,ngày", "Mặt trời tia xoay chậm.", "<Sun color /> — 1:1"),
		b("Cloud", "Đám mây", "thien-nhien-thoi-tiet", "mây,trời,thời tiết", "Đám mây bông.", "<Cloud color /> — 260×140"),
		b("Lightbulb", "Bóng đèn", "bieu-tuong", "ý tưởng,hiểu ra,đèn", "Bóng đèn sáng/tắt — ý tưởng, hiểu ra.", "<Lightbulb on color /> — 160×240"),
		b("Coin", "Đồng xu", "tien-kinh-te", "tiền,xu,giá", "Đồng xu vàng.", "<Coin color /> — 1:1"),
		b("Book", "Quyển sách", "do-vat", "sách,học,kiến thức", "Quyển sách đóng.", "<Book color /> — 200×160"),
		b("Phone", "Điện thoại", "khoa-hoc-cong-nghe", "điện thoại,mạng xã hội,màn hình", "Điện thoại, màn hình sáng.", "<Phone color glow /> — 120×220"),
		b("Magnifier", "Kính lúp", "do-vat", "kính lúp,soi,tìm", "Kính lúp — soi kỹ, điều tra.", "<Magnifier color /> — 1:1"),
		b("Mark", "Dấu đúng/sai", "bieu-tuong", "đúng,sai,tích,x", "Huy hiệu tròn dấu tích hoặc dấu X.", `<Mark kind="check"|"cross" /> — 1:1`),
		b("Sparkle", "Tia lấp lánh", "bieu-tuong", "lấp lánh,sạch,mới", "Tia bốn cánh nhấp nháy.", "<Sparkle color /> — 1:1"),
		b("Airplane", "Máy bay", "phuong-tien", "máy bay,bay,chiến tranh", "Máy bay cánh quạt nhìn ngang.", "<Airplane color /> — 320×120"),
		b("Bubble", "Bong bóng thoại", "bieu-tuong", "nói,thoại,suy nghĩ", "Bong bóng thoại/suy nghĩ chứa nhãn ngắn.", "<Bubble x y w h thought flip>Nhãn</Bubble>"),
		b("Toothbrush", "Bàn chải đánh răng", "co-the-suc-khoe", "bàn chải,đánh răng,vệ sinh", "Bàn chải có kem, lông quay lên hoặc xuống.", "<Toothbrush color paste bristlesDown /> — 400×100"),
		b("Toothpaste", "Kem đánh răng", "co-the-suc-khoe", "kem đánh răng,vệ sinh", "Tuýp kem đánh răng.", "<Toothpaste color /> — 300×120"),
	}
}

var exemplarRe = regexp.MustCompile("(?s)=== (\\w+) — ([^=]*?) ===\n```tsx\n(.*?)```")

// ExemplarIllustrations are the reference drawings of the style guide, seeded
// as read-only library rows so the Creator sees them rendered and videos can
// use them. Their code is the text the AI drawer is shown.
func ExemplarIllustrations() []Illustration {
	meta := map[string]struct{ title, folder, tags, usage string }{
		"SchoolBus":  {"Xe buýt", "phuong-tien", "xe,xe buýt,trường học,giao thông", "<SchoolBus color mood moving /> — 320×210"},
		"Cat":        {"Con mèo", "dong-vat", "mèo,thú cưng,con vật", "<Cat color mood /> — 220×240"},
		"Microscope": {"Kính hiển vi", "khoa-hoc-cong-nghe", "kính hiển vi,khoa học,phòng thí nghiệm", "<Microscope color /> — 180×240"},
	}
	var out []Illustration
	for _, m := range exemplarRe.FindAllStringSubmatch(illustrationExemplarsVI, -1) {
		info, ok := meta[m[1]]
		if !ok {
			continue
		}
		out = append(out, Illustration{
			ID: "exemplar-" + m[1], Name: m[1], Title: info.title, FolderID: info.folder,
			Tags: NormalizeTags(strings.Split(info.tags, ",")), Description: "Hình mẫu chuẩn của luật style: " + strings.TrimSpace(m[2]) + ".",
			Usage: info.usage, Code: strings.TrimSpace(m[3]) + "\n",
			Builtin: true, Exemplar: true, Status: IllustrationApproved, Version: 1,
		})
	}
	return out
}
