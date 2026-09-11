import { describe, it, expect, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { useContext } from "react";
import {
  ProjectDraftProvider,
  ProjectDraftContext,
  ProjectDraftDispatchContext,
} from "../../src/context/ProjectDraftContext";

function Consumer() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  return (
    <div>
      <span data-testid="script">{draft.scriptContent}</span>
      <button onClick={() => dispatch({ type: "SET_SCRIPT", payload: "hello" })}>set</button>
    </div>
  );
}

describe("ProjectDraftContext", () => {
  it("updates state via dispatch", () => {
    render(
      <ProjectDraftProvider>
        <Consumer />
      </ProjectDraftProvider>,
    );

    expect(screen.getByTestId("script").textContent).toBe("");
    fireEvent.click(screen.getByText("set"));
    expect(screen.getByTestId("script").textContent).toBe("hello");
  });
});

function ResetConsumer() {
  const draft = useContext(ProjectDraftContext);
  const dispatch = useContext(ProjectDraftDispatchContext);
  return (
    <div>
      <span data-testid="project-id">{draft.projectId}</span>
      <span data-testid="submitted">{String(draft.hasSubmitted)}</span>
      <span data-testid="script">{draft.scriptContent}</span>
      <span data-testid="voice-id">{draft.voiceId ?? ""}</span>
      <button onClick={() => dispatch({ type: "SET_SCRIPT", payload: "keep me" })}>write</button>
      <button onClick={() => dispatch({ type: "SET_VOICE_ID", payload: "voice-b" })}>pick-voice</button>
      <button onClick={() => dispatch({ type: "MARK_SUBMITTED" })}>submit</button>
      <button onClick={() => dispatch({ type: "RESUME_EDITING" })}>resume</button>
      <button onClick={() => dispatch({ type: "RESET" })}>reset</button>
    </div>
  );
}

describe("draft lifecycle", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it("gives a reset draft a new project id", () => {
    // Reusing the id would point a second render saga at the project the
    // previous draft already produced.
    render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    const before = screen.getByTestId("project-id").textContent;

    fireEvent.click(screen.getByText("reset"));

    expect(screen.getByTestId("project-id").textContent).not.toBe(before);
    expect(screen.getByTestId("submitted").textContent).toBe("false");
  });

  it("restores an unsubmitted draft after a reload", () => {
    const first = render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    fireEvent.click(screen.getByText("write"));
    first.unmount();

    render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    expect(screen.getByTestId("script").textContent).toBe("keep me");
  });

  it("does not restore a draft that already started a render", () => {
    const first = render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    fireEvent.click(screen.getByText("write"));
    fireEvent.click(screen.getByText("submit"));
    first.unmount();

    render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    expect(screen.getByTestId("script").textContent).toBe("");
  });

  it("giữ giọng đọc đã chọn cho video tiếp theo thay vì bắt chọn lại (bug report)", () => {
    // Trước khi sửa: RESET (và một draft mới sau khi submit) luôn đưa
    // voiceId về null, khiến NarrationPanel tự chọn giọng đầu tiên trong danh
    // sách — Creator phải chọn lại giọng mình muốn mỗi lần tạo video mới.
    const first = render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    fireEvent.click(screen.getByText("pick-voice"));
    expect(screen.getByTestId("voice-id").textContent).toBe("voice-b");
    fireEvent.click(screen.getByText("submit"));
    first.unmount();

    // Draft mới, project khác — mô phỏng Creator quay lại tạo video thứ hai.
    render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    expect(screen.getByTestId("script").textContent).toBe("");
    expect(screen.getByTestId("voice-id").textContent).toBe("voice-b");
  });

  it("RESUME_EDITING clears hasSubmitted without touching script or project id (CR-024 reject flow)", () => {
    render(
      <ProjectDraftProvider>
        <ResetConsumer />
      </ProjectDraftProvider>,
    );
    fireEvent.click(screen.getByText("write"));
    fireEvent.click(screen.getByText("submit"));
    const projectIdAfterSubmit = screen.getByTestId("project-id").textContent;
    expect(screen.getByTestId("submitted").textContent).toBe("true");

    fireEvent.click(screen.getByText("resume"));

    expect(screen.getByTestId("submitted").textContent).toBe("false");
    expect(screen.getByTestId("script").textContent).toBe("keep me");
    expect(screen.getByTestId("project-id").textContent).toBe(projectIdAfterSubmit);
  });
});
