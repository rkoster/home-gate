package cmd_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	"time"

	"home-gate/cmd"
	"home-gate/internal/fritzbox"
	"home-gate/internal/fritzbox/fritzboxfakes"
	"home-gate/internal/policy"
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

func TestRunMonitor_UsesTodayForDailyTotal(t *testing.T) {
	// arrange
	fake := &fritzboxfakes.FakeClient{}

	// choose normalized mac
	mac := "20c9d07d3b1b"
	// 96 intervals = full day
	totalIntervals := 96

	// compute intervals since midnight
	now := time.Now()
	minutesPastMidnight := now.Hour()*60 + now.Minute()
	_ = minutesPastMidnight
	intervalsSinceMidnight := minutesPastMidnight / 15

	// create active intervals only in the old part of the day (indices < totalIntervals-intervalsSinceMidnight)
	// pick up to 10 active intervals earlier in the day
	activeEarlier := 10
	start := totalIntervals - intervalsSinceMidnight
	if start < 0 {
		start = 0
	}
	activeIdxs := make(map[int]bool)
	// mark active intervals before today's start to simulate activity from previous day
	for i := 0; i < activeEarlier && i < start; i++ {
		activeIdxs[i] = true
	}

	rcv := buildMeasurements(totalIntervals, activeIdxs, 100.0)
	snd := buildMeasurements(totalIntervals, map[int]bool{}, 0.0)

	ds1 := fritzbox.SubsetData{DataSourceName: "rcv_" + mac, Measurements: rcv}
	ds2 := fritzbox.SubsetData{DataSourceName: "snd_" + mac, Measurements: snd}
	fake.GetMonitorDataReturns([]fritzbox.SubsetData{ds1, ds2}, nil)

	// policy allows 90 minutes -> 6 intervals
	pm, err := policy.NewPolicyManager("MO-SU90")
	if err != nil {
		t.Fatalf("failed to create policy manager: %v", err)
	}

	var landevices []fritzbox.Landevice
	var config fritzbox.MonitorConfig
	macToUserUID := map[string]string{}

	// act
	var out bytes.Buffer
	// activity-threshold set low so our 100.0 counts as active
	cmd.RunMonitor(fake, pm, mac, "day", 10.0, landevices, config, macToUserUID, false, &out)

	// assert: full-day active would be 10*15 =150 > 90, but today-only active should be 0 -> within policy
	s := out.String()
	if !strings.Contains(s, "Within policy") {
		t.Fatalf("expected Within policy in output, got:\n%s", s)
	}
}

func TestRunMonitor_EnforcesOnReachingLimit(t *testing.T) {
	fake := &fritzboxfakes.FakeClient{}

	mac := "6c006b9068e9"
	totalIntervals := 96

	now := time.Now()
	minutesPastMidnight := now.Hour()*60 + now.Minute()
	intervalsSinceMidnight := minutesPastMidnight / 15
	if intervalsSinceMidnight == 0 {
		// ensure at least one interval
		intervalsSinceMidnight = 1
	}

	// mark the last intervalsSinceMidnight intervals as active
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

	// allowed equals exactly today's active minutes
	allowed := intervalsSinceMidnight * 15
	policyStr := fmt.Sprintf("MO-SU%d", allowed)
	pm, err := policy.NewPolicyManager(policyStr)
	if err != nil {
		t.Fatalf("failed to create policy manager: %v", err)
	}

	// supply landevice so RunMonitor can find device and block by userUID
	dev := fritzbox.Landevice{MAC: mac, UserUIDs: "user-123", FriendlyName: "Tablet", Blocked: "0"}
	landevices := []fritzbox.Landevice{dev}
	macToUserUID := map[string]string{mac: "user-123"}
	var config fritzbox.MonitorConfig

	var out bytes.Buffer
	// act: enforce true
	cmd.RunMonitor(fake, pm, mac, "day", 10.0, landevices, config, macToUserUID, true, &out)

	// assert blocked called once with user-123 true
	if fake.BlockDeviceCallCount() != 1 {
		t.Fatalf("expected BlockDevice called once, got %d; output:\n%s", fake.BlockDeviceCallCount(), out.String())
	}
	uid, block := fake.BlockDeviceArgsForCall(0)
	if uid != "user-123" || block != true {
		t.Fatalf("unexpected block args: %v %v", uid, block)
	}
}
