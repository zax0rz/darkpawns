package session

import (
	"fmt"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func displayHour(hour int) int {
	hour %= 12
	if hour == 0 {
		return 12
	}
	return hour
}

func daySuffix(zeroBasedDay int) string {
	day := zeroBasedDay + 1
	switch {
	case day == 1:
		return "st"
	case day == 2:
		return "nd"
	case day == 3:
		return "rd"
	case day < 20:
		return "th"
	case day%10 == 1:
		return "st"
	case day%10 == 2:
		return "nd"
	case day%10 == 3:
		return "rd"
	default:
		return "th"
	}
}

func formatMudTime(snapshot game.WorldClimateSnapshot) string {
	timeInfo := snapshot.Time
	period := "am"
	if timeInfo.Hours >= 12 {
		period = "pm"
	}
	weekday := (35*timeInfo.Month + timeInfo.Day + 1) % len(game.WeekdayNames)
	day := timeInfo.Day + 1

	var output strings.Builder
	fmt.Fprintf(&output, "It is %d o'clock %s, on %s\r\n",
		displayHour(timeInfo.Hours), period, game.WeekdayNames[weekday])
	fmt.Fprintf(&output, "The %d%s Day of the %s, Year %d.\r\n",
		day, daySuffix(timeInfo.Day), game.MonthNames[timeInfo.Month], timeInfo.Year)
	if snapshot.Weather.Sunlight == game.SunDark {
		fmt.Fprintf(&output, "The moon is %s.\r\n", game.Phases[timeInfo.Moon])
	}
	return output.String()
}

func formatMudWeather(weather game.WeatherData, outside bool) string {
	if !outside {
		return "You have no feeling about the weather at all.\r\n"
	}

	sky := [...]string{"cloudless", "cloudy", "rainy", "lit by flashes of lightning"}
	direction := "you feel a warm wind from south"
	if weather.Change < 0 {
		direction = "the clouds tell you bad weather is due"
	}
	return fmt.Sprintf("The sky is %s and %s.\r\n", sky[weather.Sky], direction)
}

// cmdTime renders the same canonical clock and sunlight state used by world
// darkness and weather events.
// Source: act.informative.c do_time() lines 1498-1543.
func cmdTime(s *Session, args []string) error {
	s.Send(formatMudTime(game.TimeWeatherSnapshot()))
	return nil
}

// cmdWeather renders the canonical weather state with C's OUTSIDE gate.
// Source: act.informative.c do_weather() lines 1546-1563.
func cmdWeather(s *Session, args []string) error {
	snapshot := game.WeatherSnapshot()
	s.Send(formatMudWeather(snapshot, s.manager.world.IsOutside(s.player.GetRoom())))
	return nil
}
