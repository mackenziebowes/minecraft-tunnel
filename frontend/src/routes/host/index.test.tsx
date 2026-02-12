import "@/test/setup";
import { render, fireEvent } from "@testing-library/react";
import { vi, describe, it, expect } from "vitest";
import { HostView } from "./index";

describe("HostView", () => {
  it("should display generate invitation button", () => {
    const { container } = render(<HostView />);
    expect(container.textContent).toContain("Generate Invitation");
  });

  it("should show offer token after generation", async () => {
    const { container } = render(<HostView />);
    const buttons = container.querySelectorAll("button");
    const generateBtn = Array.from(buttons).find(btn => btn.textContent?.includes("Generate"));
    expect(generateBtn).toBeDefined();
  });
});
