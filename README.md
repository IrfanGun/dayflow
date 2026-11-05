# DailyFlow v0.3 — portable Windows daily tracker

Personal task scheduling with recurring activities, per-day progress, activity output, and reflection. Desktop data stays local. No account or telemetry in the application code. The frontend uses Svelte 5, Vite, Tailwind CSS 4, and locally bundled Plus Jakarta Sans.

## Included

- Add, edit, and delete activities with date, start time, planned duration, category, and notes.
- Repeat never, daily, Monday–Friday, weekly, or on selected weekdays.
- Track each occurrence independently, including actual minutes, activity, output, and daily summary.
- View daily completion progress and past-day history.
- Starts as a 310×160 frameless task widget near the bottom-right of the primary monitor. Drag the widget body to move it; use **Completed** to open the Activity Result Form or **Skip** to move on to another task. The widget can be covered by other windows. Use the top-right open button for the full planner or the power button to quit. The Windows minimize button minimizes the planner normally; the task-widget button in the planner switches to the widget.
- Store data locally beside the EXE in `DailyFlow-data/tasks.json` for portable backups.

## Build on Windows

Install **Go 1.26+**, **Node.js 20.19+**, and the **Wails 3.1 CLI**. Follow the [Wails installation guide](https://v3.wails.io/getting-started/installation/) for Windows WebView2 and build requirements.

```powershell
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0
wails3 setup
cd frontend
npm install
cd ..
go mod tidy
$env:WAILS_EXP_USE_WAKE = "1"
wails3 dev
```

The YAML build pipeline used by this project is enabled with `WAILS_EXP_USE_WAKE=1` in Wails 3.1. For a production build, keep that environment variable set and run:

```powershell
wails3 build
```

The compiled frontend is embedded in the portable EXE. If the selected Vite port is occupied, choose another one with `wails3 dev -port 34117`.

## Live UI development

```powershell
cd frontend
npm install
npm run dev
```

Open the Vite URL shown in the terminal for an interactive browser preview backed by localStorage. Preview data is separate from desktop data. With `wails3 dev`, Wails sets the frontend port and Vite uses that value; standalone preview defaults to port 34116 and can move to the next free port if necessary.

## Important limitations

- No system-tray icon. Closing the full planner returns to the task widget; closing the widget opens the full planner. The Windows minimize button keeps its normal taskbar behavior.
- Scheduled start and duration are displayed but do not trigger notifications or start and stop timers.
- Daily summaries are authored by the user; AI evaluation is optional and requires an API key.
- The folder beside the EXE must be writable. Do not put the app in `Program Files`.

## Safety notes

Build from this source on your own machine. Keep backups of `DailyFlow-data/tasks.json`. Imported and user text is escaped before display, and the frontend has no remote dependencies.
