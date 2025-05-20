package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/border"
)

const ( // Windows
	User     = 0 // Top
	Calendar = 1 // Middle Left
	Timer    = 2 // Middle Right
	Content  = 3 // Bottom
)

type App struct {
	vx                *vaxis.Vaxis
	focusedWindow     int
	showQuitConfirm   bool
	showDeleteConfirm bool
	showEditEntry     bool

	userRows int

	contentCols   int
	contentRows   int
	contentCursor int
	selectedEntry int

	selectedTask    int
	drawnTasks      int
	taskHierarchy   *TaskHierarchy
	taskSearchMode  bool
	taskSearchInput string

	calendarCols int
	calendarRows int
	currentMonth time.Time
	cursorDay    int
	selectedDay  int
	selectedDate time.Time

	elapsedTime time.Duration
	timerTicker *time.Ticker
	timerDone   chan struct{}

	apiToken  string
	apiClient *APIClient

	me      MeResponse
	timers  []TimersRunningResponse
	entries []EntryResponse
	tasks   map[string]TaskResponse
}

func main() {
	apiToken := os.Getenv("TIMECAMP_API_TOKEN")
	if apiToken == "" {
		fmt.Fprintf(os.Stderr, "error: TIMECAMP_API_TOKEN environment variable is not set\n")
		os.Exit(1)
	}

	vx, err := vaxis.New(vaxis.Options{})
	if err != nil {
		panic(err)
	}
	defer vx.Close()

	now := time.Now()
	app := &App{
		vx:              vx,
		focusedWindow:   Calendar,
		apiToken:        apiToken,
		currentMonth:    time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		cursorDay:       now.Day(),
		selectedDay:     now.Day(),
		selectedDate:    now,
		apiClient:       NewAPIClient("https://app.timecamp.com/third_party/api"),
		taskSearchMode:  false,
		taskSearchInput: "",
		selectedTask:    -1,
	}

	app.UpdateDimensions()
	app.Draw()
	vx.Render()

	go app.fetchInitialData()

	for ev := range vx.Events() {
		if app.HandleEvent(ev) {
			break // Exit the application
		}
		app.Draw()
	}
}

type fetchError struct {
	fetchType string
	err       error
}

func (app *App) fetchInitialData() {
	var wg sync.WaitGroup
	errChan := make(chan fetchError, 4)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.fetchMe(); err != nil {
			errChan <- fetchError{"user info", err}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.fetchEntries(app.selectedDate); err != nil {
			errChan <- fetchError{"entries", err}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.fetchTimers(); err != nil {
			errChan <- fetchError{"timers", err}
		}
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := app.fetchTasks(); err != nil {
			errChan <- fetchError{"tasks", err}
		}
	}()
	go func() {
		wg.Wait()
		close(errChan)
		app.vx.PostEvent(vaxis.Redraw{})
	}()
	go func() {
		for fetchErr := range errChan {
			fmt.Fprintf(os.Stderr, "error fetching %s: %v\n", fetchErr.fetchType, fetchErr.err)
		}
	}()
}

func (app *App) UpdateDimensions() {
	_, rows := app.vx.Window().Size()
	app.userRows = 3
	app.calendarCols = 22
	app.calendarRows = 10
	app.contentRows = rows - app.calendarRows
}

func (app *App) createStyledWindow(parent vaxis.Window, x, y, width, height int, isFocused bool) vaxis.Window {
	win := parent.New(x, y, width, height)
	style := vaxis.Style{}
	if isFocused {
		style = vaxis.Style{
			Foreground: vaxis.IndexColor(4),
			Attribute:  vaxis.AttrBold,
		}
	}
	return border.All(win, style)
}

func (app *App) Draw() {
	mainWin := app.vx.Window()
	mainWin.Clear()

	cols, _ := app.vx.Window().Size()

	userWin := app.createStyledWindow(mainWin, 0, 0, cols, app.userRows, app.focusedWindow == User)
	app.drawUserWindow(userWin)

	calendarWin := app.createStyledWindow(mainWin, 0, app.userRows, app.calendarCols, app.calendarRows, app.focusedWindow == Calendar)
	app.drawCalendarWindow(calendarWin)

	timerWin := app.createStyledWindow(mainWin, app.calendarCols, app.userRows, cols-app.calendarCols, app.calendarRows, app.focusedWindow == Timer)
	app.drawTimerWindow(timerWin)

	contentWin := app.createStyledWindow(mainWin, 0, app.calendarRows+app.userRows, cols, app.contentRows, app.focusedWindow == Content)
	app.drawEntriesWindow(contentWin)

	if app.showQuitConfirm {
		app.drawConfirmationDialog(mainWin, "Quit the application?", 4)
	}

	app.vx.Render()
}

func (app *App) drawConfirmationDialog(win vaxis.Window, message string, highlightColor uint8) {
	width, height := win.Size()
	dialogWidth := 40
	dialogHeight := 3
	dialogX := (width - dialogWidth) / 2
	dialogY := (height - dialogHeight) / 2
	dialogWin := win.New(dialogX, dialogY, dialogWidth, dialogHeight)
	dialogWin = border.All(dialogWin, vaxis.Style{
		Foreground: vaxis.IndexColor(highlightColor),
		Attribute:  vaxis.AttrBold,
	})
	dialogWin.Print(
		vaxis.Segment{
			Text: message,
			Style: vaxis.Style{
				Attribute: vaxis.AttrBold,
			},
		},
	)
}

func (app *App) HandleEvent(ev vaxis.Event) bool {
	switch ev := ev.(type) {
	case vaxis.Key:
		return app.HandleKeyEvent(ev)
	case vaxis.Resize:
		app.UpdateDimensions()
	}
	return false
}

func (app *App) HandleKeyEvent(key vaxis.Key) bool {
	if app.handleGlobalKeys(key) {
		return true
	}
	switch app.focusedWindow {
	case Calendar:
		return app.handleCalendarKeys(key)
	case Content:
		return app.handleContentKeys(key)
	case Timer:
		return app.handleTimerKeys(key)
	}
	return false
}

func (app *App) handleGlobalKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		if key.Matches('y') || key.Matches(vaxis.KeyEnter) {
			return true // Confirm quit
		} else if key.Matches('n') || key.Matches(vaxis.KeyEsc) || key.Matches('q') {
			app.showQuitConfirm = false
		}
		return false
	} else if !app.showDeleteConfirm && !app.showEditEntry {
		if key.Matches('q') {
			app.showQuitConfirm = true
			return false
		}
		if key.Matches(vaxis.KeyTab) {
			app.focusedWindow = (app.focusedWindow % 3) + 1
		}

	}
	if key.Matches('c', vaxis.ModCtrl) {
		return true // Exit application
	}
	return false
}
