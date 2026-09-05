package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ShopProto mirrors the boot-relevant fields of a CircleMUD shop record.
// The field order follows C boot_the_shops (src/shop.c:1145-1219): product
// list, buy/sell profits, trade-type list, seven message strings, temper,
// bitvector, keeper, trade-with, room list, and open/close hours. Keeping the
// command-facing fields here prevents the live shop path from silently losing
// parsed C data at the world bridge.
type ShopProto struct {
	VNum       int
	Products   []int
	BuyProfit  float64
	SellProfit float64
	BuyTypes   []int
	Messages   [7]string
	Temper     int
	Bitvector  int
	KeeperVNum int
	WithWho    int
	Rooms      []int
	OpenHour1  int
	CloseHour1 int
	OpenHour2  int
	CloseHour2 int
}

// ParseAllShopFiles reads the shop index and every .shp file it lists, the
// same way index_boot(DB_BOOT_SHOP) does on the C side.
func ParseAllShopFiles(dir string) ([]ShopProto, error) {
	indexData, err := os.ReadFile(filepath.Join(dir, "index")) // #nosec G304 -- fixed world-data path from the -world flag
	if err != nil {
		// A world tree without a shop index has no shops, exactly like C
		// refuses to boot without one; the harness copies lib/world wholesale.
		return nil, nil
	}
	var shops []ShopProto
	for _, field := range strings.Fields(string(indexData)) {
		if field == "$" {
			break
		}
		if !strings.HasSuffix(field, ".shp") {
			continue
		}
		path := filepath.Join(dir, field)
		parsed, parseErr := parseShopFile(path)
		if parseErr != nil {
			return nil, fmt.Errorf("parse %s: %w", path, parseErr)
		}
		shops = append(shops, parsed...)
	}
	return shops, nil
}

func parseShopFile(path string) ([]ShopProto, error) {
	data, err := os.ReadFile(path) // #nosec G703 -- filename comes from the shipped world index, resolved from the trusted -world dir
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")

	var shops []ShopProto
	i := 0
	for i < len(lines) {
		line := strings.TrimSpace(lines[i])
		i++
		if line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		if line == "$" || line == "$~" {
			break
		}
		if !strings.HasPrefix(line, "#") {
			continue
		}
		vnum, convErr := strconv.Atoi(strings.TrimSuffix(strings.TrimPrefix(line, "#"), "~"))
		if convErr != nil {
			continue
		}
		shop, recordErr := parseShopRecord(lines, &i, vnum, path)
		if recordErr != nil {
			return nil, recordErr
		}
		shops = append(shops, shop)
	}
	return shops, nil
}

// parseShopRecord consumes one shop record starting after the "#<vnum>~"
// header, following C's field order exactly.
func parseShopRecord(lines []string, i *int, vnum int, path string) (ShopProto, error) {
	nextLine := func() (string, error) {
		for *i < len(lines) {
			line := strings.TrimSpace(lines[*i])
			*i++
			if line != "" {
				return line, nil
			}
		}
		return "", fmt.Errorf("%s: shop #%d truncated", path, vnum)
	}
	nextInt := func() (int, error) {
		line, err := nextLine()
		if err != nil {
			return 0, err
		}
		// sscanf("%d") semantics: parse the leading integer and ignore the
		// rest of the line — trade-type entries carry a buy word ("10wheat").
		end := 0
		for end < len(line) && (line[end] == '-' || (line[end] >= '0' && line[end] <= '9')) {
			end++
		}
		value, convErr := strconv.Atoi(line[:end])
		if convErr != nil {
			return 0, fmt.Errorf("%s: shop #%d expected integer, got %q", path, vnum, line)
		}
		return value, nil
	}
	nextIntList := func() ([]int, error) {
		values := make([]int, 0)
		for {
			value, err := nextInt()
			if err != nil {
				return nil, err
			}
			if value < 0 {
				return values, nil
			}
			values = append(values, value)
		}
	}
	nextString := func() (string, error) {
		var builder strings.Builder
		for {
			line, err := nextLine()
			if err != nil {
				return "", err
			}
			if strings.HasSuffix(line, "~") {
				builder.WriteString(strings.TrimSuffix(line, "~"))
				return builder.String(), nil
			}
			builder.WriteString(line)
		}
	}

	shop := ShopProto{VNum: vnum}
	products, err := nextIntList()
	if err != nil { // producing list
		return shop, err
	}
	shop.Products = products
	buyProfit, err := nextLine()
	if err != nil { // buy profit
		return shop, err
	}
	shop.BuyProfit, err = strconv.ParseFloat(buyProfit, 64)
	if err != nil {
		return shop, fmt.Errorf("%s: shop #%d invalid buy profit %q: %w", path, vnum, buyProfit, err)
	}
	sellProfit, err := nextLine()
	if err != nil { // sell profit
		return shop, err
	}
	shop.SellProfit, err = strconv.ParseFloat(sellProfit, 64)
	if err != nil {
		return shop, fmt.Errorf("%s: shop #%d invalid sell profit %q: %w", path, vnum, sellProfit, err)
	}
	shop.BuyTypes, err = nextIntList() // trade types
	if err != nil {
		return shop, err
	}
	for index := range shop.Messages { // no_such_item1/2, do_not_buy, missing_cash1/2, message_buy, message_sell
		shop.Messages[index], err = nextString()
		if err != nil {
			return shop, err
		}
	}
	shop.Temper, err = nextInt() // temper
	if err != nil {
		return shop, err
	}
	shop.Bitvector, err = nextInt()
	if err != nil {
		return shop, err
	}
	shop.KeeperVNum, err = nextInt()
	if err != nil {
		return shop, err
	}
	shop.WithWho, err = nextInt() // trade with
	if err != nil {
		return shop, err
	}
	shop.Rooms, err = nextIntList() // in-room list
	if err != nil {
		return shop, err
	}
	shop.OpenHour1, err = nextInt()
	if err != nil {
		return shop, err
	}
	shop.CloseHour1, err = nextInt()
	if err != nil {
		return shop, err
	}
	shop.OpenHour2, err = nextInt()
	if err != nil {
		return shop, err
	}
	shop.CloseHour2, err = nextInt()
	if err != nil {
		return shop, err
	}
	return shop, nil
}
