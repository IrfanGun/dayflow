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
	ID        string `json:"id"`
	ProjectID string `json:"projectId,omitempty"`
	Title     string `json:"title"`
	Category  string `json:"category"`
	Date      string `json:"date"`
	Start     string `json:"start"`
	Duration  int    `json:"duration"`
	Repeat    string `json:"repeat"`   // none, daily, weekdays, weekly
	Weekdays  []int  `json:"weekdays"` // 0=Sunday
	Color     string `json:"color,omitempty"`
	Notes     string `json:"notes"`
	Created   string `json:"created"`
}
type Project struct {
	ID             string      `json:"id"`
	Name           string      `json:"name"`
	Objectives     []Objective `json:"objectives,omitempty"`
	Created        string      `json:"created"`
	Level          int         `json:"level,omitempty"`
	XP             int         `json:"xp,omitempty"`
	EvaluationData string      `json:"evaluationData,omitempty"`
	EvaluatedAt    string      `json:"evaluatedAt,omitempty"`
}
type Objective struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}
type Result struct {
	TaskID         string `json:"taskId"`
	Date           string `json:"date"`
	Done           bool   `json:"done"`
	Status         string `json:"status,omitempty"`
	Activity       string `json:"activity"`
	Output         string `json:"output"`
	Summary        string `json:"summary"`
	ActualMinutes  int    `json:"actualMinutes"`
	AIScore        int    `json:"aiScore,omitempty"`
	AIGrowthPoints *int   `json:"aiGrowthPoints,omitempty"`
	AIEvaluation   string `json:"aiEvaluation,omitempty"`
}
type Data struct {
	Projects []Project `json:"projects"`
	Tasks    []Task    `json:"tasks"`
	Results  []Result  `json:"results"`
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
	userName          string
	aiAPIKeys         map[string]string
	aiModelIDs        map[string]string
	aiDefaultProvider string
	data              Data
	mini              atomic.Bool
	quitting          atomic.Bool
}

func NewApp() *App {
	return &App{data: Data{Projects: []Project{}, Tasks: []Task{}, Results: []Result{}}, aiAPIKeys: map[string]string{}, aiModelIDs: map[string]string{}, trayIconEnabled: true}
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
			TrayIconEnabled *bool  `json:"trayIconEnabled"`
			UserName        string `json:"userName"`
		}
		if json.Unmarshal(raw, &preferences) == nil {
			if preferences.TrayIconEnabled != nil {
				a.trayIconEnabled = *preferences.TrayIconEnabled
			}
			a.userName = strings.TrimSpace(preferences.UserName)
		}
	}
	contents, err := os.ReadFile(a.path)
	if err == nil {
		_ = json.Unmarshal(contents, &a.data)
	}
	if a.data.Tasks == nil {
		a.data.Tasks = []Task{}
	}
	if a.data.Projects == nil {
		a.data.Projects = []Project{}
	}
	for i := range a.data.Projects {
		if a.data.Projects[i].Objectives == nil {
			a.data.Projects[i].Objectives = []Objective{}
		}
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
	if t.ProjectID != "" && !a.hasProject(t.ProjectID) {
		return errors.New("project was not found")
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

func (a *App) DeleteResult(taskID, date string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	results := make([]Result, 0, len(a.data.Results))
	found := false
	for _, result := range a.data.Results {
		if result.TaskID == taskID && result.Date == date {
			found = true
			continue
		}
		results = append(results, result)
	}
	if !found {
		return errors.New("activity log entry was not found")
	}
	previous := a.data.Results
	a.data.Results = results
	if err := a.save(); err != nil {
		a.data.Results = previous
		return err
	}
	return nil
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
func (a *App) AllProjects() []Project {
	a.mu.Lock()
	defer a.mu.Unlock()
	return append([]Project{}, a.data.Projects...)
}
func (a *App) hasProject(id string) bool {
	for _, project := range a.data.Projects {
		if project.ID == id {
			return true
		}
	}
	return false
}
func (a *App) SaveProject(project Project) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	project.Name = strings.TrimSpace(project.Name)
	if project.Objectives == nil {
		project.Objectives = []Objective{}
	}
	if project.Name == "" || len(project.Name) > 120 {
		return errors.New("enter a project name (up to 120 characters)")
	}
	if project.ID == "" {
		project.ID = fmt.Sprintf("%d", time.Now().UnixNano())
		project.Created = time.Now().Format(time.RFC3339)
	}
	for i, existing := range a.data.Projects {
		if existing.ID == project.ID {
			if project.Created == "" {
				project.Created = existing.Created
			}
			a.data.Projects[i] = project
			return a.save()
		}
	}
	if project.Created == "" {
		project.Created = time.Now().Format(time.RFC3339)
	}
	a.data.Projects = append(a.data.Projects, project)
	return a.save()
}
func (a *App) SaveProjectObjective(projectID string, objective Objective) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	objective.Title = strings.TrimSpace(objective.Title)
	if objective.Title == "" || len(objective.Title) > 180 {
		return errors.New("objective title is required and must be 180 characters or fewer")
	}
	if objective.ID == "" {
		objective.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	}
	for i := range a.data.Projects {
		if a.data.Projects[i].ID != projectID {
			continue
		}
		previous := a.data.Projects[i]
		updated := append([]Objective{}, previous.Objectives...)
		found := false
		for j := range updated {
			if updated[j].ID == objective.ID {
				updated[j] = objective
				found = true
				break
			}
		}
		if !found {
			updated = append(updated, objective)
		}
		a.data.Projects[i].Objectives = updated
		if err := a.save(); err != nil {
			a.data.Projects[i] = previous
			return err
		}
		return nil
	}
	return errors.New("project was not found")
}
func (a *App) SetProjectObjectiveDone(projectID, objectiveID string, done bool) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.data.Projects {
		if a.data.Projects[i].ID != projectID {
			continue
		}
		for j := range a.data.Projects[i].Objectives {
			if a.data.Projects[i].Objectives[j].ID != objectiveID {
				continue
			}
			previous := a.data.Projects[i].Objectives[j].Done
			a.data.Projects[i].Objectives[j].Done = done
			if err := a.save(); err != nil {
				a.data.Projects[i].Objectives[j].Done = previous
				return err
			}
			return nil
		}
		return errors.New("objective was not found")
	}
	return errors.New("project was not found")
}
func (a *App) DeleteProjectObjective(projectID, objectiveID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	for i := range a.data.Projects {
		if a.data.Projects[i].ID != projectID {
			continue
		}
		previous := a.data.Projects[i].Objectives
		updated := make([]Objective, 0, len(previous))
		found := false
		for _, objective := range previous {
			if objective.ID == objectiveID {
				found = true
				continue
			}
			updated = append(updated, objective)
		}
		if !found {
			return errors.New("objective was not found")
		}
		a.data.Projects[i].Objectives = updated
		if err := a.save(); err != nil {
			a.data.Projects[i].Objectives = previous
			return err
		}
		return nil
	}
	return errors.New("project was not found")
}
func (a *App) AssignTaskToProject(taskID, projectID string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if projectID != "" && !a.hasProject(projectID) {
		return errors.New("project was not found")
	}
	for i := range a.data.Tasks {
		if a.data.Tasks[i].ID != taskID {
			continue
		}
		previous := a.data.Tasks[i].ProjectID
		a.data.Tasks[i].ProjectID = projectID
		if err := a.save(); err != nil {
			a.data.Tasks[i].ProjectID = previous
			return err
		}
		return nil
	}
	return errors.New("task was not found")
}
func (a *App) DeleteProject(id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	projects := make([]Project, 0, len(a.data.Projects))
	found := false
	for _, project := range a.data.Projects {
		if project.ID == id {
			found = true
			continue
		}
		projects = append(projects, project)
	}
	if !found {
		return errors.New("project was not found")
	}
	previousProjects := a.data.Projects
	previousTasks := append([]Task(nil), a.data.Tasks...)
	a.data.Projects = projects
	for i := range a.data.Tasks {
		if a.data.Tasks[i].ProjectID == id {
			a.data.Tasks[i].ProjectID = ""
		}
	}
	if err := a.save(); err != nil {
		a.data.Projects = previousProjects
		a.data.Tasks = previousTasks
		return err
	}
	return nil
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
	event.Cancel()
	go a.enterWidget()
}

func (a *App) beforeWidgetClose(event *application.WindowEvent) {
	if a.quitting.Load() {
		return
	}
	event.Cancel()
	go a.RestoreFull()
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

func (a *App) GetUserName() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.userName
}

func (a *App) GetShouldPromptForName() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.shouldPromptForNameLocked()
}

func (a *App) shouldPromptForNameLocked() bool {
	return a.userName == ""
}

func (a *App) SetUserName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("please enter your name")
	}
	if len([]rune(name)) > 40 {
		return errors.New("name must be 40 characters or fewer")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.preferencesPath == "" {
		return errors.New("settings folder is unavailable")
	}
	previous := a.userName
	a.userName = name
	if err := a.savePreferencesLocked(); err != nil {
		a.userName = previous
		return err
	}
	return nil
}

func (a *App) savePreferencesLocked() error {
	if a.preferencesPath == "" {
		return errors.New("settings folder is unavailable")
	}
	raw, err := json.MarshalIndent(struct {
		TrayIconEnabled bool   `json:"trayIconEnabled"`
		UserName        string `json:"userName,omitempty"`
	}{a.trayIconEnabled, a.userName}, "", "  ")
	if err != nil {
		return err
	}
	return writePrivateFile(a.preferencesPath, raw)
}

func (a *App) SetTrayIconEnabled(enabled bool) error {
	a.mu.Lock()
	previous := a.trayIconEnabled
	a.trayIconEnabled = enabled
	if err := a.savePreferencesLocked(); err != nil {
		a.trayIconEnabled = previous
		a.mu.Unlock()
		return err
	}
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

func (a *App) GetAutoStartEnabled() (bool, error) {
	return getAutoStartEnabled()
}

func (a *App) SetAutoStartEnabled(enabled bool) error {
	return setAutoStartEnabled(enabled)
}
