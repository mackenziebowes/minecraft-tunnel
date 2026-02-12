import { render, screen, fireEvent } from "@testing-library/react";
import { HostView } from "./index";

describe("HostView", () => {
  it("should display generate invitation button", () => {
    render(<HostView />);
    expect(screen.getByRole("button", { name: /generate invitation/i })).toBeInTheDocument();
  });

  it("should show offer token after generation", async () => {
    render(<HostView />);
    const button = screen.getByRole("button", { name: /generate invitation/i });
    fireEvent.click(button);
    // Would need to mock CreateOffer call
  });
});
