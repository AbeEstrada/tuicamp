package main

import (
	"fmt"
	"os"
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
	vx              *vaxis.Vaxis
	focusedWindow   int
	showQuitConfirm bool

	totalCols int
	totalRows int

	contentCols   int
	contentRows   int
	contentCursor int
	selectedEntry int

	calendarCols int
	calendarRows int
	currentMonth time.Time
	cursorDay    int
	selectedDay  int
	selectedDate time.Time

	timerCols   int
	timerRows   int
	elapsedTime time.Duration
	timerTicker *time.Ticker
	timerDone   chan struct{}

	apiToken  string
	apiClient *APIClient

	me      MeResponse
	timers  []TimersRunningResponse
	entries []EntryResponse
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
		vx:            vx,
		focusedWindow: Calendar,
		apiToken:      apiToken,
		currentMonth:  time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()),
		cursorDay:     now.Day(),
		selectedDay:   now.Day(),
		selectedDate:  now,
	}

	app.apiClient = NewAPIClient("https://app.timecamp.com/third_party/api")
	app.fetchMe()
	app.fetchEntries(app.selectedDate)
	app.fetchTimers()

	for ev := range vx.Events() {
		if app.HandleEvent(ev) {
			break // Exit the application
		}
		app.draw()
	}
	app.UpdateDimensions()
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

func (app *App) draw() {
	mainWin := app.vx.Window()
	mainWin.Clear()

	userWin := app.createStyledWindow(mainWin, 0, 0, app.totalCols, 3, app.focusedWindow == User)
	app.drawUserWindow(userWin)

	calendarWin := app.createStyledWindow(mainWin, 0, 3, app.calendarCols, app.calendarRows,
		app.focusedWindow == Calendar)
	app.drawCalendarWindow(calendarWin)

	timerWin := app.createStyledWindow(mainWin, app.calendarCols, 3,
		app.totalCols-app.calendarCols, app.calendarRows,
		app.focusedWindow == Timer)
	app.drawTimerWindow(timerWin)

	contentWin := app.createStyledWindow(mainWin, 0, app.calendarRows+3,
		app.totalCols, app.contentRows,
		app.focusedWindow == Content)
	app.drawContentWindow(contentWin)

	if app.showQuitConfirm {
		drawQuitConfirmation(mainWin)
	}

	app.vx.Render()
}

func drawQuitConfirmation(win vaxis.Window) {
	width, height := win.Size()
	dialogWidth := 40
	dialogHeight := 3
	dialogX := (width - dialogWidth) / 2
	dialogY := (height - dialogHeight) / 2
	dialogWin := win.New(dialogX, dialogY, dialogWidth, dialogHeight)
	dialogWin = border.All(dialogWin, vaxis.Style{
		Foreground: vaxis.IndexColor(4),
		Attribute:  vaxis.AttrBold,
	})
	dialogWin.Print(
		vaxis.Segment{
			Text: "Quit the application?",
			Style: vaxis.Style{
				Attribute: vaxis.AttrBold,
			},
		},
	)
}

func (app *App) UpdateDimensions() {
	cols, rows := app.vx.Window().Size()
	app.totalCols = cols
	app.totalRows = rows
	app.calendarCols = 22
	app.calendarRows = 10
	app.contentRows = rows - app.calendarRows
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
	}
	if key.Matches('q') {
		app.showQuitConfirm = true
		return false
	}
	if key.Matches('c', vaxis.ModCtrl) {
		return true // Exit application
	}
	if key.Matches(vaxis.KeyTab) {
		app.focusedWindow = (app.focusedWindow % 3) + 1
	}
	return false
}
