package main

import (
	"fmt"
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
