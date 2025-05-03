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
