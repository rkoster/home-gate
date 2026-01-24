package cmd_test

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"home-gate/internal/fritzbox"
	"home-gate/internal/fritzbox/fritzboxfakes"
	"home-gate/internal/monitor"
)

// helper to build measurements of given length with active indices set
func buildMeasurements(length int, activeIdxs map[int]bool, activeVal float64) []float64 {
	m := make([]float64, length)
	for i := 0; i < length; i++ {
		if activeIdxs[i] {
			m[i] = activeVal
		} else {
			m[i] = 0.0
		}
	}
	return m
}

func TestMonitor_UsesTodayForDailyTotal(t *testing.T) {
	fake := &fritzboxfakes.FakeClient{}

	mac := "20c9d07d3b1b"
	totalIntervals := 96

	now := time.Now()
	minutesPastMidnight := now.Hour()*60 + now.Minute()
	intervalsSinceMidnight := minutesPastMidnight / 15

	activeEarlier := 10
	start := totalIntervals - intervalsSinceMidnight
	if start < 0 {
		start = 0
	}
	activeIdxs := make(map[int]bool)
	for i := 0; i < activeEarlier && i < start; i++ {
		activeIdxs[i] = true
	}

	rcv := buildMeasurements(totalIntervals, activeIdxs, 100.0)
	snd := buildMeasurements(totalIntervals, map[int]bool{}, 0.0)
	ds1 := fritzbox.SubsetData{DataSourceName: "rcv_" + mac, Measurements: rcv}
	ds2 := fritzbox.SubsetData{DataSourceName: "snd_" + mac, Measurements: snd}
	fake.GetMonitorDataReturns([]fritzbox.SubsetData{ds1, ds2}, nil)
	fake.GetLandevicesReturns([]fritzbox.Landevice{{MAC: mac, UserUIDs: "user-123", FriendlyName: "Test Device", Blocked: "0"}}, nil)
	fake.GetMonitorConfigReturns(fritzbox.MonitorConfig{DisplayHomenetDevices: ""}, nil)

	var out bytes.Buffer
	_, _ = monitor.Run(
		testingContext(),
		monitor.Options{
			Username:          "irrelevant",
			Password:          "irrelevant",
			Mac:               mac,
			Period:            "day",
			ActivityThreshold: 10.0,
			PolicyString:      "MO-SU90",
			Enforce:           false,
			Out:               &out,
			TestClient:        fake,
		},
	)
	// error is ignored for compatibility

	if !strings.Contains(out.String(), "Within policy") {
		t.Fatalf("expected Within policy in output, got:\n%s", out.String())
	}
}

func TestMonitor_EnforcesOnReachingLimit(t *testing.T) {
	fake := &fritzboxfakes.FakeClient{}

	mac := "6c006b9068e9"
	totalIntervals := 96
	now := time.Now()
	minutesPastMidnight := now.Hour()*60 + now.Minute()
	intervalsSinceMidnight := minutesPastMidnight / 15
	if intervalsSinceMidnight == 0 {
		intervalsSinceMidnight = 1
	}
	activeIdxs := make(map[int]bool)
	start := totalIntervals - intervalsSinceMidnight
	if start < 0 {
		start = 0
	}
	for i := start; i < totalIntervals; i++ {
		activeIdxs[i] = true
	}

	rcv := buildMeasurements(totalIntervals, activeIdxs, 100.0)
	snd := buildMeasurements(totalIntervals, map[int]bool{}, 0.0)
	ds1 := fritzbox.SubsetData{DataSourceName: "rcv_" + mac, Measurements: rcv}
	ds2 := fritzbox.SubsetData{DataSourceName: "snd_" + mac, Measurements: snd}
	fake.GetMonitorDataReturns([]fritzbox.SubsetData{ds1, ds2}, nil)
	fake.GetLandevicesReturns([]fritzbox.Landevice{{MAC: mac, UserUIDs: "user-123", FriendlyName: "Tablet", Blocked: "0"}}, nil)
	fake.GetMonitorConfigReturns(fritzbox.MonitorConfig{DisplayHomenetDevices: ""}, nil)

	allowed := intervalsSinceMidnight * 15
	policyStr := fmt.Sprintf("MO-SU%d", allowed)

	var out bytes.Buffer
	_, _ = monitor.Run(
		testingContext(),
		monitor.Options{
			Username:          "irrelevant",
			Password:          "irrelevant",
			Mac:               mac,
			Period:            "day",
			ActivityThreshold: 10.0,
			PolicyString:      policyStr,
			Enforce:           true,
			Out:               &out,
			TestClient:        fake,
		},
	)
	// error is ignored for compatibility

	if fake.BlockDeviceCallCount() != 1 {
		t.Fatalf("expected BlockDevice called once, got %d; output:\n%s", fake.BlockDeviceCallCount(), out.String())
	}
	uid, block := fake.BlockDeviceArgsForCall(0)
	if uid != "user-123" || block != true {
		t.Fatalf("unexpected block args: %v %v", uid, block)
	}
}

func testingContext() (ctx context.Context) {
	return context.Background()
}
