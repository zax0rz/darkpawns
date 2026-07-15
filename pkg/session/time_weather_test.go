package session

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zax0rz/darkpawns/pkg/game"
)

func TestFormatMudTimeHourWording(t *testing.T) {
	tests := []struct {
		hour int
		want string
	}{
		{0, "It is 12 o'clock am, on the Day of the Bull\r\n"},
		{1, "It is 1 o'clock am, on the Day of the Bull\r\n"},
		{11, "It is 11 o'clock am, on the Day of the Bull\r\n"},
		{12, "It is 12 o'clock pm, on the Day of the Bull\r\n"},
		{13, "It is 1 o'clock pm, on the Day of the Bull\r\n"},
		{23, "It is 11 o'clock pm, on the Day of the Bull\r\n"},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("hour %d", test.hour), func(t *testing.T) {
			snapshot := game.WorldClimateSnapshot{
				Time:    game.TimeInfoData{Hours: test.hour, Day: 0, Month: 0, Year: 1},
				Weather: game.WeatherData{Sunlight: game.SunLight},
			}
			want := test.want + "The 1st Day of the Month of Winter, Year 1.\r\n"
			if got := formatMudTime(snapshot); got != want {
				t.Errorf("formatMudTime(hour=%d) = %q, want %q", test.hour, got, want)
			}
		})
	}
}

func TestFormatMudTimeDateAndOrdinal(t *testing.T) {
	suffixes := map[int]string{
		0: "st", 1: "nd", 2: "rd", 3: "th", 10: "th", 11: "th", 12: "th",
		19: "th", 20: "st", 21: "nd", 22: "rd", 23: "th", 34: "th",
	}
	for day, want := range suffixes {
		if got := daySuffix(day); got != want {
			t.Errorf("daySuffix(%d) = %q, want %q", day, got, want)
		}
	}

	snapshot := game.WorldClimateSnapshot{
		Time:    game.TimeInfoData{Hours: 9, Day: 23, Month: 3, Year: 1260},
		Weather: game.WeatherData{Sunlight: game.SunLight},
	}
	want := "It is 9 o'clock am, on the Day of Thunder\r\n" +
		"The 24th Day of the Month of the Old Forces, Year 1260.\r\n"
	if got := formatMudTime(snapshot); got != want {
		t.Fatalf("fixed date = %q, want %q", got, want)
	}
}

func TestFormatMudTimeMoonOnlyWhenDark(t *testing.T) {
	for _, sunlight := range []int{game.SunRise, game.SunLight, game.SunSet} {
		snapshot := game.WorldClimateSnapshot{
			Time:    game.TimeInfoData{Moon: game.MoonHalfFull},
			Weather: game.WeatherData{Sunlight: sunlight},
		}
		if got := formatMudTime(snapshot); strings.Contains(got, "moon") {
			t.Errorf("sunlight %d unexpectedly rendered moon: %q", sunlight, got)
		}
	}

	snapshot := game.WorldClimateSnapshot{
		Time:    game.TimeInfoData{Moon: game.MoonHalfFull},
		Weather: game.WeatherData{Sunlight: game.SunDark},
	}
	if got := formatMudTime(snapshot); !strings.HasSuffix(got, "The moon is half full(waxing).\r\n") {
		t.Errorf("dark snapshot missing moon phase: %q", got)
	}
}

func TestFormatMudWeatherBranches(t *testing.T) {
	if got := formatMudWeather(game.WeatherData{}, false); got != "You have no feeling about the weather at all.\r\n" {
		t.Fatalf("indoor weather = %q", got)
	}

	skies := []struct {
		sky  int
		want string
	}{
		{game.SkyCloudless, "cloudless"},
		{game.SkyCloudy, "cloudy"},
		{game.SkyRaining, "rainy"},
		{game.SkyLightning, "lit by flashes of lightning"},
	}
	for _, sky := range skies {
		for _, change := range []int{-1, 0, 1} {
			direction := "you feel a warm wind from south"
			if change < 0 {
				direction = "the clouds tell you bad weather is due"
			}
			want := fmt.Sprintf("The sky is %s and %s.\r\n", sky.want, direction)
			if got := formatMudWeather(game.WeatherData{Sky: sky.sky, Change: change}, true); got != want {
				t.Errorf("sky %d change %d = %q, want %q", sky.sky, change, got, want)
			}
		}
	}
}

func TestTimeAndWeatherCommandsUseGameSnapshots(t *testing.T) {
	manager := makeTestManager(t)
	session := makeTestSession(t, manager, "Viewer", 1001, true)

	wantTime := formatMudTime(game.TimeWeatherSnapshot())
	if err := cmdTime(session, nil); err != nil {
		t.Fatalf("cmdTime: %v", err)
	}
	if got := readSessionText(t, session); got != wantTime {
		t.Errorf("cmdTime = %q, want canonical snapshot %q", got, wantTime)
	}

	room, ok := manager.world.GetRoom(1001)
	if !ok {
		t.Fatal("missing room 1001")
	}
	room.Sector = 0
	room.Flags = []string{"8"} // ROOM_INDOORS
	if err := cmdWeather(session, nil); err != nil {
		t.Fatalf("cmdWeather indoors: %v", err)
	}
	if got := readSessionText(t, session); got != "You have no feeling about the weather at all.\r\n" {
		t.Errorf("indoor cmdWeather = %q", got)
	}

	room.Flags = nil // C OUTSIDE: !ROOM_INDOORS even with SECT_INSIDE
	wantWeather := formatMudWeather(game.WeatherSnapshot(), true)
	if err := cmdWeather(session, nil); err != nil {
		t.Fatalf("cmdWeather outdoors: %v", err)
	}
	if got := readSessionText(t, session); got != wantWeather {
		t.Errorf("outdoor cmdWeather = %q, want %q", got, wantWeather)
	}
}
