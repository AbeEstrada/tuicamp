package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

type TaskResponse struct {
	TaskID      int    `json:"task_id"`
	ParentID    int    `json:"parent_id"`
	AssignedBy  int    `json:"assigned_by"`
	Name        string `json:"name"`
	Level       int    `json:"level"`
	BudgetUnit  string `json:"budget_unit"`
	RootGroupID int    `json:"root_group_id"`
}

type TaskHierarchy struct {
	ParentTasks map[int][]TaskResponse
	ParentIDs   []int
	AllTasksIDs []int
}

func (app *App) fetchTasks() error {
	var response map[string]TaskResponse
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint: fmt.Sprintf("/tasks?minimal=1"),
		Method:   "GET",
		Response: &response,
		Headers:  map[string]string{"Authorization": "Bearer " + app.apiToken},
	})
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed API response: %w", result.Error)
	}
	app.tasks = response
	app.taskHierarchy = nil
	return nil
}

func findTask(tasks map[string]TaskResponse, taskID int) *TaskResponse {
	for _, task := range tasks {
		if task.TaskID == taskID {
			return &task
		}
	}
	return nil
}

func (app *App) findTaskIndex(taskID string) int {
	if app.taskHierarchy == nil {
		app.taskHierarchy = app.buildTaskHierarchy()
		app.allTasksIDs = app.taskHierarchy.AllTasksIDs
	}
	for i, id := range app.allTasksIDs {
		if strconv.Itoa(id) == taskID {
			return i
		}
	}
	return -1
}

func (app *App) findParentTaskByFirstLetter(letter string) int {
	if app.taskHierarchy == nil {
		app.taskHierarchy = app.buildTaskHierarchy()
		app.allTasksIDs = app.taskHierarchy.AllTasksIDs
	}
	letter = strings.ToLower(letter)
	for _, parentID := range app.taskHierarchy.ParentIDs {
		task := findTask(app.tasks, parentID)
		if task != nil && strings.HasPrefix(strings.ToLower(task.Name), letter) {
			for i, id := range app.allTasksIDs {
				if id == task.TaskID {
					return i
				}
			}
		}
	}
	return -1 // Not found
}

func (app *App) buildTaskHierarchy() *TaskHierarchy {
	parentTasks := make(map[int][]TaskResponse)
	var parentIDs []int
	var allTasksIDs []int
	for _, task := range app.tasks {
		parentTasks[task.ParentID] = append(parentTasks[task.ParentID], task)
		if task.ParentID == 0 {
			parentIDs = append(parentIDs, task.TaskID)
		}
	}
	sort.Slice(parentIDs, func(i, j int) bool {
		taskI := findTask(app.tasks, parentIDs[i])
		taskJ := findTask(app.tasks, parentIDs[j])
		return strings.ToLower(taskI.Name) < strings.ToLower(taskJ.Name)
	})
	for _, parentID := range parentIDs {
		if parentID == 0 {
			continue
		}
		parentTask := findTask(app.tasks, parentID)
		if parentTask == nil {
			continue
		}
		allTasksIDs = append(allTasksIDs, parentTask.TaskID)
		children := parentTasks[parentID]
		sort.Slice(children, func(i, j int) bool {
			return strings.ToLower(children[i].Name) < strings.ToLower(children[j].Name)
		})
		for _, child := range children {
			allTasksIDs = append(allTasksIDs, child.TaskID)
		}
	}
	return &TaskHierarchy{
		ParentTasks: parentTasks,
		ParentIDs:   parentIDs,
		AllTasksIDs: allTasksIDs,
	}
}
