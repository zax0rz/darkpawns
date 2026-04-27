package session

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

// ---------------------------------------------------------------------------
// Time system
// Based on the original Dark Pawns (DikuMUD variant) time model:
//   1 IRL minute = 1 MUD hour
//   24 MUD hours  = 1 MUD day  (24 IRL minutes)
//   35 MUD days   = 1 MUD month (840 IRL minutes ≈ 14 hours)
//   17 MUD months = 1 MUD year
// ---------------------------------------------------------------------------

// TimePeriod names keyed by hour (0-23).
var timePeriods = map[int]string{
	5:  "dawn",
	12: "noon",
}

func timePeriodName(hour int) string {
	switch {
	case hour == 5:
		return "dawn"
	case hour >= 6 && hour <= 11:
		return "morning"
	case hour == 12:
		return "noon"
	case hour >= 13 && hour <= 18:
		return "afternoon"
	case hour >= 19 && hour <= 20:
		return "evening"
	default: // 21-23, 0-4
		return "night"
	}
}

func amPm(hour int) string {
	if hour < 12 {
		return "am"
	}
	return "pm"
}

func displayHour(hour int) int {
	h := hour % 12
	if h == 0 {
		return 12
	}
	return h
}

// Month names (17 months per year)
var monthNames = []string{
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
	"Frost", "Dark", "Void", "Ash", "Bloom",
}

// Ordinal suffix helper
func daySuffix(day int) string {
	day++ // convert from 0-based to 1-based for display
	switch day % 10 {
	case 1:
		if day%100 != 11 {
			return "st"
		}
	case 2:
		if day%100 != 12 {
			return "nd"
		}
	case 3:
		if day%100 != 13 {
			return "rd"
		}
	}
	return "th"
}

// TimeState holds the current MUD time.
type TimeState struct {
	StartTime time.Time // Reference wall-clock time for calculating MUD time
	Hour      int       // 0-23
	Day       int       // 0-34 (35 days per month)
	Month     int       // 0-16 (17 months per year)
	Year      int
	Pulse     int // running pulse counter (incremented by UpdateWorldTime)
}

var worldTime = TimeState{
	StartTime: time.Now(),
	Hour:      0,
	Day:       0,
	Month:     0,
	Year:      0,
	Pulse:     0,
}

var timeMu sync.RWMutex

// UpdateWorldTime advances the world time by one pulse.
func UpdateWorldTime() {
	timeMu.Lock()
	defer timeMu.Unlock()
	worldTime.Pulse++
}

// GetCurrentTime calculates the current MUD time from elapsed real time.
func GetCurrentTime() (hour, day, month, year int) {
	timeMu.RLock()
	start := worldTime.StartTime
	timeMu.RUnlock()

	elapsed := time.Since(start)
	totalHours := int(elapsed.Minutes()) // 1 real minute = 1 MUD hour

	hour = totalHours % 24
	totalDays := totalHours / 24
	day = totalDays % 35
	totalMonths := totalDays / 35
	month = totalMonths % 17
	year = totalMonths / 17
	return
}

// cmdTime displays the current MUD time.
// Source: act.informative.c ACMD(do_time)
func cmdTime(s *Session, args []string) error {
	hour, day, month, year := GetCurrentTime()

	period := timePeriodName(hour)
	ampm := amPm(hour)
	dspHour := displayHour(hour)
	monthName := monthNames[month]
	suffix := daySuffix(day)
	dayDisplay := day + 1 // 1-based for display

	s.Send(fmt.Sprintf("It is %d o'clock %s %s, on the %d%s day of %s, Year %d.",
		dspHour, period, ampm, dayDisplay, suffix, monthName, year))

	return nil
}

// ---------------------------------------------------------------------------
// Weather system
// ---------------------------------------------------------------------------

// WeatherState holds the current weather.
type WeatherState struct {
	Type      string    // "clear", "cloudy", "rainy", "stormy", "foggy", "snowy"
	UpdatedAt time.Time // When the weather was last rolled
}

var currentWeather = WeatherState{
	Type:      "clear",
	UpdatedAt: time.Now(),
}

var weatherMu sync.RWMutex

// getWeather returns the current weather, refreshing it if the cache has expired.
// Weather is stable per 10-minute IRL window.
func getWeather(hour, month int) string {
	weatherMu.RLock()
	cached := currentWeather.Type
	cachedAt := currentWeather.UpdatedAt
	weatherMu.RUnlock()

	// Cache for 10 real minutes
	if time.Since(cachedAt) < 10*time.Minute {
		return cached
	}

	// Determine possible weather types based on season (month)
	var options []string
	weights := []int{}

	switch {
	case month >= 0 && month <= 3: // Winter
		options = []string{"snowy", "clear", "cloudy"}
		weights = []int{40, 30, 30}
	case month >= 4 && month <= 7: // Spring
		options = []string{"rainy", "cloudy", "clear"}
		weights = []int{40, 35, 25}
	case month >= 8 && month <= 11: // Summer
		options = []string{"clear", "stormy", "hot"}
		weights = []int{50, 25, 25}
	default: // Fall (months 12-16)
		options = []string{"cloudy", "rainy", "foggy"}
		weights = []int{35, 35, 30}
	}

	// Weighted random selection
	totalWeight := 0
	for _, w := range weights {
		totalWeight += w
	}
	// #nosec G404 — game RNG, not cryptographic
// #nosec G404
	roll := rand.Intn(totalWeight)
	cumulative := 0
	selected := options[0]
	for i, w := range weights {
		cumulative += w
		if roll < cumulative {
			selected = options[i]
			break
		}
	}

	weatherMu.Lock()
	currentWeather.Type = selected
	currentWeather.UpdatedAt = time.Now()
	weatherMu.Unlock()

	return selected
}

// WeatherMessage returns the human-readable weather description.
func weatherMessage(weatherType string) string {
	switch weatherType {
	case "clear":
		return "The sky is clear."
	case "cloudy":
		return "It is cloudy."
	case "rainy":
		return "It is raining."
	case "stormy":
		return "A storm rages!"
	case "foggy":
		return "The fog is thick here."
	case "snowy":
		return "Snow falls gently."
	default:
		return "The sky is clear."
	}
}

// cmdWeather displays the current weather.
// Source: act.informative.c ACMD(do_weather)
func cmdWeather(s *Session, args []string) error {
	_, _, month, _ := GetCurrentTime()

	// Use current hour for time-of-day effects
	hour, _, _, _ := GetCurrentTime()
	_ = hour // available for future time-of-day weather modifiers

	weatherType := getWeather(hour, month)
	s.Send(weatherMessage(weatherType))

	return nil
}

// ---------------------------------------------------------------------------
// Weather cycle — ported from weather.c
// Source: src/weather.c another_hour(), weather_change()
// ---------------------------------------------------------------------------

// Sun state constants — Source: structs.h lines 572–575
const (
	SunDark  = 0
	SunRise  = 1
	SunLight = 2
	SunSet   = 3
)

// Sky condition constants — Source: structs.h lines 579–582
const (
	SkyCloudless = 0
	SkyCloudy    = 1
	SkyRaining   = 2
	SkyLightning = 3
)

// Moon phase constants — Source: structs.h lines 595–602
const (
	MoonNew          = 0
	MoonQuarterFull  = 1
	MoonHalfFull     = 2
	MoonThreeFull    = 3
	MoonFull         = 4
	MoonQuarterEmpty = 5
	MoonHalfEmpty    = 6
	MoonThreeEmpty   = 7
)

// WorldWeatherState is the full world weather/time state used by AnotherHour and WeatherChange.
// Corresponds to the global time_info (struct time_info_data) and
// weather_info (struct weather_data) in the original C code.
// Source: structs.h lines 858–861 (time_info_data), lines 1290–1296 (weather_data)
type WorldWeatherState struct {
	// Time — time_info_data
	Hours int
	Day   int
	Month int
	Moon  int
	Year  int

	// Weather — weather_data
	Pressure int // Atmospheric pressure in mb (960–1040)
	Change   int // Pressure change rate (-12 to +12)
	Sky      int // Current sky condition (SkyCloudless..SkyLightning)
	Sunlight int // Current sun state (SunDark..SunSet)
}

// OutdoorBroadcastFunc is a callback for sending weather messages to outdoor rooms.
// Corresponds to send_to_outdoor() in weather.c.
type OutdoorBroadcastFunc func(msg string)

// AnotherHourCallbacks holds optional callbacks for world events triggered by
// the hour tick (ghost ship, night gate, full moon, lunar hunter).
// These correspond to the extern function calls in another_hour() that are
// implemented elsewhere in the C codebase.
// Phase 3 — wire ghost_ship_appear/disappear, load_night_gate, remove_night_gate,
//   full_moon(), lunar_hunter() when spec_procs are fully ported.
type AnotherHourCallbacks struct {
	GhostShipAppear    func()
	GhostShipDisappear func()
	LoadNightGate      func()
	RemoveNightGate    func()
	FullMoon           func()
	LunarHunter        func()
}

// dice rolls nDice dice each with nSides sides and returns the sum.
// Source: utils.h dice() macro / utils.c — used in weather_change()
func dice(nDice, nSides int) int {
	if nDice <= 0 || nSides <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < nDice; i++ {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		total += rand.Intn(nSides) + 1
	}
	return total
}

// AnotherHour advances the world clock by one MUD hour and dispatches
// time-of-day events (sunrise, sunset, moon phase changes).
// Source: weather.c another_hour() lines 41–125
//
// mode=1: send outdoor messages and trigger world events.
// mode=0: only advance the clock (used during boot to set initial time).
func AnotherHour(state *WorldWeatherState, mode int, broadcast OutdoorBroadcastFunc, cbs *AnotherHourCallbacks) {
	state.Hours++

	if mode == 1 {
		switch state.Hours {
		case 5:
			state.Sunlight = SunRise
			broadcast("The suns rise in the east and north.\r\n")
			if cbs != nil && cbs.GhostShipDisappear != nil {
				cbs.GhostShipDisappear()
			}
			if cbs != nil && cbs.RemoveNightGate != nil {
				cbs.RemoveNightGate()
			}
		case 6:
			state.Sunlight = SunLight
			broadcast("The day has begun.\r\n")
		case 21:
			state.Sunlight = SunSet
			broadcast("The suns slowly disappear in the west and south.\r\n")
			if cbs != nil && cbs.GhostShipAppear != nil {
				cbs.GhostShipAppear()
			}
			if cbs != nil && cbs.LoadNightGate != nil {
				cbs.LoadNightGate()
			}
			// Full moon check: days 22–25 (0-based: day+1 in range 22–25)
			// Source: weather.c line 72: if (time_info.day+1 <26 && time_info.day+1 >=22)
			dayOneBased := state.Day + 1
			if dayOneBased >= 22 && dayOneBased < 26 {
				if cbs != nil && cbs.FullMoon != nil {
					cbs.FullMoon()
				}
				if cbs != nil && cbs.LunarHunter != nil {
					cbs.LunarHunter()
				}
			}
		case 22:
			state.Sunlight = SunDark
			broadcast("The night has begun.\r\n")
		}
	}

	// Day rollover — Source: weather.c lines 86–124
	if state.Hours > 23 {
		state.Hours -= 24
		state.Day++

		// Moon phase on specific days — Source: weather.c switch(time_info.day)
		switch state.Day {
		case 1:
			state.Moon = MoonNew
		case 6:
			state.Moon = MoonQuarterFull
		case 12:
			state.Moon = MoonHalfFull
		case 17:
			state.Moon = MoonThreeFull
		case 22:
			state.Moon = MoonFull
		case 26:
			state.Moon = MoonQuarterEmpty
		case 30:
			state.Moon = MoonHalfEmpty
		case 34:
			state.Moon = MoonThreeEmpty
		}

		// Month rollover at day 35
		if state.Day > 34 {
			state.Day = 0
			state.Month++

			// Year rollover at month 17
			if state.Month > 16 {
				state.Month = 0
				state.Year++
			}
		}
	}
}

// WeatherChange runs one weather pressure/sky update cycle.
// Source: weather.c weather_change() lines 130–229
//
// Pressure oscillates between 960–1040 mb.
// Sky transitions: cloudless → cloudy → raining → lightning and back.
// Summer months (9–16) target lower pressure baseline.
func WeatherChange(state *WorldWeatherState, broadcast OutdoorBroadcastFunc) {
	var diff int
	// Source: weather.c lines 132–135 — winter vs summer pressure targets
	if state.Month >= 9 && state.Month <= 16 {
		if state.Pressure > 985 {
			diff = -2
		} else {
			diff = 2
		}
	} else {
		if state.Pressure > 1015 {
			diff = -2
		} else {
			diff = 2
		}
	}

	// Random pressure change — Source: weather.c line 137
	state.Change += dice(1, 4)*diff + dice(2, 6) - dice(2, 6)

	// Clamp change to ±12
	if state.Change > 12 {
		state.Change = 12
	}
	if state.Change < -12 {
		state.Change = -12
	}

	state.Pressure += state.Change

	// Clamp pressure to 960–1040
	if state.Pressure > 1040 {
		state.Pressure = 1040
	}
	if state.Pressure < 960 {
		state.Pressure = 960
	}

	change := 0

	// Determine sky transition based on current sky and pressure
	// Source: weather.c switch(weather_info.sky) lines 149–196
	switch state.Sky {
	case SkyCloudless:
		if state.Pressure < 990 {
			change = 1
		} else if state.Pressure < 1010 {
			if dice(1, 4) == 1 {
				change = 1
			}
		}
	case SkyCloudy:
		if state.Pressure < 970 {
			change = 2
		} else if state.Pressure < 990 {
			if dice(1, 4) == 1 {
				change = 2
			} else {
				change = 0
			}
		} else if state.Pressure > 1030 {
			if dice(1, 4) == 1 {
				change = 3
			}
		}
	case SkyRaining:
		if state.Pressure < 970 {
			if dice(1, 4) == 1 {
				change = 4
			} else {
				change = 0
			}
		} else if state.Pressure > 1030 {
			change = 5
		} else if state.Pressure > 1010 {
			if dice(1, 4) == 1 {
				change = 5
			}
		}
	case SkyLightning:
		if state.Pressure > 1010 {
			change = 6
		} else if state.Pressure > 990 {
			if dice(1, 4) == 1 {
				change = 6
			}
		}
	default:
		change = 0
		state.Sky = SkyCloudless
	}

	// Apply sky transition and send outdoor message
	// Source: weather.c switch(change) lines 198–228
	switch change {
	case 0:
		// no change
	case 1:
		broadcast("The sky starts to get cloudy.\r\n")
		state.Sky = SkyCloudy
	case 2:
		broadcast("It starts to rain.\r\n")
		state.Sky = SkyRaining
	case 3:
		broadcast("The clouds disappear.\r\n")
		state.Sky = SkyCloudless
	case 4:
		broadcast("Lightning starts to show in the sky.\r\n")
		state.Sky = SkyLightning
	case 5:
		broadcast("The rain stops.\r\n")
		state.Sky = SkyCloudy
	case 6:
		broadcast("The lightning stops.\r\n")
		state.Sky = SkyRaining
	}
}

// WeatherAndTime runs one full weather+time tick.
// Source: weather.c weather_and_time() lines 32–37
// mode=1: advance clock AND run weather cycle. mode=0: advance clock only.
func WeatherAndTime(state *WorldWeatherState, mode int, broadcast OutdoorBroadcastFunc, cbs *AnotherHourCallbacks) {
	AnotherHour(state, mode, broadcast, cbs)
	if mode == 1 {
		WeatherChange(state, broadcast)
	}
}

// Go Improvements Over C (weather helpers)
// =========================================
// 1. NO GLOBALS: C kept time_info and weather_info as external globals accessed
//    by every function. Go passes WorldWeatherState as an explicit parameter —
//    testable without any global setup.
//
// 2. CALLBACKS: C's another_hour() called ghost_ship_appear() etc. directly.
//    Go uses AnotherHourCallbacks struct so callers can inject or stub these.
//
// 3. BROADCAST: C called send_to_outdoor() (global descriptor scan). Go takes
//    an OutdoorBroadcastFunc callback — decouples weather from session layer.
//
// 4. POTENTIAL MODERNIZATION (do not implement now):
//    - Merge WorldWeatherState with TimeState (already in this file) for a single
//      source of truth.
//    - Persist weather state across server restarts (pressure/sky/phase).
//    - The existing getWeather() above uses a different, simplified algorithm.
//      When WorldWeatherState is wired into the main game loop, getWeather() and
//      weatherMessage() should delegate to WorldWeatherState.Sky for consistency.

