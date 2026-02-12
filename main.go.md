# main.go

Last Updated: 2026-02-12T19:00:00Z

## Purpose

Entry point for the Wails desktop application. Initializes the app, configures window properties, and embeds the frontend assets.

## Stage-Actor-Prop Overview

The Wails runtime acts as the Stage, the main function is the Director that orchestrates app initialization, and the App instance is the Actor bound to the frontend.

## Components

### `main()`
- **Stage**: Application runtime environment
- **Actor**: Main function orchestrates startup
- **Props**: Wails options (window size, assets, colors, binding)

Creates and runs the Wails application with predefined configuration:
- Embeds frontend dist folder
- Sets window dimensions (1024x768)
- Binds App struct for frontend communication

## Usage

Run with `wails dev` for development or `wails build` for production builds.

## Dependencies

- `github.com/wailsapp/wails/v2` - Wails framework
- Embedded `frontend/dist` assets for frontend serving

## Notes

- Background color is dark blue-gray (#1b2636)
- `app.startup` is called when application launches
- App struct methods become callable from React frontend via Wails bindings
