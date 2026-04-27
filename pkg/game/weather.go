// Port of weather.c — time and weather system
//
// All rights reserved.  See license.doc for complete information.
//
// Copyright (C) 1993, 94 by the Trustees of the Johns Hopkins University
// CircleMUD is based on DikuMUD, Copyright (C) 1990, 1991.
//
// All parts of this code not covered by the copyright by the Trustees of
// the Johns Hopkins University are Copyright (C) 1996, 97, 98 by the
// Dark Pawns Coding Team.
//
// This includes all original code done for Dark Pawns MUD by other authors.
// All code is the intellectual property of the author, and is used here
// by permission.
//
// No original code may be duplicated, reused, or executed without the
// written permission of the author. All rights reserved.
//
// See dp-team.txt or "help coding" online for members of the Dark Pawns
// Coding Team.

package game

import (
	"math/rand"
	"sync"
)

// Sun state constants — from structs.h:571-575
const (
	SunDark  = 0
	SunRise  = 1
	SunLight = 2
	SunSet   = 3
)

// Sky condition constants — from structs.h:579-582
const (
	SkyCloudless = 0
	SkyCloudy    = 1
	SkyRaining   = 2
	SkyLightning = 3
)

// Moon phase constants — from structs.h:595-602
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

// TimeInfoData holds the current MUD time.
// Ported from structs.h:struct time_info_data.
type TimeInfoData struct {
	Hours int
	Day   int
	Month int
	Year  int
	Moon  int
}

// WeatherData holds the current weather state.
// Ported from structs.h:struct weather_data.
type WeatherData struct {
	Pressure  int
	Change    int
	Sky       int
	Sunlight  int
}

var (
	timeInfo = TimeInfoData{
		Hours: 0,
		Day:   0,
		Month: 0,
		Year:  0,
		Moon:  MoonNew,
	}
	weatherInfo = WeatherData{
		Pressure:  1013,
		Change:    0,
		Sky:       SkyCloudless,
		Sunlight:  SunLight,
	}
	weatherMu sync.RWMutex
)

// WeatherAndTime advances time and updates weather.
// Ported from weather.c:weather_and_time().
// mode controls whether weather_change() is called (mode=1 or true).
func WeatherAndTime(mode bool, sendToOutdoor func(string)) {
	weatherMu.Lock()
	AnotherHour(mode, sendToOutdoor)
	if mode {
		WeatherChange(sendToOutdoor)
	}
	weatherMu.Unlock()
}

// AnotherHour advances the MUD time by one hour.
// Ported from weather.c:another_hour().
func AnotherHour(mode bool, sendToOutdoor func(string)) {
	timeInfo.Hours++

	if mode {
		switch timeInfo.Hours {
		case 5:
			weatherInfo.Sunlight = SunRise
			if sendToOutdoor != nil {
				sendToOutdoor("The suns rise in the east and north.\r\n")
			}
			ghostShipDisappear()
			removeNightGate()
		case 6:
			weatherInfo.Sunlight = SunLight
			if sendToOutdoor != nil {
				sendToOutdoor("The day has begun.\r\n")
			}
		case 21:
			weatherInfo.Sunlight = SunSet
			if sendToOutdoor != nil {
				sendToOutdoor("The suns slowly disappear in the west and south.\r\n")
			}
			ghostShipAppear()
			loadNightGate()
			if timeInfo.Day+1 < 26 && timeInfo.Day+1 >= 22 {
				fullMoon()
				lunarHunter()
			}
		case 22:
			weatherInfo.Sunlight = SunDark
			if sendToOutdoor != nil {
				sendToOutdoor("The night has begun.\r\n")
			}
		}
	}

	if timeInfo.Hours > 23 {
		timeInfo.Hours -= 24
		timeInfo.Day++

		switch timeInfo.Day {
		case 1:
			timeInfo.Moon = MoonNew
		case 6:
			timeInfo.Moon = MoonQuarterFull
		case 12:
			timeInfo.Moon = MoonHalfFull
		case 17:
			timeInfo.Moon = MoonThreeFull
		case 22:
			timeInfo.Moon = MoonFull
		case 26:
			timeInfo.Moon = MoonQuarterEmpty
		case 30:
			timeInfo.Moon = MoonHalfEmpty
		case 34:
			timeInfo.Moon = MoonThreeEmpty
		}

		if timeInfo.Day > 34 {
			timeInfo.Day = 0
			timeInfo.Month++

			if timeInfo.Month > 16 {
				timeInfo.Month = 0
				timeInfo.Year++
			}
		}
	}
}

// WeatherChange updates the weather based on pressure changes.
// Ported from weather.c:weather_change().
func WeatherChange(sendToOutdoor func(string)) {
	var diff int
	if timeInfo.Month >= 9 && timeInfo.Month <= 16 {
		// Winter months
		if weatherInfo.Pressure > 985 {
			diff = -2
		} else {
			diff = 2
		}
	} else {
		// Summer months
		if weatherInfo.Pressure > 1015 {
			diff = -2
		} else {
			diff = 2
		}
	}

	change := (dice(1, 4) * diff) + dice(2, 6) - dice(2, 6)

	weatherInfo.Change += change

	if weatherInfo.Change > 12 {
		weatherInfo.Change = 12
	}
	if weatherInfo.Change < -12 {
		weatherInfo.Change = -12
	}

	weatherInfo.Pressure += weatherInfo.Change

	if weatherInfo.Pressure > 1040 {
		weatherInfo.Pressure = 1040
	}
	if weatherInfo.Pressure < 960 {
		weatherInfo.Pressure = 960
	}

	skyChange := 0

	switch weatherInfo.Sky {
	case SkyCloudless:
		if weatherInfo.Pressure < 990 {
			skyChange = 1
		} else if weatherInfo.Pressure < 1010 {
			if number(1, 4) == 1 {
				skyChange = 1
			}
		}
	case SkyCloudy:
		if weatherInfo.Pressure < 970 {
			skyChange = 2
		} else if weatherInfo.Pressure < 990 {
			if number(1, 4) == 1 {
				skyChange = 2
			} else {
				skyChange = 0
			}
		} else if weatherInfo.Pressure > 1030 {
			if number(1, 4) == 1 {
				skyChange = 3
			}
		}
	case SkyRaining:
		if weatherInfo.Pressure < 970 {
			if number(1, 4) == 1 {
				skyChange = 4
			} else {
				skyChange = 0
			}
		} else if weatherInfo.Pressure > 1030 {
			skyChange = 5
		} else if weatherInfo.Pressure > 1010 {
			if number(1, 4) == 1 {
				skyChange = 5
			}
		}
	case SkyLightning:
		if weatherInfo.Pressure > 1010 {
			skyChange = 6
		} else if weatherInfo.Pressure > 990 {
			if number(1, 4) == 1 {
				skyChange = 6
			}
		}
	default:
		skyChange = 0
		weatherInfo.Sky = SkyCloudless
	}

	switch skyChange {
	case 0:
		// no change
	case 1:
		if sendToOutdoor != nil {
			sendToOutdoor("The sky starts to get cloudy.\r\n")
		}
		weatherInfo.Sky = SkyCloudy
	case 2:
		if sendToOutdoor != nil {
			sendToOutdoor("It starts to rain.\r\n")
		}
		weatherInfo.Sky = SkyRaining
	case 3:
		if sendToOutdoor != nil {
			sendToOutdoor("The clouds disappear.\r\n")
		}
		weatherInfo.Sky = SkyCloudless
	case 4:
		if sendToOutdoor != nil {
			sendToOutdoor("Lightning starts to show in the sky.\r\n")
		}
		weatherInfo.Sky = SkyLightning
	case 5:
		if sendToOutdoor != nil {
			sendToOutdoor("The rain stops.\r\n")
		}
		weatherInfo.Sky = SkyCloudy
	case 6:
		if sendToOutdoor != nil {
			sendToOutdoor("The lightning stops.\r\n")
		}
		weatherInfo.Sky = SkyRaining
	}
}

// dice simulates the C number() dice roll: roll "num" dice with "size" sides.
func dice(num, size int) int {
	if size <= 0 {
		return 0
	}
	total := 0
	for i := 0; i < num; i++ {
		// #nosec G404 — game RNG, not cryptographic
// #nosec G404
		total += rand.Intn(size) + 1
	}
	return total
}

// ---------------------------------------------------------------------------
// Stub no-ops for special weather events (ported but not yet implemented).
// These are declared as package-level functions matching the C void prototypes.
// ---------------------------------------------------------------------------

func fullMoon()          {}
func lunarHunter()       {}
func loadNightGate()     {}
func removeNightGate()   {}
func ghostShipAppear()   {}
func ghostShipDisappear() {} 

// ModifyWeatherChange adjusts the weather change variable.
// Used by spell_control_weather.
func ModifyWeatherChange(delta int) {
	weatherMu.Lock()
	weatherInfo.Change += delta
	weatherMu.Unlock()
}

