package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/wailsapp/wails/v3/pkg/application"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Task struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	Category string `json:"category"`
	Date     string `json:"date"`
	Start    string `json:"start"`
	Duration int    `json:"duration"`
	Repeat   string `json:"repeat"`   // none, daily, weekdays, weekly
	Weekdays []int  `json:"weekdays"` // 0=Sunday
	Color    string `json:"color,omitempty"`
	Notes    string `json:"notes"`
	Created  string `json:"created"`
}
type Result struct {
	TaskID        string `json:"taskId"`
	Date          string `json:"date"`
	Done          bool   `json:"done"`
	Status        string `json:"status,omitempty"`
	Activity      string `json:"activity"`
	Output        string `json:"output"`
	Summary       string `json:"summary"`
	ActualMinutes int    `json:"actualMinutes"`
	AIScore       int    `json:"aiScore,omitempty"`
	AIEvaluation  string `json:"aiEvaluation,omitempty"`
}
type Data struct {
	Tasks   []Task   `json:"tasks"`
	Results []Result `json:"results"`
}
type Entry struct {
	Task   Task   `json:"task"`
	Result Result `json:"result"`
}
type Day struct {
	Date    string  `json:"date"`
	Entries []Entry `json:"entries"`
	Total   int     `json:"total"`
	Done    int     `json:"done"`
	Percent int     `json:"percent"`
	Summary string  `json:"summary"`
}
type App struct {
	wailsApp          *application.App
	window            *application.WebviewWindow
	widgetWindow      *application.WebviewWindow
	tray              *application.SystemTray
	mu                sync.Mutex
	trayMu            sync.Mutex
	path              string
	settingsPath      string
	preferencesPath   string
	trayIconEnabled   bool
	aiAPIKeys         map[string]string
	aiModelIDs        map[string]string
	aiDefaultProvider string
	data              Data
	mini              atomic.Bool
	quitting          atomic.Bool
}

func NewApp() *App {
	return &App{data: Data{Tasks: []Task{}, Results: []Result{}}, aiAPIKeys: map[string]string{}, aiModelIDs: map[string]string{}, trayIconEnabled: true}
}
func (a *App) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	base := filepath.Dir(exe)
	// Portable data lives beside EXE. This folder must be writable.
	dir := filepath.Join(base, "DailyFlow-data")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	a.path = filepath.Join(dir, "tasks.json")
	a.settingsPath = filepath.Join(dir, "settings.json")
	a.preferencesPath = filepath.Join(dir, "preferences.json")
	if raw, err := os.ReadFile(a.preferencesPath); err == nil {
		var preferences struct {
			TrayIconEnabled *bool `json:"trayIconEnabled"`
		}
		if json.Unmarshal(raw, &preferences) == nil && preferences.TrayIconEnabled != nil {
			a.trayIconEnabled = *preferences.TrayIconEnabled
		}
	}
	contents, err := os.ReadFile(a.path)
	if err == nil {
		_ = json.Unmarshal(contents, &a.data)
	}
	if a.data.Tasks == nil {
		a.data.Tasks = []Task{}
	}
	if a.data.Results == nil {
		a.data.Results = []Result{}
	}
	a.loadAISecret()
	return nil
}
func (a *App) save() error {
	if a.path == "" {
		return errors.New("data directory is unavailable; run from a writable folder")
	}
	raw, err := json.MarshalIndent(a.data, "", "  ")
	if err != nil {
		return err
	}
	temp := a.path + ".tmp"
	if err = os.WriteFile(temp, raw, 0600); err != nil {
		return err
	}
	return os.Rename(temp, a.path)
}
func validDate(s string) bool { _, err := time.Parse("2006-01-02", s); return err == nil }
func validTime(s string) bool { _, err := time.Parse("15:04", s); return err == nil }
func (a *App) SaveTask(t Task) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	t.Title = strings.TrimSpace(t.Title)
	if t.Title == "" {
		return errors.New("task title required")
	}
	if !validDate(t.Date) || !validTime(t.Start) || t.Duration < 1 || t.Duration > 1440 {
		return errors.New("invalid date, time, or duration")
	}
	switch t.Repeat {
	case "none", "daily", "weekdays", "weekly", "custom":
	default:
		return errors.New("invalid recurrence")
	}
	if t.Category == "" {
		t.Category = "General"
	}
	if t.ID == "" {
		t.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		t.Created = time.Now().Format(time.RFC3339)
	}
	for i, old := range a.data.Tasks {
		if old.ID == t.ID {
			a.data.Tasks[i] = t
			return a.save()
		}
	}
	a.data.Tasks = append(a.data.Tasks, t)
	return a.save()
}
func (a *App) DeleteTask(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	tasks := []Task{}
	for _, t := range a.data.Tasks {
		if t.ID != id {
			tasks = append(tasks, t)
		}
	}
	a.data.Tasks = tasks
	results := []Result{}
	for _, r := range a.data.Results {
		if r.TaskID != id {
			results = append(results, r)
		}
	}
	a.data.Results = results
	return a.save()
}
func (a *App) SaveResult(r Result) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if !validDate(r.Date) || r.ActualMinutes < 0 || r.ActualMinutes > 1440 {
		return errors.New("invalid result")
	}
	if r.Done && strings.TrimSpace(r.Activity) == "" {
		return errors.New("activity is required before completing a task")
	}
	switch r.Status {
	case "", "todo", "progress", "completed", "skipped":
	default:
		return errors.New("invalid task status")
	}
	found := false
	for _, t := range a.data.Tasks {
		if t.ID == r.TaskID && occurs(t, r.Date) {
			found = true
			break
		}
	}
	if !found {
		return errors.New("no scheduled task on this date")
	}
	for i, old := range a.data.Results {
		if old.TaskID == r.TaskID && old.Date == r.Date {
			a.data.Results[i] = r
			return a.save()
		}
	}
	a.data.Results = append(a.data.Results, r)
	return a.save()
}
func occurs(t Task, date string) bool {
	d, e1 := time.Parse("2006-01-02", date)
	start, e2 := time.Parse("2006-01-02", t.Date)
	if e1 != nil || e2 != nil || d.Before(start) {
		return false
	}
	switch t.Repeat {
	case "daily":
		return true
	case "weekdays":
		return d.Weekday() != time.Saturday && d.Weekday() != time.Sunday
	case "weekly":
		return d.Weekday() == start.Weekday()
	case "custom":
		for _, w := range t.Weekdays {
			if w == int(d.Weekday()) {
				return true
			}
		}
		return false
	default:
		return date == t.Date
	}
}
func (a *App) Day(date string) (Day, error) {
	if !validDate(date) {
		return Day{}, errors.New("invalid date")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	day := Day{Date: date, Entries: []Entry{}}
	for _, t := range a.data.Tasks {
		if !occurs(t, date) {
			continue
		}
		entry := Entry{Task: t, Result: Result{TaskID: t.ID, Date: date}}
		for _, r := range a.data.Results {
			if r.TaskID == t.ID && r.Date == date {
				entry.Result = r
				break
			}
		}
		day.Entries = append(day.Entries, entry)
		if entry.Result.Done {
			day.Done++
		}
	}
	sort.Slice(day.Entries, func(i, j int) bool { return day.Entries[i].Task.Start < day.Entries[j].Task.Start })
	day.Total = len(day.Entries)
	if day.Total > 0 {
		day.Percent = day.Done * 100 / day.Total
	}
	for _, e := range day.Entries {
		if e.Result.Summary != "" {
			if day.Summary != "" {
				day.Summary += "\n"
			}
			day.Summary += "• " + e.Task.Title + ": " + e.Result.Summary
		}
	}
	return day, nil
}
func (a *App) AllTasks() []Task {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := append([]Task{}, a.data.Tasks...)
	return out
}
func (a *App) AllResults() []Result {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := append([]Result{}, a.data.Results...)
	return out
}

func (a *App) OpenActivityResult(taskID, date string) error {
	if strings.TrimSpace(taskID) == "" || !validDate(date) {
		return errors.New("invalid task or date")
	}
	if a.window == nil {
		return errors.New("planner window is unavailable")
	}
	a.RestoreFull()
	taskIDJSON, _ := json.Marshal(taskID)
	dateJSON, _ := json.Marshal(date)
	a.window.ExecJS(fmt.Sprintf("window.dailyFlowOpenResult?.(%s,%s)", taskIDJSON, dateJSON))
	return nil
}

const widgetWidth = 310
const widgetHeight = 160

// Closing the window switches to widget mode. Only ExitApp is allowed to quit.
func (a *App) beforeClose(event *application.WindowEvent) {
	if a.quitting.Load() {
		return
	}
	a.enterWidget()
	event.Cancel()
}

func (a *App) beforeWidgetClose(event *application.WindowEvent) {
	if a.quitting.Load() {
		return
	}
	a.RestoreFull()
	event.Cancel()
}

func (a *App) enterWidget() {
	if a.mini.Load() {
		return
	}
	a.mini.Store(true)
	if a.window == nil || a.widgetWindow == nil {
		return
	}
	a.window.Hide()
	a.widgetWindow.SetAlwaysOnTop(false).SetSize(widgetWidth, widgetHeight)
	screens := a.wailsApp.Screen.GetAll()
	width, height := 1280, 720
	for _, s := range screens {
		if s.IsPrimary {
			width, height = s.WorkArea.X+s.WorkArea.Width, s.WorkArea.Y+s.WorkArea.Height
			break
		}
	}
	x := width - widgetWidth - 20
	y := height - widgetHeight - 20
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	a.widgetWindow.SetPosition(x, y)
	a.widgetWindow.Show()
	a.widgetWindow.Focus()
}

// Used by the mini overview button and by the window close handler.
func (a *App) ToggleOverview() {
	if a.mini.Load() {
		a.RestoreFull()
		return
	}
	a.enterWidget()
}
func (a *App) RestoreFull() {
	if a.window == nil || a.widgetWindow == nil {
		return
	}
	a.mini.Store(false)
	a.window.ExecJS("window.dailyFlowSetMini && window.dailyFlowSetMini(false)")
	a.widgetWindow.Hide()
	if a.window.IsMinimised() {
		a.window.Restore()
	}
	a.window.Show()
	a.window.Maximise()
	a.window.Focus()
}
func (a *App) ExitApp() {
	if a.wailsApp == nil {
		return
	}
	a.quitting.Store(true)
	a.wailsApp.Quit()
}

func (a *App) GetTrayIconEnabled() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.trayIconEnabled
}

func (a *App) SetTrayIconEnabled(enabled bool) error {
	a.mu.Lock()
	if a.preferencesPath == "" {
		a.mu.Unlock()
		return errors.New("settings folder is unavailable")
	}
	raw, err := json.MarshalIndent(struct {
		TrayIconEnabled bool `json:"trayIconEnabled"`
	}{enabled}, "", "  ")
	if err == nil {
		err = writePrivateFile(a.preferencesPath, raw)
	}
	if err != nil {
		a.mu.Unlock()
		return err
	}
	a.trayIconEnabled = enabled
	a.mu.Unlock()
	if enabled {
		setupTray(a)
	} else {
		a.trayMu.Lock()
		if a.tray != nil {
			a.tray.Hide()
			a.tray.Destroy()
			a.tray = nil
		}
		a.trayMu.Unlock()
	}
	return nil
}
