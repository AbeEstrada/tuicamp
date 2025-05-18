package main

import (
	"fmt"
	"strconv"

	"git.sr.ht/~rockorager/vaxis"
)

func (app *App) drawEditEntryWindow(win vaxis.Window) {
	dateStr := app.selectedDate.Format("Monday, January 2, 2006")
	win.Println(0, vaxis.Segment{
		Text:  dateStr,
		Style: vaxis.Style{Attribute: vaxis.AttrBold},
	})

	currentEntry := app.entries[app.selectedEntry]
	currentEntryName := "✕ No task selected"
	if currentEntry.Name != "" {
		currentEntryName = currentEntry.Name
	}
	win.Println(2, vaxis.Segment{
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
			if isCurrent {
				style.Foreground = vaxis.IndexColor(4)
			}
			if isSelected {
				style.Attribute |= vaxis.AttrReverse
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
				if isCurrent {
					style.Foreground = vaxis.IndexColor(4)
				}
				if isSelected {
					style.Attribute = vaxis.AttrReverse
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

func (app *App) updateEntryTaskID(entryID int64, taskID int) error {
	type Body struct {
		ID     int64 `json:"id"`
		TaskID int   `json:"task_id"`
	}
	type Response struct {
		EntryID string `json:"entry_id"`
		TaskID  string `json:"task_id"`
	}
	body := Body{
		ID:     entryID,
		TaskID: taskID,
	}
	var response Response
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint:    fmt.Sprintf("/entries"),
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

func (app *App) handleEditEntryKeys(key vaxis.Key) bool {
	if app.taskSearchMode {
		if key.Matches(vaxis.KeyEsc) || key.Matches(vaxis.KeyEnter) || key.Matches('/') {
			app.taskSearchMode = false
			app.taskSearchInput = ""
		} else if key.Matches(vaxis.KeyBackspace) {
			if len(app.taskSearchInput) > 0 {
				app.taskSearchInput = app.taskSearchInput[:len(app.taskSearchInput)-1]
				if len(app.taskSearchInput) > 0 {
					taskIndex := app.findParentTaskByFirstLetter(app.taskSearchInput)
					if taskIndex >= 0 {
						app.selectedTask = taskIndex
					}
				}
			}
		} else if key.Text != "" {
			app.taskSearchInput += string(key.Text)
			taskIndex := app.findParentTaskByFirstLetter(app.taskSearchInput)
			if taskIndex >= 0 {
				app.selectedTask = taskIndex
			}
		}
		return false
	}

	if key.Matches('q') || key.Matches(vaxis.KeyEsc) {
		app.showEditEntry = false
		return false
	} else if key.Matches('/') {
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
	} else if key.Matches(vaxis.KeyEnter) || key.Matches(vaxis.KeySpace) {
		app.showEditEntry = false
		go func() {
			app.updateEntryTaskID(app.entries[app.selectedEntry].ID, app.taskHierarchy.AllTasksIDs[app.selectedTask])
			app.fetchEntries(app.selectedDate)
			app.vx.PostEvent(vaxis.Redraw{})
		}()

	}

	return false
}
