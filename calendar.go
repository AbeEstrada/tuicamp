package main

import (
	"fmt"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

func (app *App) drawCalendarWindow(win vaxis.Window) {
	monthTitle := fmt.Sprintf("%s %d", app.currentMonth.Month().String(), app.currentMonth.Year())
	win.Println(0, vaxis.Segment{
		Text:  monthTitle,
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	})

	daysOfWeek := make([]string, 7)
	for i := range daysOfWeek {
		day := time.Sunday + time.Weekday(i)
		daysOfWeek[i] = day.String()[:2]
	}
	daySegments := make([]vaxis.Segment, 0, len(daysOfWeek)*2-1)
	for i, day := range daysOfWeek {
		daySegments = append(daySegments, vaxis.Segment{
			Text:  day,
			Style: vaxis.Style{UnderlineStyle: vaxis.UnderlineSingle},
		})
		if i < len(daysOfWeek)-1 {
			daySegments = append(daySegments, vaxis.Segment{
				Text: " ",
			})
		}
	}
	win.Println(2, daySegments...)

	firstDay := app.currentMonth
	firstDayOfWeek := int(firstDay.Weekday())

	year, month, _ := app.currentMonth.Date()
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, app.currentMonth.Location()).Day()

	selectedInCurrentMonth := false
	if !app.selectedDate.IsZero() {
		selectedYear, selectedMonth, _ := app.selectedDate.Date()
		if selectedYear == year && selectedMonth == month {
			selectedInCurrentMonth = true
		}
	}

	dayNum := 1
	row := 3 // Start on row 2 (after the header)

	for weekRow := 0; weekRow < 6 && dayNum <= daysInMonth; weekRow++ {
		segments := make([]vaxis.Segment, 0, 20)
		for weekDay := range 7 {
			if weekRow == 0 && weekDay < firstDayOfWeek {
				segments = append(segments, vaxis.Segment{
					Text: "  ",
				})
			} else if dayNum <= daysInMonth {
				isCursor := dayNum == app.cursorDay
				isSelected := selectedInCurrentMonth && dayNum == app.selectedDay
				now := time.Now()
				isToday := now.Year() == year &&
					now.Month() == month &&
					now.Day() == dayNum
				style := vaxis.Style{}
				if isCursor && app.focusedWindow == Calendar {
					style.Attribute = vaxis.AttrReverse
				} else if isSelected {
					style.Attribute = vaxis.AttrBold
					style.Foreground = vaxis.IndexColor(4) // Blue for selected day
				} else if isToday {
					style.Attribute = vaxis.AttrBold
					// style.Foreground = vaxis.IndexColor(15)

				}
				dayText := fmt.Sprintf("%2d", dayNum)
				segments = append(segments, vaxis.Segment{
					Text:  dayText,
					Style: style,
				})
				dayNum++
			} else {
				segments = append(segments, vaxis.Segment{
					Text: "  ",
				})
			}
			if weekDay < 6 {
				segments = append(segments, vaxis.Segment{
					Text: " ",
				})
			}
		}
		win.Println(row+weekRow, segments...)
	}
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
		app.selectedTask = -1
		app.selectedDay = app.cursorDay
		app.selectedDate = time.Date(year, month, app.selectedDay, 0, 0, 0, 0, app.currentMonth.Location())
		app.fetchEntries(app.selectedDate)
		app.fetchTimers()
	}
	return false
}
