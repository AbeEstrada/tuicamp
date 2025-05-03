package main

import (
	"fmt"

	"git.sr.ht/~rockorager/vaxis"
)

type MeResponse struct {
	UserID       string              `json:"user_id"` // string or integer
	Email        string              `json:"email"`
	RegisterTime string              `json:"register_time"`
	DisplayName  string              `json:"display_name"`  // nullable
	SynchTime    string              `json:"synch_time"`    // nullable
	RootGroupID  string              `json:"root_group_id"` // string or integer
	Permissions  PermissionsResponse `json:"permissions"`
}

type PermissionsResponse struct {
	TimeTrackingAdmin bool `json:"time_tracking_admin"`
	CreateProjects    bool `json:"create_projects"`
	CanViewRates      bool `json:"can_view_rates"`
}

func (app *App) fetchMe() error {
	if len(app.timers) == 0 {
		var response MeResponse
		resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
			Endpoint: fmt.Sprintf("/me"),
			Method:   "GET",
			Response: &response,
			Headers:  map[string]string{"Authorization": "Bearer " + app.apiToken},
		})
		result := <-resultChan
		if result.Error != nil {
			return fmt.Errorf("failed API response: %w", result.Error)
		}
		app.me = response
	}
	return nil
}

func (app *App) drawUserWindow(win vaxis.Window) {
	displayName := app.me.DisplayName
	email := app.me.Email
	if app.me.DisplayName != "" {
		email = " (" + app.me.Email + ") "
	}
	win.Println(0,
		vaxis.Segment{
			Text: displayName,
		},
		vaxis.Segment{
			Text: email,
		})
}
