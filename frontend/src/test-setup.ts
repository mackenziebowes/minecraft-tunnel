import { expect, afterEach, vi } from 'vitest'
import { cleanup } from '@testing-library/react'
import * as matchers from '@testing-library/jest-dom/matchers'

expect.extend(matchers)

// Mock Wails runtime before any imports
Object.defineProperty(global.window, 'runtime', {
  value: {
    EventsOnMultiple: () => vi.fn(),
    EventsOn: () => vi.fn(),
    EventsOff: () => vi.fn(),
    EventsOffAll: () => vi.fn(),
    EventsOnce: () => vi.fn(),
    EventsEmit: () => vi.fn(),
    LogPrint: () => vi.fn(),
    LogTrace: () => vi.fn(),
    LogDebug: () => vi.fn(),
    LogInfo: () => vi.fn(),
    LogWarning: () => vi.fn(),
    LogError: () => vi.fn(),
    LogFatal: () => vi.fn(),
    WindowReload: () => vi.fn(),
    WindowReloadApp: () => vi.fn(),
    WindowSetAlwaysOnTop: () => vi.fn(),
    WindowSetSystemDefaultTheme: () => vi.fn(),
    WindowSetLightTheme: () => vi.fn(),
    WindowSetDarkTheme: () => vi.fn(),
    WindowCenter: () => vi.fn(),
    WindowSetTitle: () => vi.fn(),
    WindowFullscreen: () => vi.fn(),
    WindowUnfullscreen: () => vi.fn(),
    WindowIsFullscreen: () => vi.fn(),
    WindowGetSize: () => vi.fn(),
    WindowSetSize: () => vi.fn(),
    WindowSetMaxSize: () => vi.fn(),
    WindowSetMinSize: () => vi.fn(),
    WindowSetPosition: () => vi.fn(),
    WindowGetPosition: () => vi.fn(),
    WindowHide: () => vi.fn(),
    WindowShow: () => vi.fn(),
    WindowMaximise: () => vi.fn(),
    WindowToggleMaximise: () => vi.fn(),
    WindowUnmaximise: () => vi.fn(),
    WindowIsMaximised: () => vi.fn(),
    WindowMinimise: () => vi.fn(),
    WindowUnminimise: () => vi.fn(),
    WindowSetBackgroundColour: () => vi.fn(),
    ScreenGetAll: () => vi.fn(),
    WindowIsMinimised: () => vi.fn(),
    WindowIsNormal: () => vi.fn(),
    BrowserOpenURL: () => vi.fn(),
    Environment: () => vi.fn(),
    Quit: () => vi.fn(),
    Hide: () => vi.fn(),
    Show: () => vi.fn(),
    ClipboardGetText: () => vi.fn(),
    ClipboardSetText: () => vi.fn(),
    OnFileDrop: () => vi.fn(),
    OnFileDropOff: () => vi.fn(),
    CanResolveFilePaths: () => vi.fn(),
    ResolveFilePaths: () => vi.fn(),
  },
  writable: true,
})

afterEach(() => {
  cleanup()
})
