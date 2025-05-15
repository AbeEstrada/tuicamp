package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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

	parentTasks := make(map[int][]TaskResponse)
	var parentIDs []int
	var allTasks []TaskResponse
	for _, task := range app.tasks {
		parentTasks[task.ParentID] = append(parentTasks[task.ParentID], task)
		if task.ParentID == 0 {
			parentIDs = append(parentIDs, task.TaskID)
		}
		allTasks = append(allTasks, task)
	}
	sort.Slice(parentIDs, func(i, j int) bool {
		taskI := findTask(app.tasks, parentIDs[i])
		taskJ := findTask(app.tasks, parentIDs[j])
		return strings.ToLower(taskI.Name) < strings.ToLower(taskJ.Name)
	})

	_, rows := win.Size()
	visibleRows := rows - 5
	if visibleRows < 1 {
		visibleRows = 1
	}
	scrollOffset := app.selectedTask - visibleRows/2
	if scrollOffset < 0 {
		scrollOffset = 0
	}
	if scrollOffset > len(allTasks)-visibleRows {
		scrollOffset = max(0, len(allTasks)-visibleRows)
	}

	row := 4
	drawnTasks := 0
	var allTasksIDs []int

	for _, parentID := range parentIDs {
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
				Text:  "• " + parentTask.Name + " " + parentTaskID,
				Style: style,
			})
			row++
		}
		drawnTasks++
		allTasksIDs = append(allTasksIDs, parentTask.TaskID)

		children := parentTasks[parentID]
		sort.Slice(children, func(i, j int) bool {
			return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
		})

		for _, child := range children {
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
				win.Println(row, vaxis.Segment{
					Text:  "  └─ " + child.Name + " " + childID,
					Style: style,
				})
				row++
			}
			drawnTasks++
			allTasksIDs = append(allTasksIDs, child.TaskID)

		}
	}
	app.drawnTasks = drawnTasks
	app.allTasksIDs = allTasksIDs
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
	if key.Matches('q') || key.Matches(vaxis.KeyEsc) {
		app.showEditEntry = false
		return false
	}
	if key.Matches('j') || key.Matches(vaxis.KeyDown) {
		if app.selectedTask < app.drawnTasks-1 {
			app.selectedTask++
		}
	} else if key.Matches('k') || key.Matches(vaxis.KeyUp) {
		if app.selectedTask > 0 {
			app.selectedTask--
		}
	} else if key.Matches(vaxis.KeyEnter) {
		app.showEditEntry = false
		go func() {
			app.updateEntryTaskID(app.entries[app.selectedEntry].ID, app.allTasksIDs[app.selectedTask])
			app.fetchEntries(app.selectedDate)
			app.vx.PostEvent(vaxis.Redraw{})
		}()

	}

	return false
}
