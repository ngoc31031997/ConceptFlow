import { describe, it, expect, vi, beforeEach, afterEach, type MockInstance } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { useContext, useEffect, type ReactNode } from "react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { CreativeBriefStepPage } from "../../src/pages/CreativeBriefStepPage";
import { ProjectDraftDispatchContext, ProjectDraftProvider } from "../../src/context/ProjectDraftContext";
import { ProjectFlowProvider } from "../../src/context/ProjectFlowContext";
import { ThemeProvider } from "../../src/context/ThemeContext";
import * as apiClient from "../../src/api/client";
import type { BriefState, CreativeBrief, Idea, IdeaState, ReviewUnit, UnitStatus } from "../../src/api/client";

const BRIEF: CreativeBrief = {
  audience: "Người xem 16–35 tuổi",
  know: "Quả sung là chùm hoa kín",
  understand: "Vòng đời ong gắn với sung",
  feel: "Ngạc nhiên",
  tone: "Tò mò, ấm",
  out_of_scope: "Không tư vấn ăn uống",
};
const IDEA: Idea = { question: "Ong vào quả sung để làm gì?", why: "w", known: "k", unknown: "u" };

function briefUnit(status: UnitStatus, content: CreativeBrief = BRIEF, stale = false): ReviewUnit<CreativeBrief> {
  return { projectId: "p", kind: "brief", key: "main", status, content, creatorFields: [], redoCount: 0, updatedAt: "", stale };
}
function ideaUnit(status: UnitStatus): ReviewUnit<Idea> {
  return { projectId: "p", kind: "idea", key: "main", status, content: IDEA, creatorFields: [], redoCount: 0, updatedAt: "", stale: false };
}

let briefState: BriefState;
let ideaState: IdeaState;
let patchSettings: MockInstance<typeof apiClient.patchWizardSettings>;
let setDuration: MockInstance<typeof apiClient.setTargetDuration>;
let setPlatformCall: MockInstance<typeof apiClient.setPlatform>;
let generate: MockInstance<typeof apiClient.generateBrief>;

beforeEach(() => {
  briefState = { unit: null, targetDuration: "4-5", platform: "YouTube", storyStale: false, storyPromptReadsBrief: true };
  ideaState = { unit: ideaUnit("approved") };
  vi.spyOn(apiClient, "getLlmStatus").mockResolvedValue({
    enabled: true,
    provider: "hive",
    reason: "",
    models: [{ id: "ollama", label: "Ollama (local AI)" }],
  });
  vi.spyOn(apiClient, "fetchVideoFormats").mockResolvedValue([]);
  vi.spyOn(apiClient, "getBriefState").mockImplementation(async () => briefState);
  vi.spyOn(apiClient, "getIdeaState").mockImplementation(async () => ideaState);
  vi.spyOn(apiClient, "getUnitImpact").mockResolvedValue([]);
  patchSettings = vi.spyOn(apiClient, "patchWizardSettings").mockResolvedValue(undefined);
  setDuration = vi.spyOn(apiClient, "setTargetDuration").mockResolvedValue(undefined);
  setPlatformCall = vi.spyOn(apiClient, "setPlatform").mockResolvedValue(undefined);
  generate = vi.spyOn(apiClient, "generateBrief").mockImplementation(async () => {
    briefState = { ...briefState, unit: briefUnit("pending") };
    return briefUnit("pending");
  });
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
  it("hiện hình thức video và thẻ brief, không còn các mục cấu hình giọng, font, nhạc, chất lượng", async () => {
    renderPage();
    expect(await screen.findByTestId("brief-form-card")).toBeInTheDocument();
    expect(screen.getByRole("heading", { level: 1 })).toHaveTextContent("Định hướng sáng tạo");
    expect(screen.getByTestId("brief-card")).toBeInTheDocument();
    for (const gone of ["settings-section-voice", "settings-section-font", "settings-section-music", "settings-section-quality", "settings-section-gates"]) {
      expect(screen.queryByTestId(gone)).not.toBeInTheDocument();
    }
  });

  it("chưa có brief mà ý tưởng chưa duyệt thì không cho AI viết và nói phải duyệt ý tưởng trước", async () => {
    ideaState = { unit: ideaUnit("pending") };
    renderPage();
    const button = await screen.findByTestId("brief-generate");
    expect(button).toBeDisabled();
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Duyệt ý tưởng ở bước 1 trước");
  });

  it("AI viết brief từ ý tưởng đã duyệt, brief chờ duyệt", async () => {
    renderPage();
    fireEvent.click(await screen.findByTestId("brief-generate"));
    await screen.findByTestId("brief-field-audience");
    expect(generate).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("brief-field-audience")).toHaveTextContent("Người xem 16–35 tuổi");
    expect(screen.getByTestId("brief-status")).toHaveTextContent("Chờ duyệt");
    expect(screen.getByTestId("creative-brief-next")).toBeDisabled();
  });

  it("đổi dạng video sang short thì lưu lên project, và đổi ngôn ngữ, thời lượng thì lưu từng cái", async () => {
    renderPage();
    await screen.findByTestId("brief-generate");
    await waitFor(() => expect(screen.getByTestId("video-output-mode-short")).not.toBeDisabled());

    fireEvent.click(screen.getByTestId("content-language-en"));
    await waitFor(() => expect(patchSettings).toHaveBeenCalledWith("p", { voiceLanguage: "en" }));

    fireEvent.change(screen.getByTestId("brief-duration"), { target: { value: "5-7" } });
    await waitFor(() => expect(setDuration).toHaveBeenCalledWith("p", "5-7"));

    fireEvent.click(screen.getByTestId("video-output-mode-short"));
    await waitFor(() => expect(patchSettings).toHaveBeenCalledWith("p", { videoOutputMode: "short" }));
    expect(screen.getByTestId("brief-duration")).toHaveTextContent("dưới 60 giây");
  });

  it("brief đã có vẫn đổi được hình thức, kèm lời nhắc làm lại brief để áp dụng", async () => {
    briefState = { ...briefState, unit: briefUnit("pending") };
    renderPage();
    await screen.findByTestId("brief-field-audience");
    expect(await screen.findByTestId("brief-form-note")).toHaveTextContent("Làm lại kèm ghi chú");
    await waitFor(() => expect(screen.getByTestId("video-output-mode-short")).not.toBeDisabled());
    expect(screen.getByTestId("content-language-en")).not.toBeDisabled();
    expect(screen.getByTestId("brief-duration")).not.toBeDisabled();
    expect(screen.getByTestId("brief-platform")).not.toBeDisabled();
  });

  it("báo lỗi khi không lưu được lựa chọn", async () => {
    patchSettings.mockRejectedValue(new apiClient.ApiError("không lưu được", undefined, undefined, 409));
    renderPage();
    await screen.findByTestId("brief-generate");
    await waitFor(() => expect(screen.getByTestId("content-language-en")).not.toBeDisabled());
    fireEvent.click(screen.getByTestId("content-language-en"));
    expect(await screen.findByTestId("brief-save-error")).toHaveTextContent("không lưu được");
  });

  it("brief viết từ ý tưởng đã đổi thì báo cần xem lại", async () => {
    briefState = { ...briefState, unit: briefUnit("approved", BRIEF, true) };
    renderPage();
    expect(await screen.findByTestId("brief-stale")).toHaveTextContent("Ý tưởng đã đổi");
  });

  it("nền tảng phát hành là một lựa chọn cùng hàng với ngôn ngữ và thời lượng, lưu ngay khi đổi", async () => {
    renderPage();
    const platform = (await screen.findByTestId("brief-platform")) as HTMLSelectElement;
    expect(platform.value).toBe("YouTube");
    expect(Array.from(platform.options).map((o) => o.value)).toEqual([...apiClient.PLATFORMS]);
    await waitFor(() => expect(platform).not.toBeDisabled());

    fireEvent.change(platform, { target: { value: "YouTube Shorts" } });
    await waitFor(() => expect(setPlatformCall).toHaveBeenCalledWith("p", "YouTube Shorts"));
  });

  it("nền tảng không còn là một ô của brief", async () => {
    briefState = { ...briefState, unit: briefUnit("pending") };
    renderPage();
    await screen.findByTestId("brief-field-audience");
    expect(screen.queryByTestId("brief-field-platform")).not.toBeInTheDocument();
  });

  it("duyệt brief thì mở Lưu và sang nghiên cứu, bấm thì chốt bước rồi sang Nghiên cứu", async () => {
    briefState = { ...briefState, unit: briefUnit("approved") };
    renderPage();
    await waitFor(() => expect(screen.getByTestId("creative-brief-next")).not.toBeDisabled());
    expect(screen.getByTestId("step-strip")).toHaveTextContent("Brief đã duyệt");

    fireEvent.click(screen.getByTestId("creative-brief-next"));
    expect(await screen.findByTestId("landed-on-research")).toBeInTheDocument();
    expect(patchSettings).toHaveBeenCalledWith("p", { confirm: true });
  });

  it("xem trước: mọi điều khiển bị khoá và không có nút đi tiếp", async () => {
    renderPage(<CreativeBriefStepPage readOnly onBack={() => undefined} />);
    expect(await screen.findByTestId("brief-read-only-notice")).toBeInTheDocument();
    expect(screen.getByTestId("brief-form-card").closest("[inert]")).not.toBeNull();
    expect(screen.queryByTestId("creative-brief-next")).not.toBeInTheDocument();
    expect(screen.getByTestId("brief-read-only-back")).toBeInTheDocument();
  });
});
