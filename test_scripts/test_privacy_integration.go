//go:build ignore

package testscripts

import (
	"fmt"
	"log"
	"os"

	"github.com/zax0rz/darkpawns/pkg/privacy"
)

func testPrivacyIntegration() {
	fmt.Println("Testing Privacy Filter Integration for Dark Pawns")
	fmt.Println("================================================")

	// Test 1: Configuration loading
	fmt.Println("\n1. Testing configuration loading...")
	config := privacy.LoadConfig()
	fmt.Printf("   URL: %s\n", config.URL)
	fmt.Printf("   Enabled: %v\n", config.Enabled)
	fmt.Printf("   Categories: %v\n", config.Categories)
	fmt.Printf("   Filter Player Names: %v\n", config.FilterPlayerNames)

	// Test 2: Client creation
	fmt.Println("\n2. Testing client creation...")
	filterConfig := config.ToFilterConfig()
	client := privacy.NewClient(config.URL, filterConfig)
	fmt.Println("   Client created successfully")

	// Test 3: Basic filtering (with fallback since service likely not running)
	fmt.Println("\n3. Testing text filtering...")
	testTexts := []string{
		"Player John Doe logged in from New York",
		"Contact email: john@example.com, phone: 555-123-4567",
		"Credit card: 4111-1111-1111-1111, expiry: 12/25",
		"Meeting on 2024-12-25 at 123 Main St, Apt 4B",
	}

	for i, text := range testTexts {
		filtered, detected, err := client.FilterText(text)
		if err != nil {
			fmt.Printf("   Test %d Error: %v\n", i+1, err)
		} else {
			fmt.Printf("   Test %d:\n", i+1)
			fmt.Printf("     Original: %s\n", text)
			fmt.Printf("     Filtered: %s\n", filtered)
			fmt.Printf("     Detected: %v\n", detected)
		}
	}

	// Test 4: Logger integration
	fmt.Println("\n4. Testing logger integration...")
	logger := privacy.NewPrivacyLogger(client, "[TEST] ", log.LstdFlags)
	logger.Println("Test log: Player Jane Smith (jane@company.com) purchased item #123")

	// Test 5: Batch filtering
	fmt.Println("\n5. Testing batch filtering...")
	filteredTexts, allDetected, err := client.BatchFilter(testTexts)
	if err != nil {
		fmt.Printf("   Batch filter error: %v\n", err)
	} else {
		fmt.Printf("   Processed %d texts\n", len(filteredTexts))
		for i, text := range filteredTexts {
			fmt.Printf("   Text %d: %s (detected: %v)\n", i+1,
				shorten(text, 50), allDetected[i])
		}
	}

	// Test 6: Environment variable override
	fmt.Println("\n6. Testing environment variable override...")
// #nosec G104
	os.Setenv("PRIVACY_FILTER_CATEGORIES", "email,phone")
// #nosec G104
	os.Setenv("FILTER_PLAYER_NAMES", "false")

	config2 := privacy.LoadConfig()
	fmt.Printf("   Overridden categories: %v\n", config2.Categories)
	fmt.Printf("   Filter Player Names: %v\n", config2.FilterPlayerNames)

	os.Unsetenv("PRIVACY_FILTER_CATEGORIES")
	os.Unsetenv("FILTER_PLAYER_NAMES")

	fmt.Println("\n================================================")
	fmt.Println("Integration test complete!")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Start privacy filter service: make privacy-up")
	fmt.Println("2. Run tests with service: make privacy-test")
	fmt.Println("3. Integrate into server (see pkg/privacy/integration_example.go)")
}

func shorten(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
