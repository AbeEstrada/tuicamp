package main

import (
	"fmt"
	"time"

	"git.sr.ht/~rockorager/vaxis"
)

type TimersRunningResponse struct {
	TimerID   string  `json:"timer_id"`
	UserID    string  `json:"user_id"`
	TaskID    *string `json:"task_id"` // Nullable field
	StartedAt string  `json:"started_at"`
	Name      *string `json:"name"` // Nullable field
}

func (app *App) fetchTimers() error {
	var timers []TimersRunningResponse
	resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
		Endpoint: fmt.Sprintf("/timer_running"),
		Method:   "GET",
		Response: &timers,
		Headers:  map[string]string{"Authorization": "Bearer " + app.apiToken},
	})
	result := <-resultChan
	if result.Error != nil {
		return fmt.Errorf("failed API response: %w", result.Error)
	}
	app.timers = timers
	app.updateTimer()
	return nil
}

func (app *App) startTimer() error {
	if len(app.timers) == 0 {
		type Body struct {
			Action string `json:"action"`
		}
		type Response struct {
			EntryID int `json:"entry_id"`
		}
		body := Body{
			Action: "start",
		}
		var reponse Response
		resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
			Endpoint:    fmt.Sprintf("/timer"),
			Method:      "POST",
			RequestBody: &body,
			Response:    &reponse,
			Headers:     map[string]string{"Authorization": "Bearer " + app.apiToken},
		})
		result := <-resultChan
		if result.Error != nil {
			return fmt.Errorf("failed API response: %w", result.Error)
		} else {
			app.fetchEntries(app.selectedDate)
			app.fetchTimers()
		}
	}
	return nil
}

func (app *App) stopTimers() error {
	if len(app.timers) > 0 {
		type Body struct {
			Action string `json:"action"`
			TaskID string `json:"task_id"`
		}
		type Response struct {
			Elapsed   int    `json:"elapsed"`
			EntryID   string `json:"entry_id"`
			EntryTime int    `json:"entry_time"`
		}
		body := Body{
			Action: "stop",
			TaskID: fmt.Sprintf("%v", app.timers[0].TaskID),
		}
		var reponse Response
		resultChan := app.apiClient.CallAsyncWithChannel(CallOptions{
			Endpoint:    fmt.Sprintf("/timer"),
			Method:      "POST",
			RequestBody: &body,
			Response:    &reponse,
			Headers:     map[string]string{"Authorization": "Bearer " + app.apiToken},
		})
		result := <-resultChan
		if result.Error != nil {
			return fmt.Errorf("failed API response: %w", result.Error)
		}
		app.timerTicker.Stop()
		app.timers = []TimersRunningResponse{}
		app.fetchEntries(app.selectedDate)
	}
	return nil
}

func (app *App) updateTimer() {
	if app.timerTicker != nil {
		app.timerTicker.Stop()
		app.timerTicker = nil
	}
	if len(app.timers) > 0 {
		startTime, err := time.ParseInLocation("2006-01-02 15:04:05", app.timers[0].StartedAt, app.currentMonth.Location())
		if err == nil {
			app.elapsedTime = time.Since(startTime)
			app.timerTicker = time.NewTicker(1 * time.Second)
			done := make(chan struct{})
			app.timerDone = done
			go func() {
				for {
					select {
					case <-app.timerTicker.C:
						app.elapsedTime = time.Since(startTime)
						app.vx.PostEvent(vaxis.Redraw{})
					case <-done:
						return
					}
				}
			}()
		}
	} else {
		app.elapsedTime = 0
		if app.timerDone != nil {
			close(app.timerDone)
			app.timerDone = nil
		}
	}
}

func (app *App) drawTimerWindow(win vaxis.Window) {
	win.Println(0,
		vaxis.Segment{
			Text:  "Timer",
			Style: vaxis.Style{Attribute: vaxis.AttrBold},
		})

	if app.timers == nil {
		win.Println(2, vaxis.Segment{
			Text:  "Loading timer status...",
			Style: vaxis.Style{Attribute: vaxis.AttrItalic},
		})
		return
	}

	buttonText := "Start timer ▶"
	if len(app.timers) > 0 {
		buttonText = "Stop timer ■"
	}
	focusedStyle := vaxis.Style{
		Attribute: vaxis.AttrReverse,
	}
	buttonStyle := vaxis.Style{
		Attribute: vaxis.AttrBold,
	}

	currentStyle := buttonStyle
	if app.focusedWindow == Timer {
		currentStyle = focusedStyle
	}

	win.Println(2,
		vaxis.Segment{Text: buttonText, Style: currentStyle},
	)
	if len(app.timers) > 0 {
		startedAt, _ := time.ParseInLocation("2006-01-02 15:04:05", app.timers[0].StartedAt, app.currentMonth.Location())
		win.Println(4, vaxis.Segment{Text: "Started: " + startedAt.Format("Monday, January 2, 2006 15:04:05")})

		hours := int(app.elapsedTime.Hours())
		minutes := int(app.elapsedTime.Minutes()) % 60
		seconds := int(app.elapsedTime.Seconds()) % 60
		elapsedText := fmt.Sprintf("Elapsed: %02d:%02d:%02d", hours, minutes, seconds)
		win.Println(6, vaxis.Segment{Text: elapsedText})
	}
}

func (app *App) handleTimerKeys(key vaxis.Key) bool {
	if app.showQuitConfirm {
		return false
	}
	if key.Matches('H') {
		app.focusedWindow = Calendar
	} else if key.Matches('J') {
		app.focusedWindow = Content
	} else if key.Matches(vaxis.KeyEnter) || key.Matches(vaxis.KeySpace) {
		if len(app.timers) > 0 {
			app.stopTimers()
		} else {
			app.startTimer()
		}
	}
	return false
}
