# AGENTS.md - Development Guide for Agentic Coding

## Build & Development Commands

### Go Backend
- **Development**: `wails dev` - Runs app with hot reload (frontend on http://localhost:5173)
- **Production build**: `wails build -clean` or `./scripts/build.sh`
- **Cross-platform builds**:
  - `./scripts/build-all.sh` - All platforms
  - `./scripts/build-linux.sh` - Linux AMD64
  - `./scripts/build-windows.sh` - Windows
  - `./scripts/build-macos-arm.sh` - macOS Apple Silicon

### Frontend (React/TypeScript)
- **Development**: `npm run dev` - Starts Vite dev server
- **Build**: `npm run build` - TypeScript compile + Vite build
- **Lint**: `npm run lint` - ESLint check
- **Preview**: `npm run preview` - Preview production build

### Testing
- **Go tests**: `go test ./...` - Run all Go tests
- **Single Go test**: `go test -run TestFunctionName ./path/to/package`
- **No test framework configured** - Currently no test files exist; add Go tests as `_test.go` or Jest/Vitest for frontend

## Code Style Guidelines

### Go Backend

**Formatting**: Use `gofmt` or `go fmt ./...` before commits

**Import Groups**: Separate with blank lines:
```go
import (
    "context"
    "net"

    "github.com/wailsapp/wails/v2/pkg/runtime"
)
```

**Naming**:
- Exported: PascalCase (e.g., `CreateOffer`, `App`)
- Unexported: camelCase (e.g., `startup`, `pumpMinecraftToChannel`)
- Interfaces: Usually no "I" prefix (e.g., `TunnelStore`)

**Error Handling**:
- Always check errors; early returns preferred
- Use `runtime.EventsEmit(ctx, "log", "message")` for logging to frontend

**Structs**:
- Embed context in app structs: `ctx context.Context`
- Use JSON tags for serialization: `type Signal struct { SDP string `json:"sdp"` }`

### Frontend (TypeScript/React)

**Import Order**:
1. External libraries
2. Internal (separated by blank line)
3. Relative/absolute imports with `@/` alias
```tsx
import React, { useEffect } from "react";
import { Button } from "@/components/ui/button";
import { useTunnelStore } from "@/lib/tunnelStore";
```

**Components**:
- Functional components with hooks
- Named exports: `export const HostView = () => { ... }`
- Props: Define interfaces above component

**State Management**:
- Use Zustand stores in `@/lib/` directory
- Store pattern: `create<TypeState>((set, get) => ({ ... }))`

**Styling**:
- Tailwind utility classes
- Use `cn()` from `@/lib/utils` for dynamic classes
- shadcn/ui components from `@/components/ui/`
- Lucide React icons

**Event Handling**:
- Wails events: `EventsOn("event-name", handler)`, `EventsOff("event-name")`
- Cleanup in useEffect return

## Project Structure

```
.
├── main.go              # Wails entry point
├── app.go               # Go application logic
├── frontend/
│   ├── src/
│   │   ├── App.tsx
│   │   ├── main.tsx
│   │   ├── components/ui/    # shadcn/ui components
│   │   ├── components/custom/# Custom components
│   │   ├── lib/              # Utils, stores
│   │   └── routes/           # Page components
│   ├── package.json
│   └── vite.config.ts
├── scripts/             # Build scripts
└── wails.json           # Wails config
```

## Key Patterns

### Go-Frontend Communication
- Exported Go functions become available to frontend via Wails bindings
- Use `runtime.EventsEmit(ctx, "event", data)` to push data to React
- Frontend listeners: `EventsOn("event", callback)`; cleanup in useEffect

### TypeScript Configuration
- Strict mode enabled
- Path alias: `@/*` maps to `./src/*`
- Target: ESNext, Module: ESNext

### Adding Dependencies
- Go: Add to go.mod, run `go mod tidy`
- Frontend: `npm install package` or `npx shadcn@latest add component`

## Notes

- No test infrastructure currently set up - add Go tests (`_test.go`) or configure Vitest/Jest for frontend
- ESLint uses default configuration - no custom rules defined
- Wails generates `frontend/wailsjs/` bindings automatically - do not edit
