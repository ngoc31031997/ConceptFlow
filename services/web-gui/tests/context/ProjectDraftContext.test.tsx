import { describe, it, expect } from "vitest";
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
