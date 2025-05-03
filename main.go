package main

import (
	"fmt"
	"os"
	"time"

	"git.sr.ht/~rockorager/vaxis"
	"git.sr.ht/~rockorager/vaxis/widgets/border"
)

const ( // Windows
	Calendar = 0 // Top Left
	Timer    = 1 // Top Right
	Content  = 2 // Bottom
)

type App struct {
	vx              *vaxis.Vaxis
	focusedWindow   int
	showQuitConfirm bool

	apiToken  string
	apiClient *APIClient

	calendarCols int
	calendarRows int
	timerCols    int
	timerRows    int
	contentCols  int
	contentRows  int
	totalCols    int
	totalRows    int

	currentMonth time.Time
	cursorDay    int
	selectedDay  int
	selectedDate time.Time

	elapsedTime time.Duration
	timerTicker *time.Ticker
	timerDone   chan struct{}

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
			// app.contentCols = cols
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

	calendarStyle, timerStyle, contentStyle := unfocusedStyle, unfocusedStyle, unfocusedStyle
	switch app.focusedWindow {
	case Calendar:
		calendarStyle = focusedStyle
	case Timer:
		timerStyle = focusedStyle
	case Content:
		contentStyle = focusedStyle
	}

	calendarWin := mainWin.New(0, 0, app.calendarCols, app.calendarRows)
	calendarWin = border.All(calendarWin, calendarStyle)
	app.drawCalendarWindow(calendarWin)

	timerWin := mainWin.New(app.calendarCols, 0, app.totalCols-app.calendarCols, app.calendarRows)
	timerWin = border.All(timerWin, timerStyle)
	app.drawTimerWindow(timerWin)

	contentWin := mainWin.New(0, app.calendarRows, app.totalCols, app.contentRows)
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
		app.focusedWindow = (app.focusedWindow + 1) % 3
	}

	if app.focusedWindow == Calendar && !app.showQuitConfirm {
		year, month, _ := app.currentMonth.Date()
		daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day()
		// firstDay := time.Date(year, month, 1, 0, 0, 0, 0, app.currentMonth.Location())
		// firstDayOfWeek := int(firstDay.Weekday())
		// currentDayOfWeek := (firstDayOfWeek + app.cursorDay - 1) % 7
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
			} /* else {
				// Go to previous month
				prevMonth := time.Date(year, month-1, 1, 0, 0, 0, 0, app.currentMonth.Location())
				daysInPrevMonth := time.Date(prevMonth.Year(), prevMonth.Month()+1, 0, 0, 0, 0, 0, prevMonth.Location()).Day()
				// Find all days in previous month that fall on the same weekday
				var daysOnSameWeekday []int
				for day := 1; day <= daysInPrevMonth; day++ {
					date := time.Date(prevMonth.Year(), prevMonth.Month(), day, 0, 0, 0, 0, prevMonth.Location())
					if int(date.Weekday()) == currentDayOfWeek {
						daysOnSameWeekday = append(daysOnSameWeekday, day)
					}
				}
				// Find the last occurrence of the weekday in the previous month
				newDay := daysOnSameWeekday[len(daysOnSameWeekday)-1]
				app.currentMonth = prevMonth
				app.cursorDay = newDay
			}*/
		} else if key.Matches('j') || key.Matches(vaxis.KeyDown) {
			// Move down a week
			if app.cursorDay+7 <= daysInMonth {
				app.cursorDay += 7
			} /* else {
				// Go to next month
				nextMonth := time.Date(year, month+1, 1, 0, 0, 0, 0, app.currentMonth.Location())
				daysInNextMonth := time.Date(nextMonth.Year(), nextMonth.Month()+1, 0, 0, 0, 0, 0, nextMonth.Location()).Day()
				// Find all days in next month that fall on the same weekday
				var daysOnSameWeekday []int
				for day := 1; day <= daysInNextMonth; day++ {
					date := time.Date(nextMonth.Year(), nextMonth.Month(), day, 0, 0, 0, 0, nextMonth.Location())
					if int(date.Weekday()) == currentDayOfWeek {
						daysOnSameWeekday = append(daysOnSameWeekday, day)
					}
				}
				// Find the first occurrence of the weekday in the next month
				newDay := daysOnSameWeekday[0]
				app.currentMonth = nextMonth
				app.cursorDay = newDay
			}*/
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
		} /* else if key.Matches('j') || key.Matches(vaxis.KeyDown) {
			if len(app.entries) > 0 {
				for i, item := range app.entries {
					if item.Selected && i < len(app.entries)-1 {
						app.entries[i].Selected = false
						app.entries[i+1].Selected = true
						break
					}
				}
			}
		} else if key.Matches('k') || key.Matches(vaxis.KeyUp) {
			if len(app.entries) > 0 {
				for i, item := range app.entries {
					if item.Selected && i > 0 {
						app.entries[i].Selected = false
						app.entries[i-1].Selected = true
						break
					}
				}
			}
		}*/
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
