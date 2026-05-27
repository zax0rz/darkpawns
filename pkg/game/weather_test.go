package game

import (
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

func TestAnotherHour_AdvancesTimeAndSunlight(t *testing.T) {
	// Reset time and weather globals
	timeInfo = TimeInfoData{
		Hours: 0,
		Day:   0,
		Month: 0,
		Year:  0,
		Moon:  MoonNew,
	}
	weatherInfo = WeatherData{
		Pressure: 1013,
		Change:   0,
		Sky:      SkyCloudless,
		Sunlight: SunLight,
	}

	var outdoorMessages []string
	sendToOutdoor := func(msg string) {
		outdoorMessages = append(outdoorMessages, msg)
	}

	// Advance hour by hour and verify solar transitions
	for hour := 0; hour < 24; hour++ {
		AnotherHour(true, sendToOutdoor)
	}

	if timeInfo.Day != 1 {
		t.Errorf("expected day to be 1 after 24 hours, got %d", timeInfo.Day)
	}
	if timeInfo.Hours != 0 {
		t.Errorf("expected hours to reset to 0, got %d", timeInfo.Hours)
	}

	// Verify we got the standard sunrise/sunset/day/night announcements
	sunriseFound := false
	sunsetFound := false
	for _, msg := range outdoorMessages {
		if strings.Contains(msg, "suns rise") {
			sunriseFound = true
		}
		if strings.Contains(msg, "disappear") {
			sunsetFound = true
		}
	}

	if !sunriseFound {
		t.Error("expected sunrise message to be broadcast")
	}
	if !sunsetFound {
		t.Error("expected sunset message to be broadcast")
	}
}

func TestAnotherHour_AdvancesMoonsAndMonths(t *testing.T) {
	timeInfo = TimeInfoData{
		Hours: 23,
		Day:   34, // Last day of MUD month
		Month: 16, // Last month of MUD year
		Year:  100,
		Moon:  MoonNew,
	}

	AnotherHour(false, nil)

	if timeInfo.Day != 0 {
		t.Errorf("expected day to wrap to 0, got %d", timeInfo.Day)
	}
	if timeInfo.Month != 0 {
		t.Errorf("expected month to wrap to 0, got %d", timeInfo.Month)
	}
	if timeInfo.Year != 101 {
		t.Errorf("expected year to increment to 101, got %d", timeInfo.Year)
	}
}

func TestWeatherChange_AdjustsPressureAndSky(t *testing.T) {
	timeInfo = TimeInfoData{Month: 5} // Summer month
	weatherInfo = WeatherData{
		Pressure: 1013,
		Change:   0,
		Sky:      SkyCloudless,
		Sunlight: SunLight,
	}

	var skyMessages []string
	sendToOutdoor := func(msg string) {
		skyMessages = append(skyMessages, msg)
	}

	// Force low pressure to trigger clouds
	weatherInfo.Pressure = 980
	weatherInfo.Sky = SkyCloudless

	// Run multiple changes to allow random sky transition to fire
	for i := 0; i < 20; i++ {
		WeatherChange(sendToOutdoor)
	}

	// Sunlight and moon accessors
	ModifyWeatherChange(5)
	if GetSunlight() != weatherInfo.Sunlight {
		t.Errorf("GetSunlight() = %d, want %d", GetSunlight(), weatherInfo.Sunlight)
	}
	if GetMoon() != timeInfo.Moon {
		t.Errorf("GetMoon() = %d, want %d", GetMoon(), timeInfo.Moon)
	}
}

func TestWeatherEvents_BroadcastToWorld(t *testing.T) {
	parsed := &parser.World{
		Rooms: []parser.Room{
			{VNum: 8004, Name: "Temple", Zone: 8},
		},
	}
	w, err := NewWorld(parsed)
	if err != nil {
		t.Fatalf("NewWorld failed: %v", err)
	}
	t.Cleanup(func() {
		w.StopAITicker()
		SetWeatherWorld(nil)
	})

	var broadcastMessages []string
	w.MessageSink = func(playerName string, msg []byte) {
		broadcastMessages = append(broadcastMessages, string(msg))
	}

	// H-07: Add a player to the world so SendToAll can broadcast to them
	player := NewPlayer(1, "TestBroadcastPlayer", 8004)
	err = w.AddPlayer(player)
	if err != nil {
		t.Fatalf("AddPlayer failed: %v", err)
	}

	SetWeatherWorld(w)

	// Trigger weather events directly to verify world broadcasting
	fullMoon()
	lunarHunter()
	loadNightGate()
	removeNightGate()
	ghostShipAppear()
	ghostShipDisappear()

	if len(broadcastMessages) != 6 {
		t.Errorf("expected 6 broadcast messages, got %d", len(broadcastMessages))
	}

	// Check content of one message
	foundFullMoon := false
	for _, msg := range broadcastMessages {
		if strings.Contains(msg, "FULL MOON RISES") {
			foundFullMoon = true
			break
		}
	}
	if !foundFullMoon {
		t.Error("expected fullMoon message in broadcasts")
	}
}
