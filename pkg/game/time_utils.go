// Package game — time/age utility functions ported from src/utils.c
//
// Ported functions:
//   real_time_passed → RealTimePassed
//   mud_time_passed  → MudTimePassed
//   age              → Age
//   playing_time     → PlayingTime
//   parse_race       → ParseRace

package game

import (
	"time"
)

// ---------------------------------------------------------------------------
// Time constants — from src/utils.h
// ---------------------------------------------------------------------------

const (
	// MUD time constants (Dark Pawns specific: 63 real sec per Mud hour)
	SECS_PER_MUD_HOUR  = 63
	SECS_PER_MUD_DAY   = 24 * SECS_PER_MUD_HOUR  // 1512
	SECS_PER_MUD_MONTH = 35 * SECS_PER_MUD_DAY   // 52920
	SECS_PER_MUD_YEAR  = 17 * SECS_PER_MUD_MONTH // 899640

	// Real time constants
	SECS_PER_REAL_HOUR = 3600  // 60 * 60
	SECS_PER_REAL_DAY  = 86400 // 24 * 3600
)

// ---------------------------------------------------------------------------
// RealTimePassed — calculate real time elapsed between two timestamps
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// MudTimePassed — calculate MUD time elapsed between two timestamps
// ---------------------------------------------------------------------------

// MudTimePassed converts real seconds into Dark Pawns MUD time units
// (hours, days, months, years) using MUD-specific time constants.
// Ported from mud_time_passed() in src/utils.c.
func MudTimePassed(t2, t1 time.Time) TimeInfoData {
	secs := int64(t2.Sub(t1).Seconds())

	hours := (secs / SECS_PER_MUD_HOUR) % 24
	secs -= SECS_PER_MUD_HOUR * hours

	days := (secs / SECS_PER_MUD_DAY) % 35
	secs -= SECS_PER_MUD_DAY * days

	months := (secs / SECS_PER_MUD_MONTH) % 17
	secs -= SECS_PER_MUD_MONTH * months

	years := secs / SECS_PER_MUD_YEAR

	return TimeInfoData{
		Hours: int(hours),
		Day:   int(days),
		Month: int(months),
		Year:  int(years),
		Moon:  0,
	}
}

// ---------------------------------------------------------------------------
// Age — calculate character age based on birth time
// ---------------------------------------------------------------------------

// Age returns the character's in-game age as a TimeInfoData structure.
// The birth year is offset by +17 (all players start at age 17).
// Ported from age() in src/utils.c.
//
// In C: player_age = mud_time_passed(time(0), ch->player.time.birth);
//
//	player_age.year += 17;
func Age(birthUnix int64) TimeInfoData {
	birth := time.Unix(birthUnix, 0)
	now := time.Now()

	age := MudTimePassed(now, birth)
	age.Year += 17

	return age
}

// ---------------------------------------------------------------------------
// PlayingTime — calculate total play time
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// PlayingTimeString — formatted play-time string
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// ParseRace — convert race name character to race constant
// ---------------------------------------------------------------------------

// ParseRace converts a single-character race abbreviation to the race constant.
// Ported from parse_race() in src/utils.c.
//
// C switch cases (single char):
//
//	'h' → RACE_HUMAN    'e' → RACE_ELF
//	'd' → RACE_DWARF    'k' → RACE_KENDER
//	'r' → RACE_RAKSHASA 'm' → RACE_MINOTAUR
//	's' → RACE_SSAUR
func ParseRace(ch byte) int {
	switch ch {
	case 'h':
		return RaceHuman
	case 'e':
		return RaceElf
	case 'd':
		return RaceDwarf
	case 'k':
		return RaceKender
	case 'r':
		return RaceRakshasa
	case 'm':
		return RaceMinotaur
	case 's':
		return RaceSsaur
	default:
		return -1 // RACE_UNDEFINED — the original only checks via this func at char creation
	}
}
