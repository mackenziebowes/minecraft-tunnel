import { render, fireEvent } from "@testing-library/react";
import { JoinView } from "./index";
import { beforeEach, afterEach, describe, it, expect, vi } from "bun:test";

let window: any;

beforeEach(() => {
  const { Window } = require("happy-dom");
  window = new Window();
  global.window = window;
  global.document = window.document;
  global.navigator = window.navigator;

  // Mock Wails runtime
  window.runtime = {
    EventsOn: vi.fn(),
    EventsOff: vi.fn(),
    EventsEmit: vi.fn(),
    EventsOnMultiple: vi.fn(),
    EventsOffAll: vi.fn(),
    EventsOnce: vi.fn(),
    OpenFileDialog: vi.fn(),
    SaveFileDialog: vi.fn(),
  };
});

afterEach(() => {
  window.happyDOM?.close();
});

describe("JoinView", () => {
  it("should display paste offer input", () => {
    const { container } = render(<JoinView />);
    const input = container.querySelector('input[placeholder*="Paste offer token"]');
    expect(input).not.toBeNull();
  });

  it("should have import token button", () => {
    const { container } = render(<JoinView />);
    expect(container.textContent).toContain("Import");
  });
});
