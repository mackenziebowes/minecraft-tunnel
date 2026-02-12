import { beforeAll, vi } from "vitest";

beforeAll(() => {
  // Mock Wails runtime
  global.window = global.window || {};
  global.window.runtime = {
    EventsOn: vi.fn(),
    EventsOff: vi.fn(),
    EventsEmit: vi.fn(),
    EventsOnMultiple: vi.fn(),
    EventsOffAll: vi.fn(),
    EventsOnce: vi.fn(),
    OpenFileDialog: vi.fn(),
    SaveFileDialog: vi.fn(),
    LogPrint: vi.fn(),
    LogTrace: vi.fn(),
    LogDebug: vi.fn(),
    LogInfo: vi.fn(),
    LogWarning: vi.fn(),
    LogError: vi.fn(),
    LogFatal: vi.fn(),
  };

  // Mock Wails Go bindings
  global.window.go = {
    main: {
      App: {
        CreateOffer: vi.fn(),
        AcceptOffer: vi.fn(),
        AcceptAnswer: vi.fn(),
        StartHostProxy: vi.fn(),
        StartJoinerProxy: vi.fn(),
        ExportToFile: vi.fn(),
        ImportFromFile: vi.fn(),
      },
    },
  };

  // Mock clipboard API - use Object.defineProperty for readonly properties
  if (!global.navigator.clipboard) {
    Object.defineProperty(global.navigator, "clipboard", {
      value: {
        writeText: vi.fn().mockResolvedValue(undefined),
        readText: vi.fn().mockResolvedValue(""),
      },
      writable: true,
      configurable: true,
    });
  }
});
