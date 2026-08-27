package parser

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// ShopProto is the damage-gate-relevant subset of a CircleMUD shop record.
// The field order mirrors C boot_the_shops (src/shop.c:1145-1219): product
// list, buy/sell profits, trade-type list, seven message strings, temper,
// bitvector, keeper, trade-with, room list, open/close hours. Only the
// keeper vnum and the behavior bitvector (shop.h: WILL_START_FIGHT=1,
// WILL_BANK_MONEY=2) are kept — they decide mobprot is_shopkeeper membership
// and ok_damage_shopkeeper's slap-and-warn prelude (shop.c:1006-1023).
type ShopProto struct {
	VNum       int
	KeeperVNum int
	Bitvector  int
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
	nextIntList := func() error {
		for {
			value, err := nextInt()
			if err != nil {
				return err
			}
			if value < 0 {
				return nil
			}
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
	if err := nextIntList(); err != nil { // producing list
		return shop, err
	}
	if _, err := nextLine(); err != nil { // buy profit
		return shop, err
	}
	if _, err := nextLine(); err != nil { // sell profit
		return shop, err
	}
	if err := nextIntList(); err != nil { // trade types
		return shop, err
	}
	for range 7 { // no_such_item1/2, do_not_buy, missing_cash1/2, message_buy, message_sell
		if _, err := nextString(); err != nil {
			return shop, err
		}
	}
	if _, err := nextInt(); err != nil { // temper
		return shop, err
	}
	bitvector, err := nextInt()
	if err != nil {
		return shop, err
	}
	keeper, err := nextInt()
	if err != nil {
		return shop, err
	}
	shop.KeeperVNum = keeper
	shop.Bitvector = bitvector
	if _, err := nextInt(); err != nil { // trade with
		return shop, err
	}
	if err := nextIntList(); err != nil { // in-room list
		return shop, err
	}
	for range 4 { // open1, close1, open2, close2
		if _, err := nextInt(); err != nil {
			return shop, err
		}
	}
	return shop, nil
}
