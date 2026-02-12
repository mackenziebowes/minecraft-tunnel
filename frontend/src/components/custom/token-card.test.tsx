import { render, screen } from "@testing-library/react";
import { TokenCard } from "./token-card";
import { beforeEach, afterEach, describe, it, expect } from "bun:test";

let window: any;

beforeEach(() => {
  const { Window } = require("happy-dom");
  window = new Window();
  global.window = window;
  global.document = window.document;
  global.navigator = window.navigator;
});

afterEach(() => {
  window.happyDOM?.close();
});

describe("TokenCard", () => {
  it("should display token text", () => {
    const { container } = render(<TokenCard token="test-token-123" type="offer" />);
    const input = container.querySelector('input');
    expect(input?.value).toBe("test-token-123");
  });

  it("should have copy button", () => {
    const { container } = render(<TokenCard token="test-token" type="answer" />);
    expect(container.textContent).toContain("Copy");
  });
});
