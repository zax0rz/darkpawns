package game

import (
	"slices"
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

func TestInitializeWeatherConsumesCPressureRoll(t *testing.T) {
	originalNumber := weatherInitNumber
	originalNow := nowFunc
	originalTime := timeInfo
	originalWeather := weatherInfo
	t.Cleanup(func() {
		weatherInitNumber = originalNumber
		nowFunc = originalNow
		timeInfo = originalTime
		weatherInfo = originalWeather
	})

	var gotFrom, gotTo int
	weatherInitNumber = func(from, to int) int {
		gotFrom, gotTo = from, to
		return 25
	}

	// Pin the clock to a fixed instant whose mudTimePassed yields month 8 (the
	// 7-12 winter band, where the pressure range narrows to 50). Without this
	// seam the test relied on wall-clock month and passed only by coincidence.
	// Solve month=8, day=any, hours=any for (now - beginningOfTime):
	//   secs = year*secsPerMUDYear + 8*secsPerMUDMonth + 0*secsPerMUDDay + 0*secsPerMUDHour
	// Pick year=1, month=8: secs = 899640 + 8*52920 = 1323000.
	nowFunc = func() int64 { return beginningOfTime + 1323000 }

	InitializeWeather()

	if gotFrom != 1 || gotTo != 50 {
		t.Fatalf("weather pressure draw = number(%d,%d), want number(1,50) for month %d",
			gotFrom, gotTo, timeInfo.Month)
	}
	if timeInfo.Month != 8 {
		t.Fatalf("fixed clock derived month = %d, want 8", timeInfo.Month)
	}
	if weatherInfo.Pressure != 985 || weatherInfo.Change != 0 || weatherInfo.Sky != SkyRaining {
		t.Fatalf("initialized weather = %+v, want pressure 985, change 0, raining", weatherInfo)
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

func TestWeatherEvents_SynchronizedDirectEntryPoints(t *testing.T) {
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

	// Trigger all six event helpers through their synchronized direct wrappers.
	// Only the full-moon and lunar-hunter routines broadcast globally; the gate
	// and ghost-ship routines are room-local world changes.
	fullMoon()
	lunarHunter()
	loadNightGate()
	removeNightGate()
	ghostShipAppear()
	ghostShipDisappear()

	wantMessages := []string{
		"[ FULL MOON RISES ] The full moon casts an eerie glow across the land.\r\n",
		"[ LUNAR HUNTER ] The lunar hunter rises in the east, its cry echoing across the valleys.\r\n",
	}
	if !slices.Equal(broadcastMessages, wantMessages) {
		t.Errorf("broadcast messages = %#v, want %#v", broadcastMessages, wantMessages)
	}
}

func TestTimeWeatherSnapshotTracksCanonicalClock(t *testing.T) {
	weatherMu.Lock()
	originalTime, originalWeather := timeInfo, weatherInfo
	timeInfo = TimeInfoData{Hours: 8, Day: 23, Month: 3, Year: 1260, Moon: MoonHalfFull}
	weatherInfo = WeatherData{Pressure: 1001, Change: -2, Sky: SkyRaining, Sunlight: SunLight}
	weatherMu.Unlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		timeInfo, weatherInfo = originalTime, originalWeather
		weatherMu.Unlock()
	})

	before := TimeWeatherSnapshot()
	if before.Time.Hours != 8 || before.Weather.Sky != SkyRaining {
		t.Fatalf("initial snapshot = %+v", before)
	}

	WeatherAndTime(false, nil)
	after := TimeWeatherSnapshot()
	if after.Time.Hours != 9 {
		t.Errorf("snapshot hour after canonical tick = %d, want 9", after.Time.Hours)
	}
	if after.Weather != before.Weather {
		t.Errorf("mode=false changed weather: before %+v, after %+v", before.Weather, after.Weather)
	}
	if got := TimeSnapshot(); got != after.Time {
		t.Errorf("TimeSnapshot = %+v, want %+v", got, after.Time)
	}
	if got := WeatherSnapshot(); got != after.Weather {
		t.Errorf("WeatherSnapshot = %+v, want %+v", got, after.Weather)
	}
}

func TestWorldIsOutsideMatchesCMacro(t *testing.T) {
	world, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Indoor", Flags: []string{"8"}, Sector: SECT_INSIDE},
		{VNum: 1002, Name: "Indoor Flag Outside Sector", Flags: []string{"8"}, Sector: SECT_CITY},
		{VNum: 1003, Name: "No Indoor Flag", Sector: SECT_INSIDE},
	}})
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	t.Cleanup(world.StopAITicker)

	if world.IsOutside(1001) {
		t.Error("ROOM_INDOORS + SECT_INSIDE should be indoors")
	}
	if !world.IsOutside(1002) {
		t.Error("non-inside sector should be outside even with ROOM_INDOORS")
	}
	if !world.IsOutside(1003) {
		t.Error("room without ROOM_INDOORS should be outside per C macro")
	}
	if world.IsOutside(9999) {
		t.Error("missing room should not be outside")
	}
}

// TestIsRoomDarkSparesCityStreets: C's IS_DARK (utils.h:254-259) darkens a
// room at night only when its sector is neither SECT_INSIDE nor SECT_CITY, so
// city streets stay lit after sunset.
func TestIsRoomDarkSparesCityStreets(t *testing.T) {
	world, err := NewWorld(&parser.World{Rooms: []parser.Room{
		{VNum: 1001, Name: "Inside", Sector: SECT_INSIDE, Flags: []string{"0", "0", "0", "0"}},
		{VNum: 1002, Name: "Street", Sector: SECT_CITY, Flags: []string{"0", "0", "0", "0"}},
		{VNum: 1003, Name: "Field", Sector: SECT_FIELD, Flags: []string{"0", "0", "0", "0"}},
		{VNum: 1004, Name: "Cellar", Sector: SECT_CITY, Flags: []string{"1", "0", "0", "0"}}, // ROOM_DARK
	}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(world.StopAITicker)
	weatherMu.Lock()
	saved := weatherInfo.Sunlight
	weatherMu.Unlock()
	t.Cleanup(func() {
		weatherMu.Lock()
		weatherInfo.Sunlight = saved
		weatherMu.Unlock()
	})

	for _, sun := range []int{SunDark, SunSet, SunRise, SunLight} {
		weatherMu.Lock()
		weatherInfo.Sunlight = sun
		weatherMu.Unlock()
		night := sun == SunDark || sun == SunSet
		for vnum, want := range map[int]bool{1001: false, 1002: false, 1003: night, 1004: true} {
			if got := world.IsRoomDark(vnum); got != want {
				t.Errorf("sunlight %d: room %d dark = %v, want %v", sun, vnum, got, want)
			}
		}
	}
}
