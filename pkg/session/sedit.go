package session

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"github.com/zax0rz/darkpawns/pkg/game"
	"github.com/zax0rz/darkpawns/pkg/parser"
)

// seditMode mirrors the SEDIT_* values in src/olc.h. The holes are part of
// the C state machine, so keep the numerical values rather than renumbering
// the modes into a prettier Go enum.
type seditMode int

const (
	seditMainMenu          seditMode = 0
	seditConfirmSave       seditMode = 1
	seditNoItem1           seditMode = 2
	seditNoItem2           seditMode = 3
	seditNoCash1           seditMode = 4
	seditNoCash2           seditMode = 5
	seditNoBuy             seditMode = 6
	seditBuy               seditMode = 7
	seditSell              seditMode = 8
	seditProductsMenu      seditMode = 11
	seditRoomsMenu         seditMode = 12
	seditNamelistMenu      seditMode = 13
	seditNamelist          seditMode = 14
	seditNumericalResponse seditMode = 20
	seditOpen1             seditMode = 21
	seditOpen2             seditMode = 22
	seditClose1            seditMode = 23
	seditClose2            seditMode = 24
	seditKeeper            seditMode = 25
	seditBuyProfit         seditMode = 26
	seditSellProfit        seditMode = 27
	seditTypeMenu          seditMode = 29
	seditDeleteType        seditMode = 30
	seditDeleteProduct     seditMode = 31
	seditNewProduct        seditMode = 32
	seditDeleteRoom        seditMode = 33
	seditNewRoom           seditMode = 34
	seditShopFlags         seditMode = 35
	seditNoTrade           seditMode = 36
)

type seditState struct {
	shop          parser.ShopProto
	number        int
	zoneNumber    int
	isNew         bool
	mode          seditMode
	olcVal        int
	pendingOutput string
}

var (
	seditSaveMu    sync.Mutex
	seditSaveShops = make(map[int]bool)
)

var seditItemTypes = []string{
	"UNDEFINED", "LIGHT", "SCROLL", "WAND", "STAFF", "WEAPON",
	"FIRE WEAPON", "MISSILE", "TREASURE", "ARMOR", "POTION", "WORN",
	"OTHER", "TRASH", "TRAP", "CONTAINER", "NOTE", "LIQ CONTAINER",
	"KEY", "FOOD", "MONEY", "PEN", "BOAT", "FOUNTAIN",
}

var seditShopFlagNames = []string{"WILL_FIGHT", "USES_BANK"}

var seditTradeLetters = []string{
	"Good", "Evil", "Neutral", "Magic User", "Cleric", "Thief", "Warrior",
}

var seditMessageModes = map[seditMode]int{
	seditNoItem1: 0,
	seditNoItem2: 1,
	seditNoCash1: 3,
	seditNoCash2: 4,
	seditNoBuy:   2,
	seditBuy:     5,
	seditSell:    6,
}

// cmdSedit ports the SCMD_OLC_SEDIT branch of do_olc (src/olc.c). SEDIT
// works on shop VNUMs, not keeper VNUMs; a missing shop enters the new-shop
// defaults from sedit_setup_new (src/sedit.c:103-130).
func cmdSedit(s *Session, args []string) error {
	if s.player == nil || s.manager == nil || s.manager.world == nil {
		return fmt.Errorf("not logged in")
	}

	var buf1, buf2 string
	if len(args) > 0 {
		buf1 = args[0]
	}
	if len(args) > 1 {
		buf2 = args[1]
	}
	save := strings.HasPrefix(buf1, "save")

	if buf1 == "" {
		s.seditSend("Specify a shop VNUM to edit.\r\n")
		return nil
	}

	var number int
	if !isASCIIDigit(firstByte(buf1)) {
		if save {
			if buf2 == "" {
				s.seditSend("Save which zone?\r\n")
				return nil
			}
			number = atoiC(buf2) * 100
		} else {
			s.seditSend("Yikes!  Stop that, someone will get hurt!\r\n")
			return nil
		}
	} else {
		number = atoiC(buf1)
	}

	if save {
		if other := s.manager.shopEditHolder(number); other != "" {
			s.seditSend(fmt.Sprintf("That shop is currently being edited by %s\r\n", other))
			return nil
		}
		zone, ok := olcZoneForVNum(s.manager.world, number)
		if !ok {
			s.seditSend("Sorry, there is no zone for that number!\r\n")
			return nil
		}
		if !olcAuthorized(s, zone.Number) {
			s.seditSend("You do not have permission to edit this zone.\r\n")
			return nil
		}
		s.seditSend("Saving all shops in zone.\r\n")
		if err := saveSeditZone(s.manager.world, zone); err != nil {
			slog.Error("sedit disk save failed", "player", s.playerName, "zone", zone.Number, "error", err)
		}
		return nil
	}

	if other, ok := s.manager.claimShopEdit(number, s); !ok {
		s.seditSend(fmt.Sprintf("That shop is currently being edited by %s.\r\n", other))
		return nil
	}
	zone, ok := olcZoneForVNum(s.manager.world, number)
	if !ok {
		s.manager.releaseShopEdit(number, s)
		s.seditSend("Sorry, there is no zone for that number!\r\n")
		return nil
	}
	if !olcAuthorized(s, zone.Number) {
		s.manager.releaseShopEdit(number, s)
		s.seditSend("You do not have permission to edit this zone.\r\n")
		return nil
	}
	if err := s.startSedit(number, zone); err != nil {
		s.manager.releaseShopEdit(number, s)
		return err
	}
	return nil
}

func (m *Manager) claimShopEdit(number int, s *Session) (string, bool) {
	m.shopEditMu.Lock()
	defer m.shopEditMu.Unlock()
	if holder, ok := m.shopEdits[number]; ok && holder != s {
		name := "someone"
		if holder != nil && holder.playerName != "" {
			name = holder.playerName
		}
		return name, false
	}
	if m.shopEdits == nil {
		m.shopEdits = make(map[int]*Session)
	}
	m.shopEdits[number] = s
	return "", true
}

func (m *Manager) releaseShopEdit(number int, s *Session) {
	m.shopEditMu.Lock()
	defer m.shopEditMu.Unlock()
	if m.shopEdits[number] == s {
		delete(m.shopEdits, number)
	}
}

func (m *Manager) shopEditHolder(number int) string {
	m.shopEditMu.Lock()
	defer m.shopEditMu.Unlock()
	if holder, ok := m.shopEdits[number]; ok && holder != nil {
		return holder.playerName
	}
	return ""
}

func (s *Session) startSedit(number int, zone *parser.Zone) error {
	shop, exists := s.manager.world.SnapshotShop(number)
	var working parser.ShopProto
	if exists {
		working = seditShopProto(shop)
	} else {
		working = newSeditShop(number)
	}

	s.textEditMu.Lock()
	if s.sedit != nil {
		s.finishSeditLocked(false)
	}
	s.sedit = &seditState{
		shop:       working,
		number:     number,
		zoneNumber: zone.Number,
		isNew:      !exists,
		mode:       seditMainMenu,
	}
	s.setPlayerWritingLocked(true)
	s.seditShowMenuLocked()
	s.flushSeditOutputLocked()
	s.textEditMu.Unlock()

	if s.player != nil {
		game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n starts using OLC.", "", game.ToRoom)
	}
	return nil
}

func newSeditShop(number int) parser.ShopProto {
	return parser.ShopProto{
		VNum:       number,
		KeeperVNum: -1,
		BuyProfit:  1.0,
		SellProfit: 1.0,
		Messages: [7]string{
			"%s Sorry, I don't stock that item.",
			"%s You don't seem to have that.",
			"%s I don't trade in such items.",
			"%s I can't afford that!",
			"%s You are too poor!",
			"%s That'll be %d coins, thanks.",
			"%s I'll give you %d coins for that.",
		},
		CloseHour1: 28,
	}
}

func seditShopProto(shop game.Shop) parser.ShopProto {
	return parser.ShopProto{
		VNum:       shop.VNum,
		Products:   append([]int(nil), shop.SellTypes...),
		BuyProfit:  shop.ProfitBuy,
		SellProfit: shop.ProfitSell,
		BuyTypes:   append([]int(nil), shop.BuyTypes...),
		BuyWords:   append([]string(nil), shop.BuyWords...),
		Messages:   shop.Messages,
		Temper:     shop.Temper,
		Bitvector:  shop.Flags,
		KeeperVNum: shop.KeeperVNum,
		WithWho:    shop.WithWho,
		Rooms:      append([]int(nil), shop.Rooms...),
		OpenHour1:  shop.OpenHour1,
		CloseHour1: shop.CloseHour1,
		OpenHour2:  shop.OpenHour2,
		CloseHour2: shop.CloseHour2,
	}
}

func (s *Session) seditSend(text string) {
	if s.sedit != nil {
		s.sedit.pendingOutput += text
		return
	}
	if err := s.SendMessage(text); err != nil {
		slog.Error("sedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) seditSendLocked(text string) { s.seditSend(text) }

func (s *Session) flushSeditOutputLocked() {
	if s.sedit == nil || s.sedit.pendingOutput == "" {
		return
	}
	text := s.sedit.pendingOutput
	s.sedit.pendingOutput = ""
	if err := s.SendMessage(text); err != nil {
		slog.Error("sedit output failed", "player", s.playerName, "error", err)
	}
}

func (s *Session) finishSeditLocked(save bool) {
	state := s.sedit
	if state == nil {
		return
	}
	if save {
		state.shop.VNum = state.number
		saveMu := zoneSaveLock(state.zoneNumber)
		saveMu.Lock()
		if s.manager.world.CommitEditedShop(state.shop) {
			seditSaveMu.Lock()
			seditSaveShops[state.number] = true
			seditSaveMu.Unlock()
		}
		saveMu.Unlock()
	}
	s.flushSeditOutputLocked()
	s.setPlayerWritingLocked(false)
	if s.player != nil {
		game.Act(s.manager.world, false, s.player, nil, nil, nil, "$n stops using OLC.", "", game.ToRoom)
	}
	s.sedit = nil
	s.manager.releaseShopEdit(state.number, s)
}

func (s *Session) cancelSedit() {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.sedit != nil {
		s.finishSeditLocked(false)
	}
}

// IsSeditEditing reports whether CON_SEDIT owns this descriptor.
func (s *Session) IsSeditEditing() bool { return s.isSeditEditing() }

func (s *Session) isSeditEditing() bool {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	return s.sedit != nil
}

func (s *Session) handleSeditInput(line string) {
	s.textEditMu.Lock()
	defer s.textEditMu.Unlock()
	if s.sedit == nil {
		return
	}
	s.parseSeditLocked(strings.TrimLeft(line, " \t\r\n\v\f"))
	s.flushSeditOutputLocked()
}

func (s *Session) seditShowMenuLocked() {
	state := s.sedit
	if state == nil {
		return
	}
	shop := &state.shop
	nrm, grn, cyn, yel := s.reditCols()
	keeperVNum := shop.KeeperVNum
	keeperName := "None"
	if keeperVNum != -1 {
		if mob, ok := s.manager.world.SnapshotMob(keeperVNum); ok {
			keeperName = mob.ShortDesc
		}
	}
	var notrade, flags string
	notrade = seditSprintBits(shop.WithWho, seditTradeLetters)
	flags = seditSprintBits(shop.Bitvector, seditShopFlagNames)

	var out strings.Builder
	fmt.Fprintf(&out, "\r\n-- Shop Number : [%s%d%s]\r\n", cyn, state.number, nrm)
	fmt.Fprintf(&out, "%s0%s) Keeper      : [%s%d%s] %s%s\r\n", grn, nrm, cyn, keeperVNum, nrm, yel, keeperName)
	fmt.Fprintf(&out, "%s1%s) Open 1      : %s%4d%s          %s2%s) Close 1     : %s%4d\r\n", grn, nrm, cyn, shop.OpenHour1, nrm, grn, nrm, cyn, shop.CloseHour1)
	fmt.Fprintf(&out, "%s3%s) Open 2      : %s%4d%s          %s4%s) Close 2     : %s%4d\r\n", grn, nrm, cyn, shop.OpenHour2, nrm, grn, nrm, cyn, shop.CloseHour2)
	fmt.Fprintf(&out, "%s5%s) Sell rate   : %s%1.2f%s          %s6%s) Buy rate    : %s%1.2f\r\n", grn, nrm, cyn, shop.BuyProfit, nrm, grn, nrm, cyn, shop.SellProfit)
	fmt.Fprintf(&out, "%s7%s) Keeper no item : %s%s\r\n", grn, nrm, yel, shop.Messages[0])
	fmt.Fprintf(&out, "%s8%s) Player no item : %s%s\r\n", grn, nrm, yel, shop.Messages[1])
	fmt.Fprintf(&out, "%s9%s) Keeper no cash : %s%s\r\n", grn, nrm, yel, shop.Messages[3])
	fmt.Fprintf(&out, "%sA%s) Player no cash : %s%s\r\n", grn, nrm, yel, shop.Messages[4])
	fmt.Fprintf(&out, "%sB%s) Keeper no buy  : %s%s\r\n", grn, nrm, yel, shop.Messages[2])
	fmt.Fprintf(&out, "%sC%s) Buy sucess     : %s%s\r\n", grn, nrm, yel, shop.Messages[5])
	fmt.Fprintf(&out, "%sD%s) Sell sucess    : %s%s\r\n", grn, nrm, yel, shop.Messages[6])
	fmt.Fprintf(&out, "%sE%s) No Trade With  : %s%s\r\n", grn, nrm, cyn, notrade)
	fmt.Fprintf(&out, "%sF%s) Shop flags     : %s%s\r\n", grn, nrm, cyn, flags)
	fmt.Fprintf(&out, "%sR%s) Rooms Menu\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sP%s) Products Menu\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sT%s) Accept Types Menu\r\n", grn, nrm)
	fmt.Fprintf(&out, "%sQ%s) Quit\r\nEnter Choice : ", grn, nrm)
	s.seditSendLocked(out.String())
	state.mode = seditMainMenu
}

func seditSprintBits(bits int, names []string) string {
	var out strings.Builder
	for i, name := range names {
		if bits&(1<<uint(i)) != 0 {
			out.WriteString(name)
			out.WriteByte(' ')
		}
	}
	if out.Len() == 0 {
		return "NOBITS "
	}
	return out.String()
}

func (s *Session) seditShowProductsMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, yel := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n##     VNUM     Product\r\n")
	for i, vnum := range state.shop.Products {
		short := ""
		if obj, ok := s.manager.world.SnapshotObj(vnum); ok {
			short = obj.ShortDesc
		}
		fmt.Fprintf(&out, "%2d - [%s%5d%s] - %s%s%s\r\n", i, cyn, vnum, nrm, yel, short, nrm)
	}
	fmt.Fprintf(&out, "\r\n%sA%s) Add a new product.\r\n%sD%s) Delete a product.\r\n%sQ%s) Quit\r\nEnter choice : ", grn, nrm, grn, nrm, grn, nrm)
	s.seditSendLocked(out.String())
	state.mode = seditProductsMenu
}

func (s *Session) seditShowCompactRoomsMenuLocked() {
	state := s.sedit
	nrm, _, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, vnum := range state.shop.Rooms {
		line := fmt.Sprintf("%2d - [%s%5d%s]  | ", i, cyn, vnum, nrm)
		if (i+1)%5 == 0 {
			line = line[:len(line)-3] + "\r\n"
		}
		out.WriteString(line)
	}
	nrm, grn, _, _ := s.reditCols()
	fmt.Fprintf(&out, "\r\n%sA%s) Add a new room.\r\n%sD%s) Delete a room.\r\n%sL%s) Long display.\r\n%sQ%s) Quit\r\nEnter choice : ", grn, nrm, grn, nrm, grn, nrm, grn, nrm)
	s.seditSendLocked(out.String())
	state.mode = seditRoomsMenu
}

func (s *Session) seditShowRoomsMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, yel := s.reditCols()
	var out strings.Builder
	out.WriteString("\x1b[H\x1b[J##     VNUM     Room\r\n\r\n")
	for i, vnum := range state.shop.Rooms {
		name := ""
		if room, ok := s.manager.world.SnapshotRoom(vnum); ok {
			name = room.Name
		}
		fmt.Fprintf(&out, "%2d - [%s%5d%s] - %s%s%s\r\n", i, cyn, vnum, nrm, yel, name, nrm)
	}
	fmt.Fprintf(&out, "\r\n%sA%s) Add a new room.\r\n%sD%s) Delete a room.\r\n%sC%s) Compact Display.\r\n%sQ%s) Quit\r\nEnter choice : ", grn, nrm, grn, nrm, grn, nrm, grn, nrm)
	s.seditSendLocked(out.String())
	state.mode = seditRoomsMenu
}

func (s *Session) seditShowNamelistMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, yel := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n##              Type   Namelist\r\n\r\n")
	for i, typ := range state.shop.BuyTypes {
		name := seditItemName(typ)
		word := "<None>"
		if i < len(state.shop.BuyWords) && state.shop.BuyWords[i] != "" {
			word = state.shop.BuyWords[i]
		}
		fmt.Fprintf(&out, "%2d - %s%15s%s - %s%s%s\r\n", i, cyn, name, nrm, yel, word, nrm)
	}
	fmt.Fprintf(&out, "\r\n%sA%s) Add a new entry.\r\n%sD%s) Delete an entry.\r\n%sQ%s) Quit\r\nEnter choice : ", grn, nrm, grn, nrm, grn, nrm)
	s.seditSendLocked(out.String())
	state.mode = seditNamelistMenu
}

func (s *Session) seditShowShopFlagsMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, name := range seditShopFlagNames {
		fmt.Fprintf(&out, "%s%2d%s) %-20.20s   ", grn, i+1, nrm, name)
		if (i+1)%2 == 0 {
			out.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&out, "\r\nCurrent Shop Flags : %s%s%s\r\nEnter choice : ", cyn, seditSprintBits(state.shop.Bitvector, seditShopFlagNames), nrm)
	s.seditSendLocked(out.String())
	state.mode = seditShopFlags
}

func (s *Session) seditShowNoTradeMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, name := range seditTradeLetters {
		fmt.Fprintf(&out, "%s%2d%s) %-20.20s   ", grn, i+1, nrm, name)
		if (i+1)%2 == 0 {
			out.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&out, "\r\nCurrently won't trade with: %s%s%s\r\nEnter choice : ", cyn, seditSprintBits(state.shop.WithWho, seditTradeLetters), nrm)
	s.seditSendLocked(out.String())
	state.mode = seditNoTrade
}

func (s *Session) seditShowTypesMenuLocked() {
	state := s.sedit
	nrm, grn, cyn, _ := s.reditCols()
	var out strings.Builder
	out.WriteString("\r\n")
	for i, name := range seditItemTypes {
		fmt.Fprintf(&out, "%s%2d%s) %s%-20s%s  ", grn, i, nrm, cyn, name, nrm)
		if (i+1)%3 == 0 {
			out.WriteString("\r\n")
		}
	}
	fmt.Fprintf(&out, "%sEnter choice : ", nrm)
	s.seditSendLocked(out.String())
	state.mode = seditTypeMenu
}

func seditItemName(index int) string {
	if index >= 0 && index < len(seditItemTypes) {
		return seditItemTypes[index]
	}
	return "UNDEFINED"
}

func (s *Session) parseSeditLocked(arg string) {
	state := s.sedit
	if state == nil {
		return
	}
	// This is deliberately the C typo at sedit.c:820-825. Unlike the sibling
	// editors it has no !*arg clause, so an empty line reaches atoi("") == 0.
	if state.mode > seditNumericalResponse && !isASCIIDigit(firstByte(arg)) && firstByte(arg) == '-' && !isASCIIDigit(seditByteAt(arg, 1)) {
		s.seditSend("Field must be numerical, try again : ")
		return
	}

	switch state.mode {
	case seditConfirmSave:
		s.parseSeditConfirmLocked(arg)
		return
	case seditMainMenu:
		s.parseSeditMainMenuLocked(arg)
		return
	case seditNamelistMenu:
		s.parseSeditNamelistMenuLocked(arg)
		return
	case seditProductsMenu:
		s.parseSeditProductsMenuLocked(arg)
		return
	case seditRoomsMenu:
		s.parseSeditRoomsMenuLocked(arg)
		return
	case seditNoItem1, seditNoItem2, seditNoCash1, seditNoCash2, seditNoBuy, seditBuy, seditSell:
		index := seditMessageModes[state.mode]
		if arg != "" && firstByte(arg) != '%' {
			arg = "%s " + arg
		} else if arg == "" {
			arg = "%s "
		}
		state.shop.Messages[index] = arg
	case seditNamelist:
		state.shop.BuyTypes = append(state.shop.BuyTypes, state.olcVal)
		state.shop.BuyWords = append(state.shop.BuyWords, arg)
		s.seditShowNamelistMenuLocked()
		return
	case seditKeeper:
		if !s.parseSeditKeeperLocked(arg) {
			return
		}
	case seditOpen1:
		state.shop.OpenHour1 = clampSeditHour(atoiC(arg))
	case seditOpen2:
		state.shop.OpenHour2 = clampSeditHour(atoiC(arg))
	case seditClose1:
		state.shop.CloseHour1 = clampSeditHour(atoiC(arg))
	case seditClose2:
		state.shop.CloseHour2 = clampSeditHour(atoiC(arg))
	case seditBuyProfit:
		if value, ok := parseSeditFloat(arg); ok {
			state.shop.BuyProfit = value
		}
	case seditSellProfit:
		if value, ok := parseSeditFloat(arg); ok {
			state.shop.SellProfit = value
		}
	case seditTypeMenu:
		state.olcVal = clampInt(atoiC(arg), 0, len(seditItemTypes)-1)
		s.seditSend("Enter namelist (return for none) :-\r\n| ")
		state.mode = seditNamelist
		return
	case seditDeleteType:
		seditRemoveBuyType(&state.shop, atoiC(arg))
		s.seditShowNamelistMenuLocked()
		return
	case seditNewProduct:
		if !s.seditAddProductLocked(atoiC(arg)) {
			return
		}
		s.seditShowProductsMenuLocked()
		return
	case seditDeleteProduct:
		removeSeditInt(&state.shop.Products, atoiC(arg))
		s.seditShowProductsMenuLocked()
		return
	case seditNewRoom:
		if !s.seditAddRoomLocked(atoiC(arg)) {
			return
		}
		s.seditShowRoomsMenuLocked()
		return
	case seditDeleteRoom:
		removeSeditInt(&state.shop.Rooms, atoiC(arg))
		s.seditShowRoomsMenuLocked()
		return
	case seditShopFlags:
		if s.seditToggleFlagLocked(&state.shop.Bitvector, atoiC(arg), len(seditShopFlagNames)) {
			s.seditShowShopFlagsMenuLocked()
			return
		}
	case seditNoTrade:
		if s.seditToggleFlagLocked(&state.shop.WithWho, atoiC(arg), len(seditTradeLetters)) {
			s.seditShowNoTradeMenuLocked()
			return
		}
	default:
		s.finishSeditLocked(false)
		return
	}

	state.olcVal = 1
	s.seditShowMenuLocked()
}

func (s *Session) parseSeditConfirmLocked(arg string) {
	switch firstByte(arg) {
	case 'y', 'Y':
		s.seditSend("Saving shop to memory.\r\n")
		s.finishSeditLocked(true)
	case 'n', 'N':
		s.finishSeditLocked(false)
	default:
		s.seditSend("Invalid choice!\r\nDo you wish to save the shop? : ")
	}
}

func (s *Session) parseSeditMainMenuLocked(arg string) {
	state := s.sedit
	choice := firstByte(arg)
	var mode seditMode
	changed := 0
	switch choice {
	case 'q', 'Q':
		if state.olcVal != 0 {
			state.mode = seditConfirmSave
			s.seditSend("Do you wish to save the changes to the shop? (y/n) : ")
		} else {
			s.finishSeditLocked(false)
		}
		return
	case '0':
		state.mode = seditKeeper
		s.seditSend("Enter virtual number of shop keeper : ")
		return
	case '1':
		mode, changed = seditOpen1, 1
	case '2':
		mode, changed = seditClose1, 1
	case '3':
		mode, changed = seditOpen2, 1
	case '4':
		mode, changed = seditClose2, 1
	case '5':
		mode, changed = seditBuyProfit, 1
	case '6':
		mode, changed = seditSellProfit, 1
	case '7':
		mode, changed = seditNoItem1, -1
	case '8':
		mode, changed = seditNoItem2, -1
	case '9':
		mode, changed = seditNoCash1, -1
	case 'a', 'A':
		mode, changed = seditNoCash2, -1
	case 'b', 'B':
		mode, changed = seditNoBuy, -1
	case 'c', 'C':
		mode, changed = seditBuy, -1
	case 'd', 'D':
		mode, changed = seditSell, -1
	case 'e', 'E':
		s.seditShowNoTradeMenuLocked()
		return
	case 'f', 'F':
		s.seditShowShopFlagsMenuLocked()
		return
	case 'r', 'R':
		s.seditShowRoomsMenuLocked()
		return
	case 'p', 'P':
		s.seditShowProductsMenuLocked()
		return
	case 't', 'T':
		s.seditShowNamelistMenuLocked()
		return
	default:
		s.seditShowMenuLocked()
		return
	}

	state.mode = mode
	if changed == 1 {
		s.seditSend("\r\nEnter new value : ")
	} else {
		s.seditSend("\r\nEnter new text :\r\n| ")
	}
}

func (s *Session) parseSeditNamelistMenuLocked(arg string) {
	switch firstByte(arg) {
	case 'a', 'A':
		s.seditShowTypesMenuLocked()
		return
	case 'd', 'D':
		s.seditSend("\r\nDelete which entry? : ")
		s.sedit.smode(seditDeleteType)
		return
	case 'q', 'Q':
		// Fall through to C's common dirty-menu redraw.
	}
	s.sedit.olcVal = 1
	s.seditShowMenuLocked()
}

func (s *Session) parseSeditProductsMenuLocked(arg string) {
	switch firstByte(arg) {
	case 'a', 'A':
		s.seditSend("\r\nEnter new product virtual number : ")
		s.sedit.mode = seditNewProduct
		return
	case 'd', 'D':
		s.seditSend("\r\nDelete which product? : ")
		s.sedit.mode = seditDeleteProduct
		return
	case 'q', 'Q':
	}
	s.sedit.olcVal = 1
	s.seditShowMenuLocked()
}

func (s *Session) parseSeditRoomsMenuLocked(arg string) {
	switch firstByte(arg) {
	case 'a', 'A':
		s.seditSend("\r\nEnter new room virtual number, \r\nor 1 to make the keeper sell anywhere : ")
		s.sedit.mode = seditNewRoom
		return
	case 'c', 'C':
		s.seditShowCompactRoomsMenuLocked()
		return
	case 'l', 'L':
		s.seditShowRoomsMenuLocked()
		return
	case 'd', 'D':
		s.seditSend("\r\nDelete which room? : ")
		s.sedit.mode = seditDeleteRoom
		return
	case 'q', 'Q':
	}
	s.sedit.olcVal = 1
	s.seditShowMenuLocked()
}

func (s *Session) parseSeditKeeperLocked(arg string) bool {
	number := atoiC(arg)
	if number != -1 {
		if _, ok := s.manager.world.SnapshotMob(number); !ok {
			s.seditSend("That mobile does not exist, try again : ")
			return false
		}
	}
	s.sedit.shop.KeeperVNum = number
	return true
}

func (s *Session) seditAddProductLocked(vnum int) bool {
	if vnum != -1 {
		if _, ok := s.manager.world.SnapshotObj(vnum); !ok {
			s.seditSend("That object does not exist, try again : ")
			return false
		}
	}
	if vnum >= 0 {
		s.sedit.shop.Products = append(s.sedit.shop.Products, vnum)
	}
	return true
}

func (s *Session) seditAddRoomLocked(vnum int) bool {
	if vnum != -1 {
		if _, ok := s.manager.world.SnapshotRoom(vnum); !ok {
			s.seditSend("That room does not exist, try again : ")
			return false
		}
	}
	if vnum >= 0 {
		s.sedit.shop.Rooms = append(s.sedit.shop.Rooms, vnum)
	}
	return true
}

func (s *Session) seditToggleFlagLocked(target *int, value, limit int) bool {
	value = clampInt(value, 0, limit)
	if value <= 0 {
		return false
	}
	bit := 1 << uint(value-1)
	*target ^= bit
	return true
}

func seditRemoveBuyType(shop *parser.ShopProto, index int) {
	if index < 0 || index >= len(shop.BuyTypes) {
		return
	}
	copy(shop.BuyTypes[index:], shop.BuyTypes[index+1:])
	shop.BuyTypes = shop.BuyTypes[:len(shop.BuyTypes)-1]
	if index < len(shop.BuyWords) {
		copy(shop.BuyWords[index:], shop.BuyWords[index+1:])
		shop.BuyWords = shop.BuyWords[:len(shop.BuyWords)-1]
	}
}

func removeSeditInt(values *[]int, index int) {
	if index < 0 || index >= len(*values) {
		return
	}
	copy((*values)[index:], (*values)[index+1:])
	*values = (*values)[:len(*values)-1]
}

func clampSeditHour(value int) int { return clampInt(value, 0, 28) }

func parseSeditFloat(input string) (float64, bool) {
	input = strings.TrimSpace(input)
	// C's sscanf("%f") accepts the longest valid numeric prefix and ignores
	// trailing bytes. Try progressively shorter prefixes to preserve that
	// behavior while retaining the float32 rounding of the C destination.
	for end := len(input); end > 0; end-- {
		value, err := strconv.ParseFloat(input[:end], 32)
		if err == nil {
			return value, true
		}
	}
	return 0, false
}

func seditByteAt(input string, index int) byte {
	if index < 0 || index >= len(input) {
		return 0
	}
	return input[index]
}

// saveSeditZone writes the C .shp format from a VNUM-ordered world snapshot.
func saveSeditZone(world *game.World, zone *parser.Zone) error {
	parsed := world.GetParsedWorld()
	if parsed == nil || parsed.SourceDir == "" {
		return fmt.Errorf("world has no source directory")
	}
	libDir := filepath.Join(parsed.SourceDir, "shp")
	if err := os.MkdirAll(libDir, 0o755); err != nil {
		return err
	}
	saveMu := zoneSaveLock(zone.Number)
	saveMu.Lock()
	defer saveMu.Unlock()

	shops := world.SnapshotShops()
	var out strings.Builder
	out.WriteString("CircleMUD v3.0 Shop File~\n")
	for i := range shops {
		shop := &shops[i]
		if shop.VNum < zone.Number*100 || shop.VNum > zone.TopRoom {
			continue
		}
		writeSeditShop(&out, shop)
	}
	out.WriteString("$~\n")
	path := filepath.Join(libDir, fmt.Sprintf("%d.shp", zone.Number))
	if err := atomicWriteFile(path, []byte(out.String()), 0o666); err != nil {
		return err
	}
	seditSaveMu.Lock()
	for i := range shops {
		if shops[i].VNum >= zone.Number*100 && shops[i].VNum <= zone.TopRoom {
			delete(seditSaveShops, shops[i].VNum)
		}
	}
	seditSaveMu.Unlock()
	return nil
}

func writeSeditShop(out *strings.Builder, shop *game.Shop) {
	fmt.Fprintf(out, "#%d~\n", shop.VNum)
	for _, vnum := range shop.SellTypes {
		fmt.Fprintf(out, "%d\n", vnum)
	}
	out.WriteString("-1\n")
	fmt.Fprintf(out, "%1.2f\n%1.2f\n", shop.ProfitBuy, shop.ProfitSell)
	for i, typ := range shop.BuyTypes {
		word := ""
		if i < len(shop.BuyWords) {
			word = shop.BuyWords[i]
		}
		fmt.Fprintf(out, "%d%s\n", typ, word)
	}
	out.WriteString("-1\n")
	message := func(index int, fallback string) string {
		if shop.Messages[index] != "" {
			return shop.Messages[index]
		}
		return fallback
	}
	fmt.Fprintf(out, "%s~\n%s~\n%s~\n%s~\n%s~\n%s~\n%s~\n",
		message(0, "%s Ke?!"),
		message(1, "%s Ke?!"),
		message(2, "%s Ke?!"),
		message(3, "%s Ke?!"),
		message(4, "%s Ke?!"),
		message(5, "%s Ke?! %d?"),
		message(6, "%s Ke?! %d?"),
	)
	fmt.Fprintf(out, "%d\n%d\n%d\n%d\n", shop.Temper, shop.Flags, shop.KeeperVNum, shop.WithWho)
	for _, vnum := range shop.Rooms {
		fmt.Fprintf(out, "%d\n", vnum)
	}
	out.WriteString("-1\n")
	fmt.Fprintf(out, "%d\n%d\n%d\n%d\n", shop.OpenHour1, shop.CloseHour1, shop.OpenHour2, shop.CloseHour2)
}

// smode is a tiny helper used only to keep the state mutation at the same
// call site as the C menu prompt in parseSeditNamelistMenuLocked.
func (state *seditState) smode(mode seditMode) { state.mode = mode }
