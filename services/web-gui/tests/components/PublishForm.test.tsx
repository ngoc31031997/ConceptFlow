import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { PublishForm } from "../../src/components/PublishForm";

describe("PublishForm", () => {
  it("disables submit until title is filled", () => {
    render(<PublishForm projectId="p1" onSubmit={vi.fn()} isSubmitting={false} />);
    expect(screen.getByTestId("publish-form-submit-button")).toBeDisabled();

    fireEvent.change(screen.getByTestId("publish-form-title-input"), {
      target: { value: "My Video" },
    });
    expect(screen.getByTestId("publish-form-submit-button")).not.toBeDisabled();
  });

  it("submits metadata with default private visibility", () => {
    const onSubmit = vi.fn();
    render(<PublishForm projectId="p1" onSubmit={onSubmit} isSubmitting={false} />);

    fireEvent.change(screen.getByTestId("publish-form-title-input"), {
      target: { value: "My Video" },
    });
    fireEvent.submit(screen.getByTestId("publish-form-submit-button").closest("form")!);

    expect(onSubmit).toHaveBeenCalledWith(
      expect.objectContaining({ youtube_title: "My Video", visibility: "private" }),
    );
  });
});
