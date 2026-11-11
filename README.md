# DailyFlow v0.3 — portable Windows daily tracker

Personal task scheduling with recurring activities, per-day progress, activity output, and reflection. Desktop data stays local. No account or telemetry in the application code. The frontend uses Svelte 5, Vite, Tailwind CSS 4, and locally bundled Plus Jakarta Sans.

## Included

- Add, edit, and delete activities with date, start time, planned duration, category, and notes.
- Group scheduled activities into projects and track objectives; create tasks from a project or assign and move existing tasks from their schedule.
- Repeat never, daily, Monday–Friday, weekly, or on selected weekdays.
- Track each occurrence independently, including actual minutes, activity, output, and daily summary.
- View daily completion progress and past-day history.
- Open the Productivity Matrix for 7, 30, 90, or custom date ranges, with effort, consistency, reported outputs, comparable performance trends, and offline scoring.
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

## Productivity Matrix scoring

The Matrix derives its values from saved activity results; nothing is awarded by a server or persisted as a duplicate ledger. Updating or deleting a result therefore recalculates the totals.

- Activity Points (AP): 10 points per recorded session, plus 1 point per 30 actual minutes, capped at 10 effort points per session.
- Growth Points (GP): before AI Evaluate, 5 points per reported output item and 10 points when an activity records more output items than its previous recorded session. After AI Evaluate, that session's GP is replaced by the AI's 0–100 estimate based on substantive growth evidenced by its recorded output and summary. Re-evaluating replaces the previous AI estimate. Time alone never earns GP.
- Growth Score: compares a selected period with the immediately preceding period of equal length using the system Effort / Consistency / Output / Output Trend weights of 15 / 20 / 30 / 35. Output trend compares reported output items per session for the same activity. If comparable history is missing, the overall score reads “Insufficient Data” instead of treating missing data as zero.

These defaults are transparent product rules, not scientifically validated measures. The system derives output counts from the output descriptions already recorded with each activity. Reported outputs are not treated as independently verified ability.

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
