package game

import (
	"bufio"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestTattooConstantsMatchCSource verifies that Go tattoo constants match
// the C #define values in src/structs.h. This catches silent drift where
// Go constants are changed but C source isn't updated, or vice versa.
//
// Source: src/structs.h lines 169-186
// #define TATTOO_NONE     0
// #define TATTOO_DRAGON   1
// ...
// #define TATTOO_OWL      17
func TestTattooConstantsMatchCSource(t *testing.T) {
	cDefines := parseCTattooDefines(t, "../../src/structs.h")
	if len(cDefines) == 0 {
		t.Fatal("no TATTOO_* defines found in C source — test data may be stale")
	}

	goConstants := map[string]int{
		"TATTOO_NONE":   TattooNone,
		"TATTOO_DRAGON": TattooDragon,
		"TATTOO_TRIBAL": TattooTribal,
		"TATTOO_SKULL":  TattooSkull,
		"TATTOO_TIGER":  TattooTiger,
		"TATTOO_WORM":   TattooWorm,
		"TATTOO_EYE":    TattooEye,
		"TATTOO_SWORDS": TattooSwords,
		"TATTOO_EAGLE":  TattooEagle,
		"TATTOO_HEART":  TattooHeart,
		"TATTOO_STAR":   TattooStar,
		"TATTOO_SHIP":   TattooShip,
		"TATTOO_SPIDER": TattooSpider,
		"TATTOO_JYHAD":  TattooJyhadi,
		"TATTOO_MOM":    TattooMom,
		"TATTOO_ANGEL":  TattooAngel,
		"TATTOO_FOX":    TattooFox,
		"TATTOO_OWL":    TattooOwl,
	}

	for name, cValue := range cDefines {
		goValue, ok := goConstants[name]
		if !ok {
			t.Errorf("C defines %s = %d but no matching Go constant exists", name, cValue)
			continue
		}
		if goValue != cValue {
			t.Errorf("%s: C = %d, Go = %d — constant drift detected", name, cValue, goValue)
		}
	}

	// Check for Go constants that don't exist in C (invented constants)
	for name := range goConstants {
		if _, ok := cDefines[name]; !ok {
			t.Errorf("Go defines %s but no matching C #define exists — invented constant?", name)
		}
	}
}

// parseCTattooDefines reads structs.h and extracts #define TATTOO_* values.
func parseCTattooDefines(t *testing.T, path string) map[string]int {
	t.Helper()

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("cannot open C source %s: %v", path, err)
	}
	defer f.Close()

	defines := make(map[string]int)
	// Match: #define TATTOO_NAME  123
	re := regexp.MustCompile(`^#define\s+(TATTOO_\w+)\s+(\d+)`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		name := matches[1]
		val, err := strconv.Atoi(matches[2])
		if err != nil {
			t.Errorf("cannot parse value for %s: %v", name, err)
			continue
		}
		defines[name] = val
	}
	return defines
}
