package policy

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

//go:generate go run github.com/maxbrunsfeld/counterfeiter/v6 -o policyfakes/fake_clock.go . Clock

type Clock interface {
	Now() time.Time
}

type RealClock struct{}

func (rc RealClock) Now() time.Time {
	return time.Now()
}

type PolicyManager struct {
	policy map[string]int
	clock  Clock
}

func NewPolicyManager(policyStr string) (*PolicyManager, error) {
	return NewPolicyManagerWithClock(policyStr, RealClock{})
}

func NewPolicyManagerWithClock(policyStr string, clock Clock) (*PolicyManager, error) {
	policy, err := parse(policyStr)
	if err != nil {
		return nil, err
	}
	return &PolicyManager{policy: policy, clock: clock}, nil
}

func (pm *PolicyManager) IsWithinPolicy(activeMinutes int) bool {
	allowed := pm.getTodayAllowed()
	return activeMinutes <= allowed
}

func parse(policyStr string) (map[string]int, error) {
	policy := make(map[string]int)
	re := regexp.MustCompile(`([A-Z-]+)(\d+)`)
	matches := re.FindAllStringSubmatch(policyStr, -1)
	for _, match := range matches {
		if len(match) == 3 {
			min, err := strconv.Atoi(match[2])
			if err != nil {
				return nil, err
			}
			policy[match[1]] = min
		}
	}
	if len(policy) == 0 {
		return nil, fmt.Errorf("no valid policy entries found")
	}
	return policy, nil
}

func (pm *PolicyManager) getTodayAllowed() int {
	now := pm.clock.Now()
	weekday := now.Weekday()
	var dayKey string
	switch weekday {
	case time.Monday:
		dayKey = "MO"
	case time.Tuesday:
		dayKey = "TU"
	case time.Wednesday:
		dayKey = "WE"
	case time.Thursday:
		dayKey = "TH"
	case time.Friday:
		dayKey = "FR"
	case time.Saturday:
		dayKey = "SA"
	case time.Sunday:
		dayKey = "SU"
	}

	// Check ranges
	for key, min := range pm.policy {
		if strings.Contains(key, "-") {
			parts := strings.Split(key, "-")
			if len(parts) == 2 {
				if dayInRange(dayKey, parts[0], parts[1]) {
					return min
				}
			}
		} else if key == dayKey {
			return min
		}
	}
	return 0 // Default if not found
}

func dayInRange(day, start, end string) bool {
	days := []string{"MO", "TU", "WE", "TH", "FR", "SA", "SU"}
	startIdx, endIdx := -1, -1
	for i, d := range days {
		if d == start {
			startIdx = i
		}
		if d == end {
			endIdx = i
		}
	}
	dayIdx := -1
	for i, d := range days {
		if d == day {
			dayIdx = i
		}
	}
	return dayIdx >= startIdx && dayIdx <= endIdx
}
