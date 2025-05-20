package main

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

type EntryResponse struct {
	ID               int64  `json:"id"`
	Duration         string `json:"duration"`
	UserID           string `json:"user_id"`
	UserName         string `json:"user_name"`
	TaskID           string `json:"task_id"`
	TaskNote         string `json:"task_note"`
	LastModify       string `json:"last_modify"`
	Date             string `json:"date"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	Locked           string `json:"locked"`
	Name             string `json:"name"`
	AddonsExternalID string `json:"addons_external_id"`
	Billable         int    `json:"billable"`
	InvoiceID        string `json:"invoiceId"`
	Color            string `json:"color"`
	Description      string `json:"description"`
}

func (app *App) drawEntriesWindow(win vaxis.Window) {
	if app.showDeleteConfirm {
		app.drawConfirmationDialog(win, "Delete this entry? (y/n)", 1)
		return
	}
	if app.showEditEntry {
		app.drawEditEntryWindow(win)
		return
	}

	if app.entries == nil {
		win.Print(vaxis.Segment{
			Text:  "Loading entries...",
			Style: vaxis.Style{Attribute: vaxis.AttrItalic},
		})
		return
	}

	if app.selectedDay == 0 {
		win.Print(vaxis.Segment{
			Text:  "No date selected",
			Style: vaxis.Style{Attribute: vaxis.AttrItalic},
		})
		return
	}

	dateStr := app.selectedDate.Format("Monday, January 2, 2006")
	win.Println(0, vaxis.Segment{
		Text:  dateStr,
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	})

	scrollOffset := 0

	_, rows := app.vx.Window().Size()

	containsBillable := slices.ContainsFunc(app.entries, func(entry EntryResponse) bool {
		return entry.Billable > 0
	})
	visibleEntries := calculateVisibleEntries(app.entries, scrollOffset, rows)
	var totalDuration time.Duration
	for i, entry := range visibleEntries {
		row := i + 2 // +1 to account for title row
		hexValue, _ := strconv.ParseUint(entry.Color[1:], 16, 32)
		seconds, _ := strconv.ParseInt(entry.Duration, 10, 64)
		duration := "0s"
		if seconds == 0 && entry.StartTime == entry.EndTime {
			givenTime, _ := time.ParseInLocation("2006-01-02 15:04:05", entry.Date+" "+entry.StartTime, app.selectedDate.Location())
			elapsedTime := time.Since(givenTime)
			totalDuration += elapsedTime.Round(time.Second)
			duration = elapsedTime.Round(time.Second).String()
		} else {
			elapsedTime := time.Duration(seconds) * time.Second
			totalDuration += elapsedTime
			duration = elapsedTime.String()
		}
		selectedStyle := vaxis.Style{}
		if i+app.contentCursor == app.selectedEntry && app.focusedWindow == Content {
			selectedStyle = vaxis.Style{
				Attribute: vaxis.AttrReverse,
			}
		}
		endTime := " - " + entry.EndTime
		if entry.StartTime == entry.EndTime {
			endTime = ""
		}
		name := entry.Name
		if entry.Name != "" {
			name = " [" + name + "]"
		}
		if entry.Billable > 0 {
			name = " $" + name
		} else if containsBillable {
			name = "  " + name
		}
		win.Println(row,
			vaxis.Segment{
				Text: "● ",
				Style: vaxis.Style{
					Foreground: vaxis.HexColor(uint32(hexValue)),
					Attribute:  vaxis.AttrBold,
				},
			},
			vaxis.Segment{
				Text: fmt.Sprintf("%-10s", duration),
				Style: vaxis.Style{
					Attribute: selectedStyle.Attribute,
				},
			},
			vaxis.Segment{
				Text: entry.StartTime,
				Style: vaxis.Style{
					Attribute: selectedStyle.Attribute,
				},
			},
			vaxis.Segment{
				Text: endTime,
				Style: vaxis.Style{
					Attribute: selectedStyle.Attribute,
				},
			},
			vaxis.Segment{
				Text: name,
				Style: vaxis.Style{
					Attribute: selectedStyle.Attribute,
				},
			},
			vaxis.Segment{
				Text: " " + entry.Description,
			},
		)
	}
	if len(app.entries) > 0 {
		win.Println(len(app.entries)+3,
			vaxis.Segment{
				Text: "Total " + totalDuration.String(),
				Style: vaxis.Style{
					Attribute: vaxis.AttrBold,
				},
			},
		)
	}
}

func (app *App) handleContentKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		return false
	}
	if app.showDeleteConfirm {
		if key.Matches('y') || key.Matches(vaxis.KeyEnter) {
			app.showDeleteConfirm = false
			app.deleteEntry(app.entries[app.selectedEntry].ID)
			app.fetchEntries(app.selectedDate)
		} else if key.Matches('n') || key.Matches(vaxis.KeyEsc) {
			app.showDeleteConfirm = false
		}
	}
	if app.showEditEntry {
		app.handleEditEntryKeys(key)
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
	} else if key.Matches('d') {
		app.showDeleteConfirm = true
	} else if key.Matches('e') || key.Matches(vaxis.KeyEnter) {
		app.showEditEntry = true
	}
	return false
}

func (app *App) fetchEntries(date time.Time) error {
	var allEntries []EntryResponse
	year := date.Year()
	month := int(date.Month())
	day := date.Day()
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint: fmt.Sprintf("/entries?from=%d-%02d-%02d&to=%d-%02d-%02d", year, month, day, year, month, day),
		Method:   "GET",
		Response: &allEntries,
		Headers:  map[string]string{"Authorization": "Bearer " + app.apiToken},
	})
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed API response: %w", result.Error)
	}
	app.entries = allEntries
	app.selectedEntry = 0
	app.contentCursor = 0
	return nil
}

func (app *App) deleteEntry(ID int64) error {
	type Body struct {
		ID string `json:"id"`
	}
	type Response struct {
		Message string `json:"message"`
	}
	body := Body{
		ID: strconv.FormatInt(ID, 10),
	}
	var response Response
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint:    "/entries",
		Method:      "DELETE",
		RequestBody: &body,
		Response:    &response,
		Headers:     map[string]string{"Authorization": "Bearer " + app.apiToken},
	})
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed API response: %w", result.Error)
	}
	return nil
}

func calculateVisibleEntries(entries []EntryResponse, scrollOffset, maxVisible int) []EntryResponse {
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > len(entries)-maxVisible {
		scrollOffset = max(0, len(entries)-maxVisible)
	}
	end := min(scrollOffset+maxVisible, len(entries))
	return entries[scrollOffset:end]
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
