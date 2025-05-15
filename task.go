package main

import (
	"fmt"
	"strconv"
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
