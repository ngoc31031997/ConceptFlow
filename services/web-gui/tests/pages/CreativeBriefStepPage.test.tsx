import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext, useEffect, type ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { CreativeBriefStepPage } from "../../src/pages/CreativeBriefStepPage";
import { ProjectDraftDispatchContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { BriefReview, BriefScreen, ProjectBrief } from "../../src/api/client";

const BRIEF: ProjectBrief = {
  audience: "Khán giả phổ thông từ cấp 2 trở lên.",
  durationSeconds: 270,
  tone: "Tò mò, dễ hiểu, chính xác.",
  learningObjectives: ["Giải thích chân trời sự kiện ở mức khái niệm.", "Hiểu lực thủy triều phụ thuộc vào điều kiện vật lý."],
  avoid: ["Mô tả chân trời sự kiện như một lớp vỏ cứng."],
  creatorFields: [],
  status: "pending",
  redoCount: 0,
  approvedAt: null,
  updatedAt: "",
};
const REVIEW: BriefReview = {
  audienceFit: { rating: "good", comment: "Khán giả rõ." },
  durationFit: { rating: "fair", comment: "Hơi dài." },
  objectives: { rating: "good", comment: "Cụ thể." },
  avoidList: { rating: "weak", comment: "Thiếu cảnh báo." },
  unclear: ["Người xem đã biết hố đen chưa?"],
  suggestions: ["Thêm một dòng về kích thước hố đen."],
  createdAt: "",
};

function screenOf(overrides: Partial<BriefScreen> = {}): BriefScreen {
  return {
    brief: null,
    idea: { centralQuestion: "Người rơi thấy gì?", curiosityPoints: ["Chân trời sự kiện là gì?"], scope: "Trực quan, không toán." },
    subject: "Chuyện gì xảy ra nếu ta rơi vào hố đen?",
    language: "vi",
    videoMode: "long",
    minDurationSeconds: 121,
    maxDurationSeconds: 1800,
    ideaChanged: false,
    review: null,
    reviewStale: false,
    ...overrides,
  };
}

let briefScreen: BriefScreen;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;
let generate: MockInstance<typeof apiClient.generateProjectBrief>;
let review: MockInstance<typeof apiClient.reviewProjectBrief>;
let approve: MockInstance<typeof apiClient.approveProjectBrief>;
let edit: MockInstance<typeof apiClient.editProjectBrief>;

beforeEach(() => {
  briefScreen = screenOf();
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    reason: "",
    models: [{ id: "ollama", label: "Ollama (local AI)" }],
  });
  vi.spyOn(apiClient, "fetchVideoFormats").mockResolvedValue([]);
  vi.spyOn(apiClient, "getBriefScreen").mockImplementation(async () => briefScreen);
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
  generate = vi.spyOn(apiClient, "generateProjectBrief").mockImplementation(async () => {
    briefScreen = { ...briefScreen, brief: BRIEF };
    return BRIEF;
  });
  review = vi.spyOn(apiClient, "reviewProjectBrief").mockImplementation(async () => {
    briefScreen = { ...briefScreen, review: REVIEW };
    return REVIEW;
  });
  approve = vi.spyOn(apiClient, "approveProjectBrief").mockImplementation(async () => {
    briefScreen = { ...briefScreen, brief: { ...(briefScreen.brief ?? BRIEF), status: "approved" } };
    return briefScreen.brief as ProjectBrief;
  });
  edit = vi.spyOn(apiClient, "editProjectBrief").mockImplementation(async () => briefScreen.brief ?? BRIEF);
});

afterEach(() => {
  vi.restoreAllMocks();
  window.localStorage.clear();
});

function WithProject({ children }: { children: ReactNode }) {
  const dispatch = useContext(ProjectDraftDispatchContext);
  useEffect(() => {
    dispatch({ type: "LOAD_PROJECT", payload: { projectId: "p", authoringTopic: "Vì sao sung có xác ong?" } });
  }, [dispatch]);
  return <>{children}</>;
}

function renderPage(page = <CreativeBriefStepPage />) {
  render(
    <ThemeProvider>
      <MemoryRouter initialEntries={["/create/script/brief"]}>
        <ProjectDraftProvider>
          <WithProject>
            <ProjectFlowProvider>
              <Routes>
                <Route path="/create/script/brief" element={page} />
                <Route path="/projects/:id/research" element={<div data-testid="landed-on-research" />} />
                <Route path="/" element={<div data-testid="landed-on-idea" />} />
              </Routes>
            </ProjectFlowProvider>
          </WithProject>
        </ProjectDraftProvider>
      </MemoryRouter>
    </ThemeProvider>,
  );
}

describe("CreativeBriefStepPage (Bước 2 — Creative brief)", () => {
  it("hiện dạng video, ý tưởng đã duyệt và thẻ brief, không còn ô thời lượng, nền tảng và ngôn ngữ", async () => {
    renderPage();
    expect(await screen.findByTestId("brief-form-card")).toBeInTheDocument();
    expect(screen.queryByTestId("content-language-en")).not.toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Định hướng sáng tạo");
    expect(screen.queryByTestId("brief-idea-card")).not.toBeInTheDocument();
    fireEvent.click(await screen.findByTestId("brief-input-open"));
    expect(await screen.findByTestId("brief-idea-question")).toHaveTextContent("Người rơi thấy gì?");
    expect(screen.getByTestId("brief-card")).toBeInTheDocument();
    expect(screen.queryByTestId("brief-duration")).not.toBeInTheDocument();
    expect(screen.queryByTestId("brief-platform")).not.toBeInTheDocument();
  });

  it("ý tưởng chưa duyệt thì không cho AI viết và nói phải duyệt ý tưởng trước", async () => {
    briefScreen = screenOf({ idea: null });
    renderPage();
    expect(await screen.findByTestId("brief-generate")).toBeDisabled();
    fireEvent.click(screen.getByTestId("brief-input-open"));
    expect(await screen.findByTestId("brief-idea-missing")).toHaveTextContent("chưa được duyệt");
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Duyệt ý tưởng ở bước 1 trước");
  });

  it("AI viết brief; brief chờ duyệt, có đủ năm ô, đánh giá chỉ chạy khi Creator nhờ", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-generate"));
    await screen.findByTestId("brief-field-audience");
    expect(generate).toHaveBeenCalledTimes(1);
    expect(review).not.toHaveBeenCalled();
    expect(screen.getByTestId("brief-review-none")).toBeInTheDocument();
    fireEvent.click(screen.getByTestId("brief-review-run"));
    await waitFor(() => expect(review).toHaveBeenCalledTimes(1));
    expect(screen.getByTestId("brief-field-audience")).toHaveTextContent("Khán giả phổ thông");
    expect(screen.getByTestId("brief-field-duration_seconds")).toHaveTextContent("4:30");
    expect(screen.getByTestId("brief-field-duration_seconds")).toHaveTextContent("270 giây");
    expect(screen.getByTestId("brief-field-tone")).toHaveTextContent("Tò mò");
    expect(screen.getByTestId("brief-field-learning_objectives")).toHaveTextContent("Hiểu lực thủy triều");
    expect(screen.getByTestId("brief-field-avoid")).toHaveTextContent("lớp vỏ cứng");
    expect(screen.getByTestId("brief-status")).toHaveTextContent("Chờ duyệt");
    expect(await screen.findByTestId("brief-review-panel")).toHaveTextContent("Thiếu cảnh báo.");
    expect(screen.getByTestId("brief-review-suggestions")).toHaveTextContent("kích thước hố đen");
    expect(screen.getByTestId("creative-brief-next")).toBeDisabled();
  });

  it("ô Creator đã viết hiện nhãn Bạn viết", async () => {
    briefScreen = screenOf({ brief: { ...BRIEF, creatorFields: ["tone"] } });
    renderPage();
    expect(await screen.findByTestId("brief-mine-tone")).toBeInTheDocument();
    expect(screen.queryByTestId("brief-mine-audience")).not.toBeInTheDocument();
  });

  it("duyệt brief thì mở Lưu và sang nghiên cứu, bấm thì chốt bước rồi sang Nghiên cứu", async () => {
    briefScreen = screenOf({ brief: BRIEF });
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-actions-approve"));
    await waitFor(() => expect(approve).toHaveBeenCalledTimes(1));
    await waitFor(() => expect(screen.getByTestId("creative-brief-next")).not.toBeDisabled());
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Brief đã duyệt");

    fireEvent.click(screen.getByTestId("creative-brief-next"));
    expect(await screen.findByTestId("landed-on-research")).toBeInTheDocument();
    expect(patchSettings).toHaveBeenCalledWith("p", { confirm: true });
  });

  it("brief còn ô trống thì không duyệt được và nói ô nào còn trống", async () => {
    briefScreen = screenOf({ brief: { ...BRIEF, avoid: [] } });
    renderPage();
    expect(await screen.findByTestId("brief-actions-approve")).toBeDisabled();
    expect(screen.getByTestId("brief-missing")).toHaveTextContent("Cần tránh");
  });

  it("thời lượng ngoài khoảng của dạng video thì không duyệt được", async () => {
    briefScreen = screenOf({ brief: { ...BRIEF, durationSeconds: 90 } });
    renderPage();
    expect(await screen.findByTestId("brief-actions-approve")).toBeDisabled();
    expect(screen.getByTestId("brief-duration-range")).toHaveTextContent("từ 2:01 đến 30:00");
    expect(approve).not.toHaveBeenCalled();
  });

  it("sửa tay: gõ thời lượng dạng phút:giây và lưu thành số giây, chỉ gửi ô đã đổi", async () => {
    briefScreen = screenOf({ brief: BRIEF });
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-actions-edit"));
    fireEvent.change(await screen.findByTestId("brief-edit-field-duration_seconds"), { target: { value: "5:15" } });
    fireEvent.change(screen.getByTestId("brief-edit-field-avoid"), { target: { value: "Điều A\nĐiều B" } });
    fireEvent.click(screen.getByTestId("brief-edit-save"));
    await waitFor(() => expect(edit).toHaveBeenCalledWith("p", { durationSeconds: 315, avoid: ["Điều A", "Điều B"] }));
  });

  it("sửa tay: thời lượng gõ sai thì báo lỗi và không gọi máy chủ", async () => {
    briefScreen = screenOf({ brief: BRIEF });
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-actions-edit"));
    fireEvent.change(await screen.findByTestId("brief-edit-field-duration_seconds"), { target: { value: "bốn phút" } });
    fireEvent.click(screen.getByTestId("brief-edit-save"));
    expect(await screen.findByRole("alert")).toHaveTextContent("phút:giây");
    expect(edit).not.toHaveBeenCalled();
  });

  it("làm lại kèm ghi chú gửi ghi chú cho AI", async () => {
    briefScreen = screenOf({ brief: BRIEF });
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-actions-redo"));
    fireEvent.change(await screen.findByTestId("brief-redo-note"), { target: { value: "ngắn hơn" } });
    fireEvent.click(screen.getByTestId("brief-redo-send"));
    await waitFor(() => expect(generate).toHaveBeenCalledWith("p", "ngắn hơn", expect.anything()));
  });

  it("ý tưởng đã đổi sau khi viết brief thì báo và để Creator quyết", async () => {
    briefScreen = screenOf({ brief: BRIEF, ideaChanged: true });
    renderPage();
    expect(await screen.findByTestId("brief-idea-changed")).toHaveTextContent("đã đổi");
    expect(generate).not.toHaveBeenCalled();
  });

  it("đánh giá cũ thì báo cần đánh giá lại", async () => {
    briefScreen = screenOf({ brief: BRIEF, review: REVIEW, reviewStale: true });
    renderPage();
    expect(await screen.findByTestId("brief-review-stale")).toHaveTextContent("Đánh giá lại");
  });

  it("đổi dạng video thì lưu lên project rồi tải lại khoảng thời lượng", async () => {
    renderPage();
    await screen.findByTestId("brief-generate");
    await waitFor(() => expect(screen.getByTestId("video-output-mode-short")).not.toBeDisabled());
    const loadsBefore = vi.mocked(apiClient.getBriefScreen).mock.calls.length;
    fireEvent.click(screen.getByTestId("video-output-mode-short"));
    await waitFor(() => expect(patchSettings).toHaveBeenCalledWith("p", { videoOutputMode: "short" }));
    await waitFor(() => expect(vi.mocked(apiClient.getBriefScreen).mock.calls.length).toBeGreaterThan(loadsBefore));
  });

  it("báo lỗi khi không lưu được lựa chọn", async () => {
    patchSettings.mockRejectedValue(new apiClient.ApiError("không lưu được", undefined, undefined, 409));
    renderPage();
    await screen.findByTestId("brief-generate");
    await waitFor(() => expect(screen.getByTestId("video-output-mode-short")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("video-output-mode-short"));
    expect(await screen.findByTestId("brief-save-error")).toHaveTextContent("không lưu được");
  });

  it("xem trước: mọi điều khiển bị khoá và không có nút đi tiếp", async () => {
    renderPage(<CreativeBriefStepPage readOnly onBack={() => undefined} />);
    expect(await screen.findByTestId("brief-read-only-notice")).toBeInTheDocument();
    expect(screen.getByTestId("brief-form-card").closest("[inert]")).not.toBeNull();
    expect(screen.queryByTestId("creative-brief-next")).not.toBeInTheDocument();
    expect(screen.getByTestId("brief-read-only-back")).toBeInTheDocument();
  });
});
