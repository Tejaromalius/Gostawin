# Gostawin

## Core Idea

A **background service + CLI tool** that:

1. **Monitors** active application windows (e.g., `firefox`, `code`, `terminal`).
2. **Logs** how long each window stays open.
3. **Aggregates** usage time over **24h (daily)** and **168h (weekly)**.

## How It Works

### 1. Tracker Service (Daemon)

- **Polls** the system every **60 seconds** (via `wmctrl`) to detect:
  - **New windows** → Logs their `WM_CLASS` (app name) and start time.
  - **Closed windows** → Records the duration they were open.
- **Stores** data in an **GOB** (lightweight, binary object).
- **Tracks**:
  - `app_name` (e.g., `firefox`, `gnome-terminal`)
  - `duration` (seconds active)

### 2. CLI Tool

- **Queries** the database to show:
  - **"Total time open per app"** (e.g., "Firefox: 5h23m today").
  - **Weekly summaries** ("VSCode: 32h this week").
- **Filters** by:
  - Time range (`--hours 24` or `--hours 168`).
  - App name (`--app firefox`).
