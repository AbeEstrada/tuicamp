package main

import (
	"git.sr.ht/~rockorager/vaxis"
)

func (app *App) drawEditEntryWindow(win vaxis.Window) {
	dateStr := app.selectedDate.Format("Monday, January 2, 2006")
	win.Println(0, vaxis.Segment{
		Text:  dateStr,
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	})
	win.Println(2, vaxis.Segment{
		Text: app.entries[app.selectedEntry].Name,
	})
	return
}
