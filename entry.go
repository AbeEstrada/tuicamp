package main

import (
	"fmt"
	"strconv"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

const (
	EntryCursorStart = iota
	EntryCursorEnd
	EntryCursorTask
)

func (app *App) isEntryTimer(entry EntryResponse) bool {
	return entry.StartTime == entry.EndTime && len(app.timers) > 0
}

func (app *App) drawEditEntryWindow(win vaxis.Window) {
	dateStr := app.selectedDate.Format("Monday, January 2, 2006")
	currentEntry := app.entries[app.selectedEntry]
	isTimer := app.isEntryTimer(currentEntry)

	if app.entryStartTime == "" && currentEntry.StartTime != "" && !app.entryTimeInitialized {
		app.entryStartTime = currentEntry.StartTime
	}
	if app.entryEndTime == "" && currentEntry.EndTime != "" && !app.entryTimeInitialized {
		app.entryEndTime = currentEntry.EndTime
	}
	app.entryTimeInitialized = true

	win.Println(0, vaxis.Segment{
		Text:  dateStr,
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	})
	isValid := app.validateTimes()
	startTimeStyle := vaxis.Style{}
	if app.entryEditCursor == EntryCursorStart {
		startTimeStyle.Attribute |= vaxis.AttrReverse
	}
	if !isValid && app.entryStartTime != "" {
		startTimeStyle.Foreground = vaxis.IndexColor(1) // Red for invalid
	}
	win.Println(1, vaxis.Segment{
		Text:  "Start: ",
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	}, vaxis.Segment{
		Text:  app.entryStartTime,
		Style: startTimeStyle,
	})
	endTimeStyle := vaxis.Style{}
	if app.entryEditCursor == EntryCursorEnd {
		endTimeStyle.Attribute |= vaxis.AttrReverse
	}
	if !isValid && app.entryEndTime != "" {
		endTimeStyle.Foreground = vaxis.IndexColor(1) // Red for invalid
	}
	currentEntryEndTime := app.entryEndTime
	if isTimer {
		currentEntryEndTime = "⏱ "
	}
	win.Println(2, vaxis.Segment{
		Text:  "End:   ",
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	}, vaxis.Segment{
		Text:  currentEntryEndTime,
		Style: endTimeStyle,
	})
	currentEntryName := "✕ No task selected"
	if currentEntry.Name != "" {
		currentEntryName = currentEntry.Name
	}
	win.Println(3, vaxis.Segment{
		Text:  "Task:  ",
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	}, vaxis.Segment{
		Text: currentEntryName,
	})

	if currentEntry.TaskID != "" && app.selectedTask == -1 {
		app.selectedTask = app.findTaskIndex(currentEntry.TaskID)
		app.vx.PostEvent(vaxis.Redraw{})
	}

	if app.taskSearchMode {
		win.Println(4, vaxis.Segment{
			Text:  "Search: " + app.taskSearchInput,
			Style: vaxis.Style{Foreground: vaxis.IndexColor(3)},
		})
	}

	if app.taskHierarchy == nil {
		app.taskHierarchy = app.buildTaskHierarchy()
	}

	_, rows := win.Size()
	visibleRows := rows - 6
	if visibleRows < 1 {
		visibleRows = 1
	}
	scrollOffset := app.selectedTask - visibleRows/2
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > len(app.taskHierarchy.AllTasksIDs)-visibleRows {
		scrollOffset = max(0, len(app.taskHierarchy.AllTasksIDs)-visibleRows)
	}

	row := 5
	drawnTasks := 0

	for _, parentID := range app.taskHierarchy.ParentIDs {
		if parentID == 0 {
			continue
		}
		parentTask := findTask(app.tasks, parentID)
		if parentTask == nil {
			continue
		}
		if drawnTasks >= scrollOffset && drawnTasks < scrollOffset+visibleRows {
			parentTaskID := strconv.Itoa(parentTask.TaskID)
			isCurrent := parentTaskID == currentEntry.TaskID
			isSelected := drawnTasks == app.selectedTask
			style := vaxis.Style{Attribute: vaxis.AttrBold}
			if app.entryEditCursor == EntryCursorTask {
				if isCurrent {
					style.Foreground = vaxis.IndexColor(4)
				}
				if isSelected {
					style.Attribute |= vaxis.AttrReverse
				}
			}
			win.Println(row, vaxis.Segment{
				Text:  "• " + parentTask.Name,
				Style: style,
			})
			row++
		}
		drawnTasks++

		children := app.taskHierarchy.ParentTasks[parentID]
		for childIndex, child := range children {
			if drawnTasks >= scrollOffset && drawnTasks < scrollOffset+visibleRows {
				childID := strconv.Itoa(child.TaskID)
				isCurrent := childID == currentEntry.TaskID
				isSelected := drawnTasks == app.selectedTask
				style := vaxis.Style{}
				if app.entryEditCursor == EntryCursorTask {
					if isCurrent {
						style.Foreground = vaxis.IndexColor(4)
					}
					if isSelected {
						style.Attribute = vaxis.AttrReverse
					}
				}
				branch := "└─"
				if childIndex < len(children)-1 {
					branch = "├─"
				}
				win.Println(row, vaxis.Segment{
					Text:  "  " + branch + " " + child.Name,
					Style: style,
				})
				row++
			}
			drawnTasks++
		}
	}
	app.drawnTasks = drawnTasks
}

func (app *App) updateEntry(entryID int64, taskID *int, startTime, endTime string) error {
	type Body struct {
		ID        int64  `json:"id"`
		Date      string `json:"date,omitempty"`
		StartTime string `json:"start_time,omitempty"`
		EndTime   string `json:"end_time,omitempty"`
		Duration  int    `json:"duration,omitempty"`
		TaskID    *int   `json:"task_id,omitempty"`
	}
	type Response struct {
		EntryID string `json:"entry_id"`
		TaskID  string `json:"task_id"`
	}
	body := Body{
		ID:   entryID,
		Date: app.selectedDate.Format("2006-01-02"),
	}
	if taskID != nil {
		body.TaskID = taskID
	}
	currentEntry := app.entries[app.selectedEntry]
	isTimer := app.isEntryTimer(currentEntry)
	if !isTimer {
		if startTime != "" {
			body.StartTime = startTime
		}
		if endTime != "" {
			body.EndTime = endTime
		}
		if startTime != "" && endTime != "" {
			start, err1 := time.Parse("15:04:05", startTime)
			end, err2 := time.Parse("15:04:05", endTime)
			if err1 == nil && err2 == nil {
				duration := end.Sub(start)
				body.Duration = int(duration.Seconds())
			}
		}
	}
	var response Response
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint:    "/entries",
		Method:      "PUT",
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

func (app *App) validateTimes() bool {
	if app.entryStartTime != "" && app.entryEndTime != "" {
		start, _ := time.Parse("15:04:05", app.entryStartTime)
		end, _ := time.Parse("15:04:05", app.entryEndTime)
		if end.Before(start) {
			return false
		}
	}
	return true
}

func (app *App) handleEditEntryKeys(key vaxis.Key) bool {
	currentEntry := app.entries[app.selectedEntry]
	isTimer := app.isEntryTimer(currentEntry)

	if app.taskSearchMode {
		if key.Matches(vaxis.KeyEsc) || key.Matches(vaxis.KeyEnter) || key.Matches('/') {
			app.taskSearchMode = false
			app.taskSearchInput = ""
		} else if key.Matches(vaxis.KeyBackspace) {
			if len(app.taskSearchInput) > 0 {
				app.taskSearchInput = app.taskSearchInput[:len(app.taskSearchInput)-1]
				if len(app.taskSearchInput) > 0 {
					taskIndex := app.findParentTask(app.taskSearchInput)
					if taskIndex >= 0 {
						app.selectedTask = taskIndex
					}
				}
			}
		} else if key.Text != "" {
			app.taskSearchInput += string(key.Text)
			taskIndex := app.findParentTask(app.taskSearchInput)
			if taskIndex >= 0 {
				app.selectedTask = taskIndex
			}
		}
		return false
	}

	if key.Matches('q') || key.Matches(vaxis.KeyEsc) {
		app.showEditEntry = false
		app.entryStartTime = ""
		app.entryEndTime = ""
		app.entryTimeInitialized = false
		app.selectedTask = -1
		return false
	} else if key.Matches(vaxis.KeyTab) {
		app.entryEditCursor = (app.entryEditCursor + 1) % 3
		if app.entryEditCursor == EntryCursorEnd && len(app.timers) > 0 {
			app.entryEditCursor += 1
		}
		if app.entryEditCursor == EntryCursorTask && app.selectedTask < 0 {
			app.selectedTask = 0
		}
	} else if key.Matches(vaxis.KeyEnter) || key.Matches(vaxis.KeySpace) {
		if !app.validateTimes() {
			return false
		}
		app.showEditEntry = false
		go func() {
			var taskID *int
			if app.selectedTask >= 0 {
				taskIDVal := app.taskHierarchy.AllTasksIDs[app.selectedTask]
				taskID = &taskIDVal
			}
			app.updateEntry(
				app.entries[app.selectedEntry].ID,
				taskID,
				app.entryStartTime,
				app.entryEndTime,
			)
			app.selectedTask = -1
			app.fetchEntries(app.selectedDate)
			app.vx.PostEvent(vaxis.Redraw{})
		}()
	}

	if app.entryEditCursor == EntryCursorStart || app.entryEditCursor == EntryCursorEnd {
		currentTime := app.entryStartTime
		if app.entryEditCursor == EntryCursorEnd {
			currentTime = app.entryEndTime
		}
		if key.Matches('/') {
			app.entryEditCursor = EntryCursorTask
			app.taskSearchMode = true
			app.taskSearchInput = ""
		} else if key.Matches(vaxis.KeyDown) {
			app.entryEditCursor += 1
		} else if key.Matches(vaxis.KeyBackspace) {
			if len(currentTime) > 0 && !isTimer {
				if app.entryEditCursor == EntryCursorStart {
					app.entryStartTime = currentTime[:len(currentTime)-1]
				} else {
					app.entryEndTime = currentTime[:len(currentTime)-1]
				}
			}
		} else if key.Text != "" {
			if key.Text[0] >= '0' && key.Text[0] <= '9' {
				if len(currentTime) < 8 {
					newTime := currentTime + key.Text
					// Automatically add colon after two digits for hours
					if len(newTime) == 2 {
						// If we have exactly 2 digits (hours), add a colon
						newTime += ":"
					} else if len(newTime) == 5 {
						// If we have exactly 5 characters (HH:MM), add a colon
						newTime += ":"
					}
					// Validate the time format at each step
					valid := false
					if len(newTime) == 1 {
						// First digit of hour (0-2)
						if h, err := strconv.Atoi(newTime); err == nil && h >= 0 && h <= 2 {
							valid = true
						}
					} else if len(newTime) == 3 { // After auto-adding colon (HH:)
						// Complete hour (00-23)
						if h, err := strconv.Atoi(newTime[:2]); err == nil && h >= 0 && h <= 23 {
							valid = true
						}
					} else if len(newTime) == 4 {
						// First digit of minute (0-5)
						if newTime[2] == ':' {
							if m, err := strconv.Atoi(newTime[3:]); err == nil && m >= 0 && m <= 5 {
								valid = true
							}
						}
					} else if len(newTime) == 6 { // After auto-adding colon (HH:MM:)
						// Complete minute (00-59)
						if newTime[2] == ':' {
							if m, err := strconv.Atoi(newTime[3:5]); err == nil && m >= 0 && m <= 59 {
								valid = true
							}
						}
					} else if len(newTime) == 7 {
						// First digit of second (0-5)
						if newTime[5] == ':' {
							if s, err := strconv.Atoi(newTime[6:]); err == nil && s >= 0 && s <= 5 {
								valid = true
							}
						}
					} else if len(newTime) == 8 {
						// Complete second (00-59)
						if newTime[5] == ':' {
							if s, err := strconv.Atoi(newTime[6:]); err == nil && s >= 0 && s <= 59 {
								valid = true
							}
						}
					}
					if valid {
						currentTime = newTime
						if app.entryEditCursor == EntryCursorStart {
							app.entryStartTime = currentTime
						} else {
							app.entryEndTime = currentTime
						}
					}
				}
			}
		}
	} else if app.entryEditCursor == EntryCursorTask {
		if key.Matches('/') {
			app.taskSearchMode = true
			app.taskSearchInput = ""
			return false
		} else if key.Matches('j') || key.Matches(vaxis.KeyDown) {
			if app.selectedTask < app.drawnTasks-1 {
				app.selectedTask++
			}
		} else if key.Matches('k') || key.Matches(vaxis.KeyUp) {
			if app.selectedTask > 0 {
				app.selectedTask--
			}
		} else if key.Matches('g') {
			app.selectedTask = 0
		} else if key.Matches('G') {
			app.selectedTask = app.drawnTasks - 1
		}
	}

	return false
}
