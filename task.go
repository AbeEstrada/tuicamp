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
	for i, id := range app.allTasksIDs {
		if strconv.Itoa(id) == taskID {
			return i
		}
	}
	return 0
}

func (app *App) findParentTaskByFirstLetter(letter string) int {
	letter = strings.ToLower(letter)
	var parentIDs []int
	for _, task := range app.tasks {
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
