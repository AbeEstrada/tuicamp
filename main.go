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
		switch ev := ev.(type) {
		case vaxis.Key:
			if app.handleKey(ev) {
				return
			}
		case vaxis.Resize:
			cols, rows := vx.Window().Size()
			app.calendarCols = 22
			app.calendarRows = 10
			app.contentRows = rows - app.calendarRows
			app.totalCols = cols
			app.totalRows = rows
		}
		app.draw()
	}
}

func (app *App) draw() {
	mainWin := app.vx.Window()
	mainWin.Clear()

	focusedStyle := vaxis.Style{
		Foreground: vaxis.IndexColor(4),
		Attribute:  vaxis.AttrBold,
	}
	unfocusedStyle := vaxis.Style{}

	userStyle, calendarStyle, timerStyle, contentStyle := unfocusedStyle, unfocusedStyle, unfocusedStyle, unfocusedStyle
	switch app.focusedWindow {
	case User:
		userStyle = unfocusedStyle
	case Calendar:
		calendarStyle = focusedStyle
	case Timer:
		timerStyle = focusedStyle
	case Content:
		contentStyle = focusedStyle
	}

	userWin := mainWin.New(0, 0, app.totalCols, 3)
	userWin = border.All(userWin, userStyle)
	app.drawUserWindow(userWin)

	calendarWin := mainWin.New(0, 3, app.calendarCols, app.calendarRows)
	calendarWin = border.All(calendarWin, calendarStyle)
	app.drawCalendarWindow(calendarWin)

	timerWin := mainWin.New(app.calendarCols, 3, app.totalCols-app.calendarCols, app.calendarRows)
	timerWin = border.All(timerWin, timerStyle)
	app.drawTimerWindow(timerWin)

	contentWin := mainWin.New(0, app.calendarRows+3, app.totalCols, app.contentRows)
	contentWin = border.All(contentWin, contentStyle)
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

func (app *App) handleKey(key vaxis.Key) bool {
	if !app.showQuitConfirm {
		if key.Matches('q') {
			app.showQuitConfirm = true
		} else if key.Matches('c', vaxis.ModCtrl) {
			return true
		}
	} else {
		if key.Matches('y') || key.Matches(vaxis.KeyEnter) {
			return true // Confirm quit
		} else if key.Matches('n') || key.Matches(vaxis.KeyEsc) || key.Matches('q') {
			app.showQuitConfirm = false
			return false
		}
	}

	if key.Matches(vaxis.KeyTab) && !app.showQuitConfirm {
		app.focusedWindow = (app.focusedWindow % 3) + 1
	}

	if app.focusedWindow == Calendar && !app.showQuitConfirm {
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
	} else if app.focusedWindow == Content && !app.showQuitConfirm {
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
	} else if app.focusedWindow == Timer && !app.showQuitConfirm {
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
	}

	return false
}
