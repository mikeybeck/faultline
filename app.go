package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/mikey/faultline/internal/applog"
	"github.com/mikey/faultline/internal/config"
	"github.com/mikey/faultline/internal/detect"
	"github.com/mikey/faultline/internal/editor"
	"github.com/mikey/faultline/internal/engine"
	"github.com/mikey/faultline/internal/event"
	"github.com/mikey/faultline/internal/ingest"
	"github.com/mikey/faultline/internal/mark"
	"github.com/mikey/faultline/internal/persist"
	"github.com/mikey/faultline/internal/source"
	"github.com/mikey/faultline/internal/statefile"
)

// App is the Wails backend for the desktop UI.
type App struct {
	ctx           context.Context
	eng           *engine.Engine
	flagConfig    string
	flagFromStart bool
	projectDir    string
	configPath    string
	fromStart     bool
}

func NewApp(configPath string, fromStart bool) *App {
	eng := engine.New()
	eng.UseMarks(mark.Default())
	if db, err := persist.Default(); err == nil {
		eng.UsePersist(db)
	} else {
		eng.ReportInternal("inbox database: "+err.Error(), "")
	}
	return &App{
		eng:           eng,
		flagConfig:    configPath,
		flagFromStart: fromStart,
		fromStart:     fromStart,
	}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	go a.relay()

	if a.flagConfig != "" {
		cfg, err := config.Load(a.flagConfig)
		if err != nil {
			runtime.LogError(ctx, err.Error())
			a.eng.ReportInternal("failed to load "+a.flagConfig+": "+err.Error(), "")
			return
		}
		a.configPath = a.flagConfig
		a.projectDir = filepath.Dir(a.flagConfig)
		a.eng.SetProjectDir(a.projectDir)
		if err := a.eng.Start(ctx, cfg, a.fromStart); err != nil {
			runtime.LogError(ctx, err.Error())
			a.eng.ReportInternal("failed to start watchers: "+err.Error(), "")
		}
		_ = statefile.Remember(a.projectDir, a.configPath)
		return
	}

	st, err := statefile.Load()
	if err != nil || st.ConfigPath == "" {
		return
	}
	if _, err := os.Stat(st.ConfigPath); err != nil {
		return
	}
	cfg, err := config.Load(st.ConfigPath)
	if err != nil {
		a.eng.ReportInternal("failed to load "+st.ConfigPath+": "+err.Error(), "")
		return
	}
	a.projectDir = st.ProjectDir
	a.configPath = st.ConfigPath
	a.eng.SetProjectDir(a.projectDir)
	if err := a.eng.Start(ctx, cfg, a.fromStart); err != nil {
		runtime.LogError(ctx, err.Error())
		a.eng.ReportInternal("failed to start watchers: "+err.Error(), "")
	}
	_ = statefile.Remember(a.projectDir, a.configPath)
}

func (a *App) shutdown(ctx context.Context) {
	a.eng.Stop()
	a.eng.ClosePersist()
}

func (a *App) relay() {
	for {
		select {
		case <-a.ctx.Done():
			return
		case _, ok := <-a.eng.Events():
			if !ok {
				return
			}
			runtime.EventsEmit(a.ctx, "inbox:changed")
		case st, ok := <-a.eng.Statuses():
			if !ok {
				return
			}
			runtime.EventsEmit(a.ctx, "status", st)
		}
	}
}

// Bootstrap is the initial UI payload.
type Bootstrap struct {
	Screen     string              `json:"screen"`
	ProjectDir string              `json:"projectDir"`
	ConfigPath string              `json:"configPath"`
	Config     *config.Config      `json:"config"`
	FromStart  bool                `json:"fromStart"`
	Running    bool                `json:"running"`
	Recent     []statefile.Project `json:"recent"`
	Editors    []string            `json:"editors"`
	IngestAddr string              `json:"ingestAddr"`
}

func (a *App) Bootstrap() Bootstrap {
	cfg := a.eng.Config()
	screen := "welcome"
	if a.eng.Running() && cfg != nil {
		screen = "inbox"
	}
	return Bootstrap{
		Screen:     screen,
		ProjectDir: a.projectDir,
		ConfigPath: a.configPath,
		Config:     cfg,
		FromStart:  a.fromStart,
		Running:    a.eng.Running(),
		Recent:     statefile.RecentProjects(),
		Editors:    editor.Installed(),
		IngestAddr: a.eng.BoundIngest(),
	}
}

func (a *App) DefaultConfig() config.Config {
	return config.Config{
		Notifications: config.DefaultEnabledNotifications(),
		Editor:        config.EditorConfig{Command: editor.PreferredCommand()},
	}
}

// AppState is a snapshot of the inbox.
type AppState struct {
	Events       []EventDTO      `json:"events"`
	Sources      []source.Status `json:"sources"`
	Running      bool            `json:"running"`
	Total        int             `json:"total"`
	Marked       bool            `json:"marked"`
	IngestAddr   string          `json:"ingestAddr"`
	SourceCounts map[string]int  `json:"sourceCounts"`
}

func (a *App) GetState(filter, sort, severity, source string) AppState {
	st := AppState{
		Sources:      a.eng.SnapshotStatuses(),
		Running:      a.eng.Running(),
		IngestAddr:   a.eng.BoundIngest(),
		SourceCounts: a.eng.Store().CountsBySource(),
	}
	var items []event.Event
	if sort == "frequency" {
		items = a.eng.Store().ByFrequency(filter)
	} else {
		items = a.eng.Store().Summaries(filter)
	}
	if sort == "frequency" {
		for i := range items {
			items[i].Stack = ""
			items[i].Raw = ""
		}
	}
	sev := strings.ToLower(strings.TrimSpace(severity))
	src := strings.TrimSpace(source)
	st.Events = make([]EventDTO, 0, len(items))
	for _, ev := range items {
		if sev != "" && sev != "all" {
			want := map[string]bool{}
			for _, part := range strings.Split(sev, ",") {
				part = strings.TrimSpace(part)
				if part != "" {
					want[part] = true
				}
			}
			if len(want) > 0 && !want[string(ev.Severity)] {
				continue
			}
		}
		if src != "" && ev.Source != src {
			continue
		}
		st.Events = append(st.Events, toSummaryDTO(ev))
	}
	st.Total = len(st.Events)
	st.Marked = a.eng.HasMarks()
	return st
}

func (a *App) GetEvent(hash string) (EventDTO, error) {
	ev, ok := a.eng.Store().Get(hash)
	if !ok {
		return EventDTO{}, fmt.Errorf("unknown event")
	}
	return toDTO(ev, a.projectDir), nil
}

func (a *App) Clear() {
	a.eng.Clear()
}

// ClearMark forgets file resume points and re-reads logs. Browser events are kept.
func (a *App) ClearMark() {
	cfg := a.eng.Config()
	running := a.eng.Running()
	a.eng.ClearMarks()
	if running && cfg != nil {
		_ = a.eng.Restart(a.ctx, cfg, a.fromStart)
	}
}

func (a *App) Dismiss(hashes []string) {
	a.eng.Dismiss(hashes)
}

func (a *App) DismissMatching(filter, severity, source, typ string) {
	a.eng.DismissMatching(filter, severity, source, typ)
}

func (a *App) Mute(kind, value string) error {
	return a.eng.Mute(kind, value)
}

func (a *App) Unmute(kind, value string) error {
	return a.eng.Unmute(kind, value)
}

func (a *App) Mutes() []persist.Mute {
	m := a.eng.Mutes()
	if m == nil {
		return []persist.Mute{}
	}
	return m
}

func (a *App) OpenInEditor(hash string) error {
	ev, ok := a.eng.Store().Get(hash)
	if !ok {
		return fmt.Errorf("unknown event")
	}
	return a.eng.Open(ev.File, ev.Line)
}

func (a *App) OpenPath(file string, line int) error {
	file = strings.TrimSpace(file)
	if file == "" {
		return fmt.Errorf("no file associated with this location")
	}
	if a.projectDir != "" {
		file = ingest.ResolveFile(file, a.projectDir)
	}
	return a.eng.Open(file, line)
}

func (a *App) RecentProjects() []statefile.Project {
	return statefile.RecentProjects()
}

func (a *App) OpenRecent(configPath string) error {
	configPath = strings.TrimSpace(configPath)
	if configPath == "" {
		return fmt.Errorf("no project selected")
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}
	dir := filepath.Dir(configPath)
	a.projectDir = dir
	a.configPath = configPath
	a.eng.SetProjectDir(dir)
	if err := a.eng.Restart(a.ctx, cfg, a.fromStart); err != nil {
		a.eng.ReportInternal("failed to open project: "+err.Error(), "")
		return err
	}
	return statefile.Remember(dir, configPath)
}

func (a *App) DetectEditors() []string {
	return editor.Installed()
}

func (a *App) PickProjectDir() (string, error) {
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open project folder",
	})
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (a *App) PickLogFile() (string, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "Add log file",
		Filters: []runtime.FileFilter{{DisplayName: "Log files (*.log)", Pattern: "*.log;*.txt;*.log.txt"}},
	})
	if err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) PickEditor() (string, error) {
	opts := runtime.OpenDialogOptions{
		Title: "Choose editor executable",
		Filters: []runtime.FileFilter{
			{DisplayName: "Executables", Pattern: "*.exe;*.cmd;*.bat;*"},
			{DisplayName: "All files", Pattern: "*.*"},
		},
	}
	if os.Getenv("OS") == "Windows_NT" {
		if local := os.Getenv("LOCALAPPDATA"); local != "" {
			scripts := filepath.Join(local, "JetBrains", "Toolbox", "scripts")
			if st, err := os.Stat(scripts); err == nil && st.IsDir() {
				opts.DefaultDirectory = scripts
			}
		}
		if opts.DefaultDirectory == "" {
			if pf := os.Getenv("ProgramFiles"); pf != "" {
				jetbrains := filepath.Join(pf, "JetBrains")
				if st, err := os.Stat(jetbrains); err == nil && st.IsDir() {
					opts.DefaultDirectory = jetbrains
				}
			}
		}
	}
	return runtime.OpenFileDialog(a.ctx, opts)
}

func (a *App) DetectSources(dir string) ([]detect.Candidate, error) {
	if dir == "" {
		return nil, fmt.Errorf("no folder selected")
	}
	return detect.ScanDir(dir)
}

func (a *App) CandidateFromPath(path string) detect.Candidate {
	path = strings.TrimSpace(path)
	name := strings.TrimSuffix(filepath.Base(path), filepath.Ext(filepath.Base(path)))
	if name == "" {
		name = "log"
	}
	typ := detect.TypeGeneric
	if _, err := os.Stat(path); err == nil {
		typ = detect.Sniff(path)
	}
	return detect.Candidate{Name: name, Path: path, Type: typ}
}

func (a *App) SaveAndWatch(projectDir string, cfg config.Config, fromStart bool) error {
	if projectDir == "" {
		return fmt.Errorf("choose a project folder so settings can be saved")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	path := filepath.Join(projectDir, "faultline.yaml")
	if err := config.Save(path, &cfg); err != nil {
		return err
	}
	a.projectDir = projectDir
	a.configPath = path
	a.fromStart = fromStart
	a.eng.SetProjectDir(projectDir)
	if err := a.eng.Restart(a.ctx, &cfg, fromStart); err != nil {
		a.eng.ReportInternal("failed to start watchers: "+err.Error(), "")
		return err
	}
	return statefile.Remember(projectDir, path)
}

func (a *App) ApplyAndWatch(cfg config.Config, fromStart bool) error {
	if a.projectDir == "" {
		return fmt.Errorf("no project folder — save from the welcome screen first")
	}
	return a.SaveAndWatch(a.projectDir, cfg, fromStart)
}

// SaveProjectConfig writes faultline.yaml without restarting watchers.
func (a *App) SaveProjectConfig(cfg config.Config) error {
	if a.projectDir == "" {
		return fmt.Errorf("no project folder")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	path := filepath.Join(a.projectDir, "faultline.yaml")
	if err := config.Save(path, &cfg); err != nil {
		return err
	}
	a.configPath = path
	return nil
}

func (a *App) NewProject() {
	a.eng.Stop()
	a.eng.ResetInbox()
	a.projectDir = ""
	a.configPath = ""
}

// ReportError records a Faultline UI or backend error in the log file and inbox.
func (a *App) ReportError(message, stack string) {
	a.eng.ReportInternal(message, stack)
}

func (a *App) FaultlineLogPath() string {
	p, err := applog.Path()
	if err != nil {
		return ""
	}
	return p
}

func (a *App) OpenFaultlineLog() error {
	p, err := applog.Path()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	_ = f.Close()
	return a.eng.Open(p, 1)
}

// EventDTO is the JSON shape sent to the frontend.
type EventDTO struct {
	Source    string         `json:"source"`
	Time      string         `json:"time"`
	Type      string         `json:"type"`
	Message   string         `json:"message"`
	File      string         `json:"file"`
	Line      int            `json:"line"`
	Severity  string         `json:"severity"`
	Stack     string         `json:"stack"`
	Raw       string         `json:"raw"`
	Hash      string         `json:"hash"`
	Count     int            `json:"count"`
	FirstSeen string         `json:"firstSeen"`
	LastSeen  string         `json:"lastSeen"`
	Title     string         `json:"title"`
	Location  string         `json:"location"`
	Frames    []ingest.Frame `json:"frames"`
}

func toDTO(ev event.Event, projectDir string) EventDTO {
	d := EventDTO{
		Source:    ev.Source,
		Time:      formatTime(ev.Time),
		Type:      ev.Type,
		Message:   ev.Message,
		File:      ev.File,
		Line:      ev.Line,
		Severity:  string(ev.Severity),
		Stack:     ev.Stack,
		Raw:       ev.Raw,
		Hash:      ev.Hash,
		Count:     ev.Count,
		FirstSeen: formatTime(ev.FirstSeen),
		LastSeen:  formatTime(ev.LastSeen),
		Title:     ev.Title(),
		Location:  ev.Location(),
	}
	if ev.Stack != "" {
		d.Frames = ingest.ParseFrames(ev.Stack, projectDir)
	}
	if d.Frames == nil {
		d.Frames = []ingest.Frame{}
	}
	return d
}

func toSummaryDTO(ev event.Event) EventDTO {
	d := toDTO(ev, "")
	d.Stack = ""
	d.Raw = ""
	d.Frames = []ingest.Frame{}
	return d
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(time.RFC3339)
}
