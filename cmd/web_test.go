package cmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"home-gate/internal/monitor"
	"home-gate/internal/state"
)

func TestWebStatusEndpoint_ReturnsLatestMonitorState(t *testing.T) {
	// Arrange
	dev := monitor.DeviceUsage{
		MAC:                "aa11bb22cc33",
		Name:               "iPad",
		DailyActiveMinutes: 45,
		Active:             []string{"10:00+02:00/PT45M", "14:15+02:00/PT1H"},
	}
	summary := monitor.Summary{
		DevicesChecked: 3,
		UsersFetched:   2,
		Errors:         nil,
		StartTime:      time.Unix(1000, 0),
		Duration:       500 * time.Millisecond,
		Devices:        []monitor.DeviceUsage{dev},
	}
	state.Update(summary)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(state.Get()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	ts := httptest.NewServer(h)
	defer ts.Close()

	// Act
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatalf("HTTP GET failed: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var got monitor.Summary
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("decode JSON: %v", err)
	}

	// Assert
	if got.DevicesChecked != summary.DevicesChecked || got.UsersFetched != summary.UsersFetched {
		t.Errorf("expected %+v got %+v", summary, got)
	}
	if len(got.Devices) != 1 {
		t.Fatalf("expected 1 device usage, got %d", len(got.Devices))
	}
	actual := got.Devices[0]
	if actual.MAC != dev.MAC {
		t.Errorf("expected device MAC %s got %s", dev.MAC, actual.MAC)
	}
	if actual.DailyActiveMinutes != dev.DailyActiveMinutes {
		t.Errorf("expected daily_active_minutes %d got %d", dev.DailyActiveMinutes, actual.DailyActiveMinutes)
	}
	if len(actual.Active) != len(dev.Active) {
		t.Fatalf("expected %d active blocks got %d", len(dev.Active), len(actual.Active))
	}
	for i, v := range dev.Active {
		if actual.Active[i] != v {
			t.Errorf("at index %d, expected active block %q got %q", i, v, actual.Active[i])
		}
	}
}
