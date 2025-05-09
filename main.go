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
	case User:
		return app.handleUserKeys()
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

func (app *App) handleCalendarKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		return false
	}
	year, month, _ := app.currentMonth.Date()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day()

	if key.Matches('L') {
		app.focusedWindow = Timer
	} else if key.Matches('J') {
		app.focusedWindow = Content
	} else if key.Matches('h') || key.Matches(vaxis.KeyLeft) {
		if app.cursorDay > 1 {
			app.cursorDay--
		}
	} else if key.Matches('l') || key.Matches(vaxis.KeyRight) {
		if app.cursorDay < daysInMonth {
			app.cursorDay++
		}
	} else if key.Matches('k') || key.Matches(vaxis.KeyUp) {
		// Move up a week
		if app.cursorDay > 7 {
			app.cursorDay -= 7
		}
	} else if key.Matches('j') || key.Matches(vaxis.KeyDown) {
		// Move down a week
		if app.cursorDay+7 <= daysInMonth {
			app.cursorDay += 7
		}
	} else if key.Matches('g') || key.Matches(vaxis.KeyHome) {
		// First day of month
		app.cursorDay = 1
	} else if key.Matches('G') || key.Matches(vaxis.KeyEnd) {
		// Last day of month
		app.cursorDay = daysInMonth
	} else if key.Matches('p') || key.Matches(vaxis.KeyPgUp) {
		// Previous month
		app.currentMonth = time.Date(year, month-1, 1, 0, 0, 0, 0, app.currentMonth.Location())
		if app.cursorDay > time.Date(app.currentMonth.Year(), app.currentMonth.Month()+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day() {
			app.cursorDay = time.Date(app.currentMonth.Year(), app.currentMonth.Month()+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day()
		}
	} else if key.Matches('n') || key.Matches(vaxis.KeyPgDown) {
		// Next month
		app.currentMonth = time.Date(year, month+1, 1, 0, 0, 0, 0, app.currentMonth.Location())
		if app.cursorDay > time.Date(app.currentMonth.Year(), app.currentMonth.Month()+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day() {
			app.cursorDay = time.Date(app.currentMonth.Year(), app.currentMonth.Month()+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day()
		}
	} else if key.Matches('t') {
		// Today
		now := time.Now()
		app.currentMonth = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		app.cursorDay = now.Day()
	} else if key.Matches(vaxis.KeyEnter) || key.Matches(vaxis.KeySpace) {
		app.selectedDay = app.cursorDay
		app.selectedDate = time.Date(year, month, app.selectedDay, 0, 0, 0, 0, app.currentMonth.Location())
		app.fetchEntries(app.selectedDate)
		app.fetchTimers()
	}
	return false
}

func (app *App) handleContentKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		return false
	}
	if key.Matches('K') {
		app.focusedWindow = Calendar
	} else if key.Matches('j') || key.Matches(vaxis.KeyDown) {
		if app.selectedEntry < len(app.entries)-1 {
			app.selectedEntry++
			visibleRows := app.contentRows - 3 // Account for header and footer
			if app.selectedEntry >= app.contentCursor+visibleRows {
				app.contentCursor = app.selectedEntry - visibleRows + 1
			}
		}
	} else if key.Matches('k') || key.Matches(vaxis.KeyUp) {
		if app.selectedEntry > 0 {
			app.selectedEntry--
			if app.selectedEntry < app.contentCursor {
				app.contentCursor = app.selectedEntry
			}
		}
	}
	return false
}

func (app *App) handleTimerKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		return false
	}
	if key.Matches('H') {
		app.focusedWindow = Calendar
	} else if key.Matches('J') {
		app.focusedWindow = Content
	} else if key.Matches(vaxis.KeyEnter) || key.Matches(vaxis.KeySpace) {
		if len(app.timers) > 0 {
			app.stopTimers()
		} else {
			app.startTimer()
		}
	}
	return false
}

func (app *App) handleUserKeys() bool {
	return false
}
