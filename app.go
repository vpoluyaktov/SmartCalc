package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"smartcalc/internal/calc"
	"smartcalc/internal/eval"
	"smartcalc/internal/updater"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

const maxRecentFiles = 10

// App struct
type App struct {
	ctx         context.Context
	recentFiles []string
	hasUnsaved  bool
	currentFile string
}

// NewApp creates a new App application struct
func NewApp() *App {
	app := &App{}
	app.loadRecentFiles()
	return app
}

// startup is called when the app starts
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

// beforeClose is called when the app is about to close
// Returns true to prevent closing (if user cancels), false to allow closing
func (a *App) beforeClose(ctx context.Context) (prevent bool) {
	if !a.hasUnsaved {
		return false // No unsaved changes, allow close
	}

	// If file has a name, silently save and close
	if a.currentFile != "" {
		runtime.EventsEmit(a.ctx, "menu:save")
		return false
	}

	// Only show dialog for unnamed/untitled documents
	result, err := runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:          runtime.QuestionDialog,
		Title:         "Unsaved Changes",
		Message:       "You have unsaved changes in an untitled document. Do you want to save before closing?",
		Buttons:       []string{"Save", "Don't Save", "Cancel"},
		DefaultButton: "Save",
		CancelButton:  "Cancel",
	})

	if err != nil {
		return false // On error, allow close
	}

	// Handle different button labels across platforms:
	// - Custom buttons: "Save", "Don't Save", "Cancel"
	// - macOS native: "Yes", "No", "Cancel" or button index
	// - Linux/GTK: "Yes", "No" or button text
	// - Windows: button text or "Yes", "No"
	switch result {
	case "Save", "Yes", "OK":
		// Emit saveAndQuit event - frontend will save and then quit
		runtime.EventsEmit(a.ctx, "app:saveAndQuit")
		return true // Prevent close - frontend will call Quit after saving
	case "Don't Save", "No":
		return false // Allow close without saving
	case "Cancel":
		return true // Prevent close
	}

	return false
}

// SetUnsavedState is called from frontend to update unsaved state
func (a *App) SetUnsavedState(hasUnsaved bool, currentFile string) {
	a.hasUnsaved = hasUnsaved
	a.currentFile = currentFile
}

// Quit closes the application
func (a *App) Quit() {
	runtime.Quit(a.ctx)
}

// getConfigPath returns the path to the config directory
func getConfigPath() string {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = os.TempDir()
	}
	return filepath.Join(configDir, "smartcalc")
}

// loadRecentFiles loads recent files from config
func (a *App) loadRecentFiles() {
	configPath := filepath.Join(getConfigPath(), "recent.json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		a.recentFiles = []string{}
		return
	}
	json.Unmarshal(data, &a.recentFiles)
}

// saveRecentFiles saves recent files to config
func (a *App) saveRecentFiles() {
	configDir := getConfigPath()
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "recent.json")
	data, _ := json.Marshal(a.recentFiles)
	os.WriteFile(configPath, data, 0644)
}

// GetRecentFiles returns the list of recent files
func (a *App) GetRecentFiles() []string {
	return a.recentFiles
}

// AddRecentFile adds a file to the recent files list
func (a *App) AddRecentFile(path string) {
	// Remove if already exists
	filtered := []string{}
	for _, f := range a.recentFiles {
		if f != path {
			filtered = append(filtered, f)
		}
	}
	// Add to front
	a.recentFiles = append([]string{path}, filtered...)
	// Limit size
	if len(a.recentFiles) > maxRecentFiles {
		a.recentFiles = a.recentFiles[:maxRecentFiles]
	}
	a.saveRecentFiles()
	// Also save as last file
	a.saveLastFile(path)
}

// GetLastFile returns the last opened file path
func (a *App) GetLastFile() string {
	configPath := filepath.Join(getConfigPath(), "lastfile.txt")
	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}
	path := strings.TrimSpace(string(data))
	// Check if file still exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return ""
	}
	return path
}

// saveLastFile saves the last opened file path
func (a *App) saveLastFile(path string) {
	configDir := getConfigPath()
	os.MkdirAll(configDir, 0755)
	configPath := filepath.Join(configDir, "lastfile.txt")
	os.WriteFile(configPath, []byte(path), 0644)
}

// AutoSave saves content to the current file (silent, no dialogs)
func (a *App) AutoSave(path, content string) error {
	if path == "" {
		return nil
	}
	return os.WriteFile(path, []byte(content), 0644)
}

// AdjustReferences adjusts line references when lines are added or removed
func (a *App) AdjustReferences(oldText, newText string) string {
	return eval.AdjustReferences(oldText, newText)
}

// EvalResult represents a single line evaluation result
type EvalResult struct {
	LineNum int    `json:"lineNum"`
	Input   string `json:"input"`
	Output  string `json:"output"`
}

// Evaluate evaluates all lines and returns results
// activeLineNum is 1-based line number of the line currently being edited (skip formatting for this line)
// Pass 0 or negative to format all lines
func (a *App) Evaluate(text string, activeLineNum int) []EvalResult {
	lines := strings.Split(text, "\n")
	results := calc.EvalLines(lines, activeLineNum)

	evalResults := make([]EvalResult, len(results))
	for i, r := range results {
		evalResults[i] = EvalResult{
			LineNum: i + 1,
			Input:   lines[i],
			Output:  r.Output,
		}
	}
	return evalResults
}

// EvaluateParallel evaluates all lines in parallel and returns results as they complete
// This is used for Ctrl+R to make the UI more responsive
func (a *App) EvaluateParallel(text string) []EvalResult {
	lines := strings.Split(text, "\n")
	results := calc.EvalLinesParallel(lines)

	evalResults := make([]EvalResult, len(results))
	for i, r := range results {
		evalResults[i] = EvalResult{
			LineNum: i + 1,
			Input:   lines[i],
			Output:  r.Output,
		}
	}
	return evalResults
}

// GetVersion returns the app version
func (a *App) GetVersion() string {
	return version
}

// OpenFileDialog opens a file dialog and returns the selected path
func (a *App) OpenFileDialog() (string, error) {
	return runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Open File",
		Filters: []runtime.FileFilter{
			{DisplayName: "SmartCalc Files", Pattern: "*.txt;*.sc"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
}

// SaveFileDialog opens a save dialog and returns the selected path
func (a *App) SaveFileDialog() (string, error) {
	return runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Save File",
		DefaultFilename: "untitled.txt",
		Filters: []runtime.FileFilter{
			{DisplayName: "Text Files", Pattern: "*.txt"},
			{DisplayName: "SmartCalc Files", Pattern: "*.sc"},
			{DisplayName: "All Files", Pattern: "*"},
		},
	})
}

// ReadFile reads a file and returns its contents
func (a *App) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// WriteFile writes content to a file
func (a *App) WriteFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0644)
}

// CopyWithResolvedRefs copies text with references replaced by values
func (a *App) CopyWithResolvedRefs(text string) string {
	return calc.ReplaceRefsWithValues(text)
}

// ShowInfoDialog shows an information dialog with the given title and message
func (a *App) ShowInfoDialog(title, message string) {
	runtime.MessageDialog(a.ctx, runtime.MessageDialogOptions{
		Type:    runtime.InfoDialog,
		Title:   title,
		Message: message,
	})
}

// StripLineResult removes the result from a line, keeping expression, '=' and inline comment
func (a *App) StripLineResult(line string) string {
	return calc.StripResult(line)
}

// HasLineResult checks if a line has a calculated result
func (a *App) HasLineResult(line string) bool {
	return calc.HasResult(line)
}

// FindDependentLines returns line numbers (1-based) that depend on the given line
func (a *App) FindDependentLines(text string, changedLine int) []int {
	lines := strings.Split(text, "\n")
	return calc.FindDependentLines(lines, changedLine)
}

// EvaluateLines evaluates specific lines and their dependents
// changedLine is the 1-based line number that was changed
// Returns results for all lines
func (a *App) EvaluateLines(text string, changedLine int) []EvalResult {
	lines := strings.Split(text, "\n")
	results := calc.EvalLines(lines, 0)

	evalResults := make([]EvalResult, len(results))
	for i, r := range results {
		evalResults[i] = EvalResult{
			LineNum: i + 1,
			Input:   lines[i],
			Output:  r.Output,
		}
	}
	return evalResults
}

// StripAllResults strips results from all expression lines and removes output lines
func (a *App) StripAllResults(text string) string {
	return calc.StripAllResults(text)
}

// StripAndEvalReferencingLines strips results from lines with references and re-evaluates them
func (a *App) StripAndEvalReferencingLines(text string) string {
	return calc.StripAndEvalReferencingLines(text)
}

// GetGitHubRepoURL returns the GitHub repository URL
func (a *App) GetGitHubRepoURL() string {
	return updater.GitHubRepoURL
}

// CheckForUpdates checks if there's a newer version available on GitHub
func (a *App) CheckForUpdates() (*updater.ReleaseInfo, error) {
	return updater.CheckForUpdates(version)
}

// OpenURL opens a URL in the default browser
func (a *App) OpenURL(url string) {
	runtime.BrowserOpenURL(a.ctx, url)
}
