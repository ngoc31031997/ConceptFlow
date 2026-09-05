import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { useSSE } from "../../src/hooks/useSSE";

class FakeEventSource {
  static instances: FakeEventSource[] = [];
  onmessage: ((event: MessageEvent) => void) | null = null;
  closed = false;

  constructor(public url: string) {
    FakeEventSource.instances.push(this);
  }

  emit(data: unknown) {
    this.onmessage?.({ data: JSON.stringify(data) } as MessageEvent);
  }

  close() {
    this.closed = true;
  }
}

describe("useSSE", () => {
  beforeEach(() => {
    FakeEventSource.instances = [];
    // @ts-expect-error test stub
    global.EventSource = FakeEventSource;
  });

  it("updates state on incoming progress message", async () => {
    const { result } = renderHook(() => useSSE("project-1"));

    const source = FakeEventSource.instances[0];
    source.emit({
      project_id: "project-1",
      step: "render_scenes",
      status: "in_progress",
      scene_index: 2,
      scene_total: 5,
    });

    await waitFor(() => expect(result.current.currentStep).toBe("render_scenes"));
    expect(result.current.sceneIndex).toBe(2);
    expect(result.current.sceneTotal).toBe(5);
  });

  it("closes the EventSource on unmount", () => {
    const { unmount } = renderHook(() => useSSE("project-1"));
    const source = FakeEventSource.instances[0];
    unmount();
    expect(source.closed).toBe(true);
  });
});
