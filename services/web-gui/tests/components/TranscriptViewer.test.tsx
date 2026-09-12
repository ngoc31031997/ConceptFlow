import { describe, it, expect, vi, afterEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { TranscriptViewer } from "../../src/components/TranscriptViewer";
import type { Scene } from "../../src/types";

const mockScenes: Scene[] = [
  {
    scene_index: 0,
    narration_text: "First narration text",
    duration_seconds: 3.5,
  },
  {
    scene_index: 1,
    narration_text: "Second narration text",
    duration_seconds: 2.8,
  },
  {
    scene_index: 2,
    narration_text: "Third narration text",
    duration_seconds: 4.2,
  },
];

describe("TranscriptViewer", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("renders nothing when no narration", () => {
    const emptyScenes: Scene[] = [
      { scene_index: 0, narration_text: "" },
    ];

    const { container } = render(
      <TranscriptViewer scenes={emptyScenes} contentLanguage="vi" />
    );

    expect(container.firstChild).toBeNull();
  });

  it("renders transcript viewer when narration exists", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);
    expect(screen.getByTestId("transcript-viewer-toggle")).toBeInTheDocument();
  });

  it("shows correct narration count", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);
    expect(screen.getByText("3 đoạn lời thoại")).toBeInTheDocument();
  });

  it("shows English labels when contentLanguage is en", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="en" />);
    expect(screen.getByText("Text Transcript")).toBeInTheDocument();
    expect(screen.getByText("3 narration segments")).toBeInTheDocument();
  });

  it("toggles content visibility", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);

    const toggle = screen.getByTestId("transcript-viewer-toggle");

    // Initially closed
    expect(screen.queryByTestId("transcript-viewer-content")).not.toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "false");

    // Open
    fireEvent.click(toggle);
    expect(screen.getByTestId("transcript-viewer-content")).toBeInTheDocument();
    expect(toggle).toHaveAttribute("aria-expanded", "true");

    // Close again
    fireEvent.click(toggle);
    expect(screen.queryByTestId("transcript-viewer-content")).not.toBeInTheDocument();
  });

  it("displays all narration texts when open", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);

    fireEvent.click(screen.getByTestId("transcript-viewer-toggle"));

    expect(screen.getByText("First narration text")).toBeInTheDocument();
    expect(screen.getByText("Second narration text")).toBeInTheDocument();
    expect(screen.getByText("Third narration text")).toBeInTheDocument();
  });

  it("downloads transcript as text file", () => {
    // Mock URL.createObjectURL and document methods
    const mockCreateObjectURL = vi.fn(() => "blob:mock-url");
    const mockRevokeObjectURL = vi.fn();
    global.URL.createObjectURL = mockCreateObjectURL;
    global.URL.revokeObjectURL = mockRevokeObjectURL;

    const mockClick = vi.fn();

    const originalCreateElement = document.createElement.bind(document);
    vi.spyOn(document, "createElement").mockImplementation((tag) => {
      if (tag === "a") {
        return {
          click: mockClick,
          href: "",
          download: "",
        } as any;
      }
      return originalCreateElement(tag);
    });

    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);

    // Mocked only now, after RTL has already appended its own container —
    // mocking these before render() breaks RTL itself, since it relies on
    // appendChild's real return value (the appended node) as the container.
    const mockAppendChild = vi.spyOn(document.body, "appendChild").mockImplementation((node) => node);
    const mockRemoveChild = vi.spyOn(document.body, "removeChild").mockImplementation((node) => node);

    fireEvent.click(screen.getByTestId("transcript-viewer-toggle"));
    fireEvent.click(screen.getByTestId("transcript-download-button"));

    expect(mockCreateObjectURL).toHaveBeenCalled();
    expect(mockClick).toHaveBeenCalled();
    expect(mockAppendChild).toHaveBeenCalled();
    expect(mockRemoveChild).toHaveBeenCalled();
    expect(mockRevokeObjectURL).toHaveBeenCalled();

    // Cleanup
    mockAppendChild.mockRestore();
    mockRemoveChild.mockRestore();
  });

  it("has proper accessibility attributes", () => {
    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);

    const toggle = screen.getByTestId("transcript-viewer-toggle");
    expect(toggle).toHaveAttribute("aria-expanded");

    fireEvent.click(toggle);

    const list = screen.getByRole("list");
    expect(list).toHaveAttribute("aria-label", "Danh sách lời thoại");
  });

  it("handles scenes without narrations field", () => {
    const scenesWithoutNarrations: Scene[] = [
      { scene_index: 0, narration_text: "" },
    ];

    const { container } = render(
      <TranscriptViewer scenes={scenesWithoutNarrations} contentLanguage="vi" />
    );

    expect(container.firstChild).toBeNull();
  });

  it("formats download content correctly for Vietnamese", () => {
    const mockBlob = vi.fn();
    global.Blob = mockBlob as any;

    render(<TranscriptViewer scenes={mockScenes} contentLanguage="vi" />);

    fireEvent.click(screen.getByTestId("transcript-viewer-toggle"));
    fireEvent.click(screen.getByTestId("transcript-download-button"));

    expect(mockBlob).toHaveBeenCalled();
    const content = mockBlob.mock.calls[0][0][0];
    expect(content).toContain("TRANSCRIPT - VIDEO CONCEPTFLOW");
    expect(content).toContain("[Đoạn 1]");
    expect(content).toContain("First narration text");
  });

  it("formats download content correctly for English", () => {
    const mockBlob = vi.fn();
    global.Blob = mockBlob as any;

    render(<TranscriptViewer scenes={mockScenes} contentLanguage="en" />);

    fireEvent.click(screen.getByTestId("transcript-viewer-toggle"));
    fireEvent.click(screen.getByTestId("transcript-download-button"));

    expect(mockBlob).toHaveBeenCalled();
    const content = mockBlob.mock.calls[0][0][0];
    expect(content).toContain("TRANSCRIPT - CONCEPTFLOW VIDEO");
    expect(content).toContain("[Segment 1]");
  });
});
