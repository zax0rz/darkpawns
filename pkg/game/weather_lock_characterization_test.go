package game

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const weatherLockCharacterizationCaseEnv = "DP_WEATHER_LOCK_CHARACTERIZATION_CASE"

type weatherLockCharacterizationCase struct {
	name            string
	startHour       int
	mode            bool
	liveWorld       bool
	wantDeadlock    bool
	wantStackHelper string
	wantOutdoor     string
	wantHour        int
	wantSunlight    int
}

func weatherLockCharacterizationCases() []weatherLockCharacterizationCase {
	return []weatherLockCharacterizationCase{
		{
			name:            "event-hour-4-nil",
			startHour:       4,
			mode:            true,
			wantDeadlock:    true,
			wantStackHelper: "ghostShipDisappear",
			wantOutdoor:     "The suns rise in the east and north.\r\n",
			wantHour:        5,
			wantSunlight:    SunRise,
		},
		{
			name:            "event-hour-4-live",
			startHour:       4,
			mode:            true,
			liveWorld:       true,
			wantDeadlock:    true,
			wantStackHelper: "ghostShipDisappear",
			wantOutdoor:     "The suns rise in the east and north.\r\n",
			wantHour:        5,
			wantSunlight:    SunRise,
		},
		{
			name:            "event-hour-20-nil",
			startHour:       20,
			mode:            true,
			wantDeadlock:    true,
			wantStackHelper: "ghostShipAppear",
			wantOutdoor:     "The suns slowly disappear in the west and south.\r\n",
			wantHour:        21,
			wantSunlight:    SunSet,
		},
		{
			name:            "event-hour-20-live",
			startHour:       20,
			mode:            true,
			liveWorld:       true,
			wantDeadlock:    true,
			wantStackHelper: "ghostShipAppear",
			wantOutdoor:     "The suns slowly disappear in the west and south.\r\n",
			wantHour:        21,
			wantSunlight:    SunSet,
		},
		{
			name:      "non-event-hour-7-live",
			startHour: 7,
			mode:      true,
			liveWorld: true,
		},
		{
			name:      "non-event-hour-7-nil",
			startHour: 7,
			mode:      true,
		},
		{
			name:      "mode-false-hour-4-nil",
			startHour: 4,
			mode:      false,
		},
		{
			name:      "mode-false-hour-20-live",
			startHour: 20,
			mode:      false,
			liveWorld: true,
		},
	}
}

// TestWeatherLockCharacterization is intentionally green on the current
// defective implementation. The event-hour cases require a bounded,
// isolated deadlock with the expected lock stack; after the production repair
// they must be converted into completion regressions that require return and
// preserve the event/output order.
func TestWeatherLockCharacterization(t *testing.T) {
	for _, tc := range weatherLockCharacterizationCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWeatherLockCharacterizationWorker$", "-test.v")
			cmd.Env = append(os.Environ(), weatherLockCharacterizationCaseEnv+"="+tc.name)
			output, err := cmd.CombinedOutput()
			text := string(output)
			if ctx.Err() != nil {
				t.Fatalf("characterization worker timed out and was cleaned up: %v\n%s", ctx.Err(), text)
			}
			if err != nil {
				t.Fatalf("characterization worker failed before reporting a result: %v\n%s", err, text)
			}
			if !strings.Contains(text, "READY case="+tc.name) {
				t.Fatalf("worker did not complete setup for %s\n%s", tc.name, text)
			}
			t.Logf("worker evidence:\n%s", text)

			if tc.wantDeadlock {
				assertWeatherLockDeadlockEvidence(t, tc, text)
				return
			}
			if !strings.Contains(text, "CHARACTERIZATION_RETURNED case="+tc.name) {
				t.Fatalf("control case did not return\n%s", text)
			}
			if strings.Contains(text, "CHARACTERIZATION_DEADLOCK") {
				t.Fatalf("control case reported a deadlock\n%s", text)
			}
		})
	}
}

func assertWeatherLockDeadlockEvidence(t *testing.T, tc weatherLockCharacterizationCase, output string) {
	t.Helper()

	for _, marker := range []string{
		"CHARACTERIZATION_DEADLOCK case=" + tc.name,
		"WeatherAndTime",
		"AnotherHour",
		tc.wantStackHelper,
		"sync.(*RWMutex).RLock",
		"OUTDOOR_CALLBACK message=\"" + strings.ReplaceAll(tc.wantOutdoor, "\r\n", "\\r\\n") + "\"",
		"observed_hour=" + intString(tc.wantHour),
		"observed_sunlight=" + intString(tc.wantSunlight),
	} {
		if !strings.Contains(output, marker) {
			t.Errorf("missing expected deadlock evidence %q\n%s", marker, output)
		}
	}
}

func intString(value int) string {
	return strconv.Itoa(value)
}

// TestWeatherLockCharacterizationWorker runs in a separate process so an
// expected characterization deadlock cannot retain weatherMu in the parent
// test process. The worker returns after capturing all goroutine stacks; the
// subprocess then exits and releases any package-global lock it held.
func TestWeatherLockCharacterizationWorker(t *testing.T) {
	caseName := os.Getenv(weatherLockCharacterizationCaseEnv)
	if caseName == "" {
		return
	}

	var tc weatherLockCharacterizationCase
	for _, candidate := range weatherLockCharacterizationCases() {
		if candidate.name == caseName {
			tc = candidate
			break
		}
	}
	if tc.name == "" {
		t.Fatalf("unknown characterization case %q", caseName)
	}

	weatherMu.Lock()
	timeInfo = TimeInfoData{Hours: tc.startHour, Day: 21, Month: 0, Year: 0, Moon: MoonFull}
	weatherInfo = WeatherData{Pressure: 1013, Change: 0, Sky: SkyCloudless, Sunlight: SunLight}
	if tc.liveWorld {
		weatherWorld = &World{}
	} else {
		weatherWorld = nil
	}
	weatherMu.Unlock()

	t.Logf("READY case=%s start_hour=%d mode=%t world=%t", tc.name, tc.startHour, tc.mode, tc.liveWorld)

	var callbackCount atomic.Int32
	var observedHour atomic.Int32
	var observedSunlight atomic.Int32
	outdoor := func(message string) {
		callbackCount.Add(1)
		observedHour.Store(int32(timeInfo.Hours))
		observedSunlight.Store(int32(weatherInfo.Sunlight))
		t.Logf("OUTDOOR_CALLBACK message=%q observed_hour=%d observed_sunlight=%d", message, timeInfo.Hours, weatherInfo.Sunlight)
	}
	returned := make(chan struct{})
	go func() {
		WeatherAndTime(tc.mode, outdoor)
		close(returned)
	}()

	select {
	case <-returned:
		t.Logf("CHARACTERIZATION_RETURNED case=%s hour=%d callbacks=%d", tc.name, timeInfo.Hours, callbackCount.Load())
	case <-time.After(250 * time.Millisecond):
		stacks := make([]byte, 64*1024)
		stacks = stacks[:runtime.Stack(stacks, true)]
		t.Logf("CHARACTERIZATION_DEADLOCK case=%s observed_hour=%d observed_sunlight=%d callbacks=%d", tc.name, observedHour.Load(), observedSunlight.Load(), callbackCount.Load())
		t.Logf("ALL_GOROUTINE_STACKS\n%s", stacks)
	}
}
