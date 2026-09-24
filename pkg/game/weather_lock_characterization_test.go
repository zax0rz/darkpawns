package game

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zax0rz/darkpawns/pkg/dprng"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

const weatherLockRegressionCaseEnv = "DP_WEATHER_LOCK_REGRESSION_CASE"

type weatherLockRegressionCase struct {
	name         string
	startHour    int
	day          int
	mode         bool
	liveWorld    bool
	wantHour     int
	wantSunlight int
	wantEvents   []string
}

const (
	sunriseOutdoor   = "outdoor:The suns rise in the east and north.\r\n"
	sunsetOutdoor    = "outdoor:The suns slowly disappear in the west and south.\r\n"
	fullMoonEvent    = "event:[ FULL MOON RISES ] The full moon casts an eerie glow across the land.\r\n"
	lunarHunterEvent = "event:[ LUNAR HUNTER ] The lunar hunter rises in the east, its cry echoing across the valleys.\r\n"
)

func weatherLockRegressionCases() []weatherLockRegressionCase {
	return []weatherLockRegressionCase{
		{
			name:         "event-hour-4-nil",
			startHour:    4,
			day:          21,
			mode:         true,
			wantHour:     5,
			wantSunlight: SunRise,
			wantEvents:   []string{sunriseOutdoor},
		},
		{
			name:         "event-hour-4-live",
			startHour:    4,
			day:          21,
			mode:         true,
			liveWorld:    true,
			wantHour:     5,
			wantSunlight: SunRise,
			wantEvents:   []string{sunriseOutdoor},
		},
		{
			name:         "event-hour-20-nil",
			startHour:    20,
			day:          21,
			mode:         true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor},
		},
		{
			name:         "event-hour-20-live",
			startHour:    20,
			day:          21,
			mode:         true,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor, fullMoonEvent, lunarHunterEvent},
		},
		{
			name:         "non-event-hour-7-nil",
			startHour:    7,
			day:          21,
			mode:         true,
			wantHour:     8,
			wantSunlight: SunLight,
			wantEvents:   []string{},
		},
		{
			name:         "non-event-hour-7-live",
			startHour:    7,
			day:          21,
			mode:         true,
			liveWorld:    true,
			wantHour:     8,
			wantSunlight: SunLight,
			wantEvents:   []string{},
		},
		{
			name:         "mode-false-hour-4-nil",
			startHour:    4,
			day:          21,
			mode:         false,
			wantHour:     5,
			wantSunlight: SunLight,
			wantEvents:   []string{},
		},
		{
			name:         "mode-false-hour-20-live",
			startHour:    20,
			day:          21,
			mode:         false,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunLight,
			wantEvents:   []string{},
		},
		{
			name:         "moon-day-20-nonqualifying",
			startHour:    20,
			day:          20,
			mode:         true,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor},
		},
		{
			name:         "moon-day-21-qualifying",
			startHour:    20,
			day:          21,
			mode:         true,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor, fullMoonEvent, lunarHunterEvent},
		},
		{
			name:         "moon-day-24-qualifying",
			startHour:    20,
			day:          24,
			mode:         true,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor, fullMoonEvent, lunarHunterEvent},
		},
		{
			name:         "moon-day-25-nonqualifying",
			startHour:    20,
			day:          25,
			mode:         true,
			liveWorld:    true,
			wantHour:     21,
			wantSunlight: SunSet,
			wantEvents:   []string{sunsetOutdoor},
		},
	}
}

// TestWeatherLockReentryRegression converts the bounded deadlock proof from
// PR #1456 into completion regressions. The subprocess remains intentional:
// if a future change reintroduces a lock hang, the parent can cleanly kill the
// isolated worker instead of retaining a package-global lock in this test.
func TestWeatherLockReentryRegression(t *testing.T) {
	for _, tc := range weatherLockRegressionCases() {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWeatherLockReentryRegressionWorker$", "-test.v")
			cmd.Env = append(os.Environ(), weatherLockRegressionCaseEnv+"="+tc.name)
			output, err := cmd.CombinedOutput()
			text := string(output)
			if ctx.Err() != nil {
				t.Fatalf("weather regression worker timed out and was cleaned up: %v\n%s", ctx.Err(), text)
			}
			if err != nil {
				t.Fatalf("weather regression worker failed: %v\n%s", err, text)
			}
			if !strings.Contains(text, "READY case="+tc.name) {
				t.Fatalf("worker did not complete setup for %s\n%s", tc.name, text)
			}
			if !strings.Contains(text, "REGRESSION_RETURNED case="+tc.name) {
				t.Fatalf("worker did not report a completed tick for %s\n%s", tc.name, text)
			}
			t.Logf("worker evidence:\n%s", text)
		})
	}
}

// TestWeatherLockReentryRegressionWorker runs a single completion case in a
// separate process. The callback deliberately reads the already lock-owned
// state directly; callbacks must not call weather accessors or mutators that
// try to acquire weatherMu.
func TestWeatherLockReentryRegressionWorker(t *testing.T) {
	caseName := os.Getenv(weatherLockRegressionCaseEnv)
	if caseName == "" {
		return
	}

	var tc weatherLockRegressionCase
	for _, candidate := range weatherLockRegressionCases() {
		if candidate.name == caseName {
			tc = candidate
			break
		}
	}
	if tc.name == "" {
		t.Fatalf("unknown weather regression case %q", caseName)
	}

	observed := make([]string, 0)
	var w *World
	if tc.liveWorld {
		w = newWeatherLockRegressionWorld(t, &observed)
	}

	weatherMu.Lock()
	timeInfo = TimeInfoData{Hours: tc.startHour, Day: tc.day, Month: 0, Year: 0, Moon: MoonFull}
	// Saturating change and pressure keep the ordering assertion focused on
	// the event outputs; WeatherChange still consumes its existing draws.
	weatherInfo = WeatherData{Pressure: 1040, Change: 12, Sky: SkyCloudless, Sunlight: SunLight}
	weatherWorld = w
	weatherMu.Unlock()

	t.Cleanup(func() {
		SetWeatherWorld(nil)
	})

	t.Logf("READY case=%s start_hour=%d day=%d mode=%t world=%t", tc.name, tc.startHour, tc.day, tc.mode, tc.liveWorld)

	var callbackCount atomic.Int32
	var observedCallbackHour atomic.Int32
	var observedCallbackSunlight atomic.Int32
	outdoor := func(message string) {
		callbackCount.Add(1)
		observedCallbackHour.Store(int32(timeInfo.Hours))
		observedCallbackSunlight.Store(int32(weatherInfo.Sunlight))
		observed = append(observed, "outdoor:"+message)
	}
	returned := make(chan struct{})
	go func() {
		WeatherAndTime(tc.mode, outdoor)
		close(returned)
	}()

	select {
	case <-returned:
		state := TimeWeatherSnapshot()
		if state.Time.Hours != tc.wantHour {
			t.Fatalf("hour after weather tick = %d, want %d", state.Time.Hours, tc.wantHour)
		}
		if state.Weather.Sunlight != tc.wantSunlight {
			t.Fatalf("sunlight after weather tick = %d, want %d", state.Weather.Sunlight, tc.wantSunlight)
		}
		if got := int(callbackCount.Load()); got != countOutdoorEvents(tc.wantEvents) {
			t.Fatalf("outdoor callback count = %d, want %d; events=%q", got, countOutdoorEvents(tc.wantEvents), observed)
		}
		if !slices.Equal(observed, tc.wantEvents) {
			t.Fatalf("event order/output = %q, want %q", observed, tc.wantEvents)
		}
		if tc.mode && tc.startHour == 4 {
			if got := observedCallbackHour.Load(); got != 5 {
				t.Fatalf("sunrise callback observed hour = %d, want 5", got)
			}
			if got := observedCallbackSunlight.Load(); got != SunRise {
				t.Fatalf("sunrise callback observed sunlight = %d, want %d", got, SunRise)
			}
		}
		if tc.mode && tc.startHour == 20 {
			if got := observedCallbackHour.Load(); got != 21 {
				t.Fatalf("sunset callback observed hour = %d, want 21", got)
			}
			if got := observedCallbackSunlight.Load(); got != SunSet {
				t.Fatalf("sunset callback observed sunlight = %d, want %d", got, SunSet)
			}
		}
		t.Logf("REGRESSION_RETURNED case=%s hour=%d sunlight=%d callbacks=%d events=%q", tc.name, state.Time.Hours, state.Weather.Sunlight, callbackCount.Load(), observed)
	case <-time.After(250 * time.Millisecond):
		stacks := make([]byte, 64*1024)
		stacks = stacks[:runtime.Stack(stacks, true)]
		t.Fatalf("weather tick did not return within bounded worker window; ALL_GOROUTINE_STACKS\n%s", stacks)
	}
}

func countOutdoorEvents(events []string) int {
	count := 0
	for _, event := range events {
		if strings.HasPrefix(event, "outdoor:") {
			count++
		}
	}
	return count
}

func newWeatherLockRegressionWorld(t *testing.T, observed *[]string) *World {
	t.Helper()
	w, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 8004, Name: "Weather Observer Room", Zone: 8},
	}})
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(w.StopAITicker)
	w.MessageSink = func(_ string, message []byte) {
		*observed = append(*observed, "event:"+string(message))
	}
	if err := w.AddPlayer(NewPlayer(1, "WeatherObserver", 8004)); err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}
	return w
}

// TestWeatherSynchronizationRace is the focused race surface for the new
// ownership contracts. It runs direct entry points, all six synchronized
// helper wrappers, and SetWeatherWorld concurrently; -race is the required
// invocation for this test.
func TestWeatherSynchronizationRace(t *testing.T) {
	weatherMu.Lock()
	originalTime, originalWeather, originalWorld := timeInfo, weatherInfo, weatherWorld
	timeInfo = TimeInfoData{Hours: 4, Day: 21, Moon: MoonFull}
	weatherInfo = WeatherData{Pressure: 1013, Sky: SkyCloudless, Sunlight: SunLight}
	weatherWorld = nil
	weatherMu.Unlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		timeInfo, weatherInfo, weatherWorld = originalTime, originalWeather, originalWorld
		weatherMu.Unlock()
	})

	worldA := &World{}
	worldB := &World{}
	SetWeatherWorld(worldA)

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(5)
		go func(i int) {
			defer wg.Done()
			if i%2 == 0 {
				SetWeatherWorld(worldA)
			} else {
				SetWeatherWorld(worldB)
			}
		}(i)
		go func() {
			defer wg.Done()
			WeatherAndTime(true, nil)
		}()
		go func() {
			defer wg.Done()
			AnotherHour(false, nil)
		}()
		go func() {
			defer wg.Done()
			WeatherChange(nil)
		}()
		go func() {
			defer wg.Done()
			fullMoon()
			lunarHunter()
			loadNightGate()
			removeNightGate()
			ghostShipAppear()
			ghostShipDisappear()
		}()
	}
	wg.Wait()
	_ = TimeWeatherSnapshot()
}

type weatherDraw struct {
	method string
	a      int
	b      int
}

// referenceWeatherChange mirrors only the C weather_change draw sequence in
// an independent generator. It is intentionally separate from production
// dice/number calls so the test detects moved, added, removed, or re-gated
// draws rather than merely checking the final weather message.
func referenceWeatherChange(seed uint32, month int, initial WeatherData) (WeatherData, []weatherDraw) {
	rng := dprng.New(seed)
	weather := initial
	draws := make([]weatherDraw, 0, 6)
	diceReference := func(num, sides int) int {
		draws = append(draws, weatherDraw{method: "Dice", a: num, b: sides})
		return rng.Dice(num, sides)
	}
	numberReference := func(from, to int) int {
		draws = append(draws, weatherDraw{method: "Number", a: from, b: to})
		return rng.Number(from, to)
	}

	diff := 2
	if month >= 9 && month <= 16 {
		if weather.Pressure > 985 {
			diff = -2
		}
	} else if weather.Pressure > 1015 {
		diff = -2
	}
	weather.Change += diceReference(1, 4)*diff + diceReference(2, 6) - diceReference(2, 6)
	if weather.Change > 12 {
		weather.Change = 12
	}
	if weather.Change < -12 {
		weather.Change = -12
	}
	weather.Pressure += weather.Change
	if weather.Pressure > 1040 {
		weather.Pressure = 1040
	}
	if weather.Pressure < 960 {
		weather.Pressure = 960
	}

	skyChange := 0
	switch weather.Sky {
	case SkyCloudless:
		if weather.Pressure < 990 {
			skyChange = 1
		} else if weather.Pressure < 1010 && numberReference(1, 4) == 1 {
			skyChange = 1
		}
	case SkyCloudy:
		if weather.Pressure < 970 {
			skyChange = 2
		} else if weather.Pressure < 990 {
			if numberReference(1, 4) == 1 {
				skyChange = 2
			}
		} else if weather.Pressure > 1030 && numberReference(1, 4) == 1 {
			skyChange = 3
		}
	case SkyRaining:
		if weather.Pressure < 970 {
			if numberReference(1, 4) == 1 {
				skyChange = 4
			}
		} else if weather.Pressure > 1030 {
			skyChange = 5
		} else if weather.Pressure > 1010 && numberReference(1, 4) == 1 {
			skyChange = 5
		}
	case SkyLightning:
		if weather.Pressure > 1010 {
			skyChange = 6
		} else if weather.Pressure > 990 && numberReference(1, 4) == 1 {
			skyChange = 6
		}
	default:
		weather.Sky = SkyCloudless
	}

	switch skyChange {
	case 1:
		weather.Sky = SkyCloudy
	case 2:
		weather.Sky = SkyRaining
	case 3:
		weather.Sky = SkyCloudless
	case 4:
		weather.Sky = SkyLightning
	case 5:
		weather.Sky = SkyCloudy
	case 6:
		weather.Sky = SkyRaining
	}
	return weather, draws
}

func TestWeatherAndTimePreservesWeatherDrawOrder(t *testing.T) {
	originalTime, originalWeather := TimeSnapshot(), WeatherSnapshot()
	t.Cleanup(func() {
		weatherMu.Lock()
		timeInfo, weatherInfo = originalTime, originalWeather
		weatherMu.Unlock()
		SetWeatherWorld(nil)
		dprng.ResetStream(1)
	})

	cases := []struct {
		name                string
		seed                uint32
		mode                bool
		startHour           int
		month               int
		weather             WeatherData
		wantConditionalDraw bool
	}{
		{
			name:      "event-hour-removal-reaches-existing-weather-change",
			seed:      1,
			mode:      true,
			startHour: 4,
			month:     0,
			weather:   WeatherData{Pressure: 1040, Change: 12, Sky: SkyCloudless, Sunlight: SunLight},
		},
		{
			name:                "cloudless-pressure-window",
			seed:                1,
			mode:                true,
			startHour:           7,
			month:               0,
			weather:             WeatherData{Pressure: 1000, Change: 0, Sky: SkyCloudless, Sunlight: SunLight},
			wantConditionalDraw: true,
		},
		{
			name:                "cloudy-pressure-window",
			seed:                3,
			mode:                true,
			startHour:           7,
			month:               0,
			weather:             WeatherData{Pressure: 980, Change: 0, Sky: SkyCloudy, Sunlight: SunLight},
			wantConditionalDraw: true,
		},
		{
			name:                "raining-pressure-window",
			seed:                5,
			mode:                true,
			startHour:           7,
			month:               0,
			weather:             WeatherData{Pressure: 1015, Change: 0, Sky: SkyRaining, Sunlight: SunLight},
			wantConditionalDraw: true,
		},
		{
			name:                "lightning-pressure-window",
			seed:                8,
			mode:                true,
			startHour:           7,
			month:               0,
			weather:             WeatherData{Pressure: 1000, Change: 0, Sky: SkyLightning, Sunlight: SunLight},
			wantConditionalDraw: true,
		},
		{
			name:      "mode-false-skips-weather-change",
			seed:      13,
			mode:      false,
			startHour: 4,
			month:     0,
			weather:   WeatherData{Pressure: 980, Change: 0, Sky: SkyCloudless, Sunlight: SunLight},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			wantWeather := tc.weather
			wantDraws := []weatherDraw(nil)
			if tc.mode {
				wantWeather, wantDraws = referenceWeatherChange(tc.seed, tc.month, tc.weather)
				switch tc.startHour {
				case 4:
					wantWeather.Sunlight = SunRise
				case 20:
					wantWeather.Sunlight = SunSet
				}
			}
			hasConditionalDraw := false
			for _, draw := range wantDraws {
				if draw.method == "Number" && draw.a == 1 && draw.b == 4 {
					hasConditionalDraw = true
				}
			}
			if hasConditionalDraw != tc.wantConditionalDraw {
				t.Fatalf("independent C draw plan = %#v, conditional draw=%t, want conditional=%t", wantDraws, hasConditionalDraw, tc.wantConditionalDraw)
			}

			weatherMu.Lock()
			timeInfo = TimeInfoData{Hours: tc.startHour, Month: tc.month}
			weatherInfo = tc.weather
			weatherWorld = nil
			weatherMu.Unlock()
			dprng.ResetStream(tc.seed)
			WeatherAndTime(tc.mode, nil)
			gotWeather := WeatherSnapshot()
			gotNext := dprng.Number(0, 999)
			reference := dprng.New(tc.seed)
			for _, draw := range wantDraws {
				switch draw.method {
				case "Dice":
					reference.Dice(draw.a, draw.b)
				case "Number":
					reference.Number(draw.a, draw.b)
				}
			}
			wantNext := reference.Number(0, 999)
			if gotWeather != wantWeather {
				t.Fatalf("weather after draw sequence = %+v, want %+v; plan=%#v", gotWeather, wantWeather, wantDraws)
			}
			if gotNext != wantNext {
				t.Fatalf("next draw after weather sequence = %d, want %d; plan=%#v", gotNext, wantNext, wantDraws)
			}
		})
	}
}
