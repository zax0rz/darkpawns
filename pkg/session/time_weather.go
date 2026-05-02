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
	var weights []int

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
