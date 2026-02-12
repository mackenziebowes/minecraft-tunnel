import { useTunnelStore } from "./tunnelStore";

describe("tunnelStore", () => {
  it("should initialize with default state", () => {
    const store = useTunnelStore.getState();
    expect(store.status).toBe("disconnected");
    expect(store.offerToken).toBe("");
    expect(store.answerToken).toBe("");
  });
});
