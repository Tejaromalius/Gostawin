# Gostawin

[![Go Report Card](https://goreportcard.com/badge/github.com/Tejaromalius/Gostawin)](https://goreportcard.com/report/github.com/your_username/gostawin) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Gostawin is a terminal-based application for Linux desktops (specifically X11 Windows) that tracks the cumulative time you spend actively using different application windows. It provides a simple TUI (Text-based User Interface) to view program usage statistics, persisted across sessions.

<!-- ![Gostawin Demo](https://raw.githubusercontent.com/Tejaromalius/Gostawin/main/assets/demo.webp) -->

## Features

* **Active Window Time Tracking:** Monitors currently focused windows and accumulates uptime for each application.
* **Persistent Storage:** Saves usage statistics to a cache file (`~/.cache/gostawin/stats.gob`) so tracking continues across reboots and application restarts.
* **TUI Display:** Uses `bubbletea` and `bubbles` to present a clean, sortable table view of tracked programs and their uptime.
* **Program Renaming:** Allows renaming the displayed program name for better readability (e.g., renaming "Navigator.firefox" to just "Firefox").
* **Responsive UI:** Adapts table column widths to the terminal size.

## How It Works

Gostawin periodically queries the window manager using the `wmctrl -lx` command to get a list of currently open windows and their associated `WM_CLASS` property.

1. **Polling:** Every second (by default), it fetches the list of active window classes.
2. **Uptime Calculation:** It compares the current list with the previous one. For windows that remain active, it increments their stored `Uptime` duration by the time elapsed since the last check.
3. **New Window Detection:** If a new window class appears, it's added to the tracked list with an initial uptime of zero.
4. **Persistence:** The map containing program names, their `WM_CLASS` identifiers, and accumulated uptimes is saved to a `gob` encoded file in the user's cache directory after each update cycle.
5. **TUI Rendering:** The [`bubbletea`](https://github.com/charmbracelet/bubbletea/tree/main) framework renders the data in a sortable table, handling user input for navigation, quitting, and initiating the renaming process.

## Dependencies

### System Dependencies

* **`wmctrl`:** A command-line tool to interact with EWMH/NetWM compatible X Window managers. You likely need to install this using your system's package manager.
  * Debian/Ubuntu: `sudo apt update && sudo apt install wmctrl`
  * Fedora: `sudo dnf install wmctrl`
  * Arch: `sudo pacman -S wmctrl`

### Go Modules

Gostawin relies on the following Go modules (managed via `go.mod`):

* `github.com/charmbracelet/bubbletea`
* `github.com/charmbracelet/bubbles`
* `github.com/charmbracelet/lipgloss`
* `github.com/charmbracelet/x/term`

## Installation

1. **Clone the repository:**

    ```bash
    git clone [https://github.com/Tejaromalius/Gostawin.git](https://github.com/Tejaromalius/Gostawin.git)
    
    cd Gostawin
    ```

2. **Build the application:**

    ```bash
    go build -o gostawin ./src
    ```

    This will create an executable file named `gostawin` in the current directory.

3. **Install (Optional):** Move the executable to a directory in your system's `PATH`, for example:

    ```bash
    sudo mv gostawin /usr/local/bin/
    ```

## Usage

Simply run the executable from your terminal:

```bash
gostawin
```

### Controls

* Up/Down Arrows: Navigate the program list.
* Enter: Select the highlighted program to rename it.
* In rename mode: Type the new name and press Enter to confirm, or Esc to cancel.
* r / R: Manually refresh the window list (usually not needed due to automatic refresh).
* q / Ctrl+C: Quit the application.
* Esc: Cancel renaming or blur/focus the table (less relevant now).

## Cache

Usage data is stored in: `~/.cache/gostawin/stats.gob`

You can safely delete this file if you want to reset all tracked times and custom names. The directory and file will be recreated automatically when you next run the application.
Contributing

---
Contributions are welcome! Please feel free to open an issue or submit a pull request.
