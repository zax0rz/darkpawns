// Package game — ported from src/mail.c / src/mail.h.
// Mud Mail System: file-backed mail storage with postmaster mob special.
//
// Source: mail.c and mail.h by Jeremy Elson (jelson@cs.jhu.edu)

package game

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/zax0rz/darkpawns/pkg/parser"
)

// =========================================================================
// Constants from mail.h
// =========================================================================

const (
	MinMailLevel = 2    // minimum level to send mail
	StampPrice   = 25   // gold cost to send mail
	MaxMailSize  = 4096 // maximum message size in bytes
	BlockSize    = 100  // allocation block size

	HeaderBlock  int8 = -1 // block type: header
	LastBlock    int8 = -2 // block type: last data block in a message
	DeletedBlock int8 = -3 // block type: deleted/free block
)

// HEADER_BLOCK_DATASIZE = BLOCK_SIZE - sizeof(long) - sizeof(header_data_type) - sizeof(char)
// = 100 - 8 - 32 - 1 = 59
const HeaderBlockDataSize = 59

// DATA_BLOCK_DATASIZE = BLOCK_SIZE - sizeof(long) - sizeof(char)
// = 100 - 8 - 1 = 91
const DataBlockDataSize = 91

// MailFilePath is the path to the on-disk mail store.
var MailFilePath = "mail"

// =========================================================================
// Data structures
// =========================================================================

// mailBlock is a fixed-size on-disk block: 100 bytes, matching BLOCK_SIZE.
// Byte layout (all int64 are little-endian):
//
//	[0:8]   block_type (int8 padded to 8 bytes)
//	[8:16]  header_data.next_block  (int64)
//	[16:24] header_data.from        (int64)
//	[24:32] header_data.to          (int64)
//	[32:40] header_data.mail_time   (int64)
//	[40:99] txt (up to 59 bytes for header, null-terminated)
//
// For data blocks the layout is simpler:
//	[0:8]   block_type (link to next block or LAST_BLOCK)
//	[8:99]  txt (up to 91 bytes)
type mailBlock [BlockSize]byte

// MailIndexType is a linked list node mapping a recipient to their mail positions.
type MailIndexType struct {
	Recipient int64
	ListStart *PositionListType
	Next      *MailIndexType
}

// PositionListType is a linked list node for free block positions.
type PositionListType struct {
	Position int64
	Next     *PositionListType
}

// MailPendingTo holds the recipient id that a player is composing mail to.
var MailPendingTo = make(map[int]int64)

// MailBuffer holds the mail text being composed.
var MailBuffer = make(map[int]string)

// =========================================================================
// Mail system state
// =========================================================================

var (
	mailIndex  *MailIndexType
	freeList   *PositionListType
	fileEndPos int64
	noMail     bool
	mailMu     sync.Mutex
)

// =========================================================================
// Block encoding helpers
// =========================================================================

func (b *mailBlock) setBlockType(bt int8)   { b[0] = byte(bt) }
func (b *mailBlock) blockType() int8        { return int8(b[0]) }

func readLE64(buf []byte) int64 {
	return int64(buf[0]) | int64(buf[1])<<8 | int64(buf[2])<<16 | int64(buf[3])<<24 |
		int64(buf[4])<<32 | int64(buf[5])<<40 | int64(buf[6])<<48 | int64(buf[7])<<56
}

func writeLE64(buf []byte, v int64) {
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	buf[2] = byte(v >> 16)
	buf[3] = byte(v >> 24)
	buf[4] = byte(v >> 32)
	buf[5] = byte(v >> 40)
	buf[6] = byte(v >> 48)
	buf[7] = byte(v >> 56)
}

func (b *mailBlock) setHeaderNextBlock(v int64) { writeLE64(b[8:], v) }
func (b *mailBlock) headerNextBlock() int64     { return readLE64(b[8:]) }
func (b *mailBlock) setHeaderFrom(v int64)      { writeLE64(b[16:], v) }
func (b *mailBlock) headerFrom() int64          { return readLE64(b[16:]) }
func (b *mailBlock) setHeaderTo(v int64)        { writeLE64(b[24:], v) }
func (b *mailBlock) headerTo() int64            { return readLE64(b[24:]) }
func (b *mailBlock) setHeaderMailTime(v int64)  { writeLE64(b[32:], v) }
func (b *mailBlock) headerMailTime() int64      { return readLE64(b[32:]) }

func (b *mailBlock) setHeaderTxt(s string) { copy(b[40:40+59], s) }

func (b *mailBlock) headerTxt() string {
	return cStr(b[40 : 40+59])
}

func (b *mailBlock) setDataBlockType(bt int8) { b[0] = byte(bt) }
func (b *mailBlock) dataBlockType() int8      { return int8(b[0]) }

func (b *mailBlock) setDataTxt(s string) { copy(b[8:8+91], s) }

func (b *mailBlock) dataTxt() string {
	return cStr(b[8 : 8+91])
}

func cStr(buf []byte) string {
	for i, c := range buf {
		if c == 0 {
			return string(buf[:i])
		}
	}
	return string(buf)
}

// =========================================================================
// Mail file I/O
// =========================================================================

func pushFreeList(pos int64) {
	freeList = &PositionListType{Position: pos, Next: freeList}
}

func popFreeList() int64 {
	if freeList == nil {
		return fileEndPos
	}
	pos := freeList.Position
	freeList = freeList.Next
	return pos
}

func findCharInIndex(searchee int64) *MailIndexType {
	if searchee < 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #1 (searchee == %d).", searchee)
		return nil
	}
	for tmp := mailIndex; tmp != nil; tmp = tmp.Next {
		if tmp.Recipient == searchee {
			return tmp
		}
	}
	return nil
}

func writeToFile(buf *mailBlock, filepos int64) {
	if filepos%BlockSize != 0 {
		log.Printf("SYSERR: Mail system -- fatal error #2!!! (invalid file position %d)", filepos)
		noMail = true
		return
	}
	f, err := os.OpenFile(MailFilePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("SYSERR: Unable to open mail file '%s'.", MailFilePath)
		noMail = true
		return
	}
	defer f.Close()
	if _, err := f.Seek(filepos, 0); err != nil {
		log.Printf("SYSERR: seek failed on mail file: %v", err)
		noMail = true
		return
	}
	if _, err := f.Write(buf[:]); err != nil {
		log.Printf("SYSERR: write failed on mail file: %v", err)
		noMail = true
		return
	}
	stat, err := f.Stat()
	if err != nil {
		log.Printf("SYSERR: stat failed on mail file: %v", err)
		noMail = true
		return
	}
	fileEndPos = stat.Size()
}

func readFromFile(m *mailBlock, filepos int64) {
	if filepos%BlockSize != 0 {
		log.Printf("SYSERR: Mail system -- fatal error #3!!! (invalid filepos read %d)", filepos)
		noMail = true
		return
	}
	f, err := os.OpenFile(MailFilePath, os.O_RDWR, 0644)
	if err != nil {
		log.Printf("SYSERR: Unable to open mail file '%s'.", MailFilePath)
		noMail = true
		return
	}
	defer f.Close()
	if _, err := f.Seek(filepos, 0); err != nil {
		log.Printf("SYSERR: seek failed on mail file: %v", err)
		noMail = true
		return
	}
	if _, err := f.Read(m[:]); err != nil {
		log.Printf("SYSERR: read failed on mail file: %v", err)
		noMail = true
		return
	}
}

func indexMail(idToIndex int64, pos int64) {
	if idToIndex < 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #4. (id_to_index == %d)", idToIndex)
		return
	}
	idx := findCharInIndex(idToIndex)
	if idx == nil {
		idx = &MailIndexType{Recipient: idToIndex}
		idx.Next = mailIndex
		mailIndex = idx
	}
	idx.ListStart = &PositionListType{Position: pos, Next: idx.ListStart}
}

// =========================================================================
// Mail system functions (ported from mail.c)
// =========================================================================

// ScanFile scans the mail file at boot time and indexes all entries.
func ScanFile() bool {
	f, err := os.Open(MailFilePath)
	if err != nil {
		log.Println("   Mail file non-existent... creating new file.")
		_ = os.WriteFile(MailFilePath, []byte{}, 0644)
		return true
	}
	defer f.Close()

	totalMessages := 0
	blockNum := 0
	var block mailBlock

	for {
		n, err := f.Read(block[:])
		if err != nil || n != BlockSize {
			break
		}
		bt := block.blockType()
		if bt == HeaderBlock {
			indexMail(block.headerTo(), int64(blockNum)*BlockSize)
			totalMessages++
		} else if bt == DeletedBlock {
			pushFreeList(int64(blockNum) * BlockSize)
		}
		blockNum++
	}

	fileEndPos = int64(blockNum) * BlockSize

	log.Printf("   %d bytes read.", fileEndPos)
	if fileEndPos%BlockSize != 0 {
		log.Println("SYSERR: Error booting mail system -- Mail file corrupt!")
		log.Println("SYSERR: Mail disabled!")
		return false
	}
	log.Printf("   Mail file read -- %d messages.", totalMessages)
	return true
}

// HasMail checks whether a recipient has any mail waiting.
func HasMail(recipient int64) bool {
	return findCharInIndex(recipient) != nil
}

// StoreMail stores a mail message in the file.
func StoreMail(to int64, from int64, message string) {
	if from < 0 || to < 0 || len(message) == 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #5. (from == %d, to == %d)", from, to)
		return
	}

	msgBytes := []byte(message)
	totalLength := len(msgBytes)

	var header mailBlock
	header.setBlockType(HeaderBlock)
	header.setHeaderNextBlock(-1)
	header.setHeaderFrom(from)
	header.setHeaderTo(to)
	header.setHeaderMailTime(time.Now().Unix())
	header.setHeaderTxt(string(msgBytes))
	if totalLength > 0 && totalLength < 59 {
		b := header[40+totalLength : 40+59]
		for i := range b {
			b[i] = 0
		}
	}

	targetAddress := popFreeList()
	indexMail(to, targetAddress)
	writeToFile(&header, targetAddress)

	if totalLength <= HeaderBlockDataSize {
		return
	}

	bytesWritten := HeaderBlockDataSize
	msgBytes = msgBytes[HeaderBlockDataSize:]

	lastAddress := targetAddress
	targetAddress = popFreeList()
	header.setHeaderNextBlock(targetAddress)
	writeToFile(&header, lastAddress)

	var data mailBlock
	data.setDataBlockType(LastBlock)
	data.setDataTxt(string(msgBytes))
	writeToFile(&data, targetAddress)
	bytesWritten += len(msgBytes)
	if len(msgBytes) > DataBlockDataSize {
		msgBytes = msgBytes[DataBlockDataSize:]
	} else {
		msgBytes = nil
	}

	for bytesWritten < totalLength {
		lastAddress = targetAddress
		targetAddress = popFreeList()

		data.setDataBlockType(int8(targetAddress))
		writeToFile(&data, lastAddress)

		data.setDataBlockType(LastBlock)
		data.setDataTxt(string(msgBytes))
		writeToFile(&data, targetAddress)

		bytesWritten += len(msgBytes)
		if len(msgBytes) > DataBlockDataSize {
			msgBytes = msgBytes[DataBlockDataSize:]
		} else {
			msgBytes = nil
		}
	}
}

// ReadDelete retrieves one mail message for the player, then deletes it.
func ReadDelete(recipient int64) string {
	if recipient < 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #6. (recipient: %d)", recipient)
		return ""
	}

	mailPtr := findCharInIndex(recipient)
	if mailPtr == nil {
		log.Println("SYSERR: Mail system -- post office spec_proc error?  Error #7.")
		return ""
	}
	if mailPtr.ListStart == nil {
		log.Println("SYSERR: Mail system -- non-fatal error #8. (invalid position pointer)")
		return ""
	}

	var mailAddress int64

	if mailPtr.ListStart.Next == nil {
		mailAddress = mailPtr.ListStart.Position
		mailPtr.ListStart = nil
		if mailIndex == mailPtr {
			mailIndex = mailPtr.Next
		} else {
			prev := mailIndex
			for prev != nil && prev.Next != mailPtr {
				prev = prev.Next
			}
			if prev != nil {
				prev.Next = mailPtr.Next
			}
		}
	} else {
		posPtr := mailPtr.ListStart
		for posPtr.Next.Next != nil {
			posPtr = posPtr.Next
		}
		mailAddress = posPtr.Next.Position
		posPtr.Next = nil
	}

	var block mailBlock
	readFromFile(&block, mailAddress)

	if block.blockType() != HeaderBlock {
		log.Printf("SYSERR: Oh dear. (Header block %d != %d)", block.blockType(), HeaderBlock)
		noMail = true
		log.Println("SYSERR: Mail system disabled!  -- Error #9. (Invalid header block.)")
		return ""
	}

	mailTime := block.headerMailTime()
	fromID := block.headerFrom()
	toID := block.headerTo()

	fromName := resolveName(fromID)
	toName := resolveName(toID)
	if fromName == "" {
		fromName = "Unknown"
	}
	if toName == "" {
		toName = "Unknown"
	}

	dateStr := time.Unix(mailTime, 0).Format(time.ANSIC)
	msgBody := fmt.Sprintf(" * * * * Dark Pawns Mail System * * * *\r\n"+
		"Date: %s\r\n"+
		"  To: %s\r\n"+
		"From: %s\r\n\r\n", dateStr, toName, fromName)
	msgBody += block.headerTxt()

	block.setBlockType(DeletedBlock)
	writeToFile(&block, mailAddress)
	pushFreeList(mailAddress)

	nextBlock := block.headerNextBlock()
	for nextBlock != int64(LastBlock) {
		var data mailBlock
		readFromFile(&data, nextBlock)
		msgBody += data.dataTxt()

		addr := nextBlock
		nextBlock = int64(data.dataBlockType())

		data.setDataBlockType(DeletedBlock)
		writeToFile(&data, addr)
		pushFreeList(addr)
	}

	return msgBody
}

// resolveName resolves a player ID to a display name (default: empty).
// Set the variable to a real function at startup.
var resolveName = func(id int64) string { return "" }

// resolveID resolves a player name to an ID (default: -1).
// Set the variable to a real function at startup.
var resolveID = func(name string) int64 { return -1 }

// SetMailResolveFuncs sets callbacks for ID ↔ name resolution.
func SetMailResolveFuncs(nameFn func(int64) string, idFn func(string) int64) {
	resolveName = nameFn
	resolveID = idFn
}

// =========================================================================
// Postmaster mob special procedure
// =========================================================================

func postmaster(w *World, ch *Player, me *MobInstance, cmd string, arg string) bool {
	if ch.IsNPC() {
		return false
	}
	if cmd != "mail" && cmd != "check" && cmd != "receive" {
		return false
	}
	if noMail {
		sendToChar(ch, "Sorry, the mail system is having technical difficulties.\r\n")
		return false
	}
	switch cmd {
	case "mail":
		postmasterSendMail(w, ch, me, arg)
		return true
	case "check":
		postmasterCheckMail(w, ch, me, arg)
		return true
	case "receive":
		postmasterReceiveMail(w, ch, me, arg)
		return true
	}
	return false
}

func postmasterSendMail(w *World, ch *Player, me *MobInstance, arg string) {
	if ch.Level < MinMailLevel {
		sendToChar(ch, fmt.Sprintf("$n tells you, 'Sorry, you have to be level %d to send mail!'", MinMailLevel))
		return
	}

	recipientName := strings.TrimSpace(arg)
	if recipientName == "" {
		sendToChar(ch, "$n tells you, 'You need to specify an addressee!'")
		return
	}

	if ch.GetGold() < StampPrice {
		sendToChar(ch, fmt.Sprintf("$n tells you, 'A stamp costs %d coins.'\r\n"+
			"$n tells you, '...which I see you can't afford.'", StampPrice))
		return
	}

	recipientID := resolveID(recipientName)
	if recipientID < 0 {
		sendToChar(ch, "$n tells you, 'No one by that name is registered here!'")
		return
	}

	w.roomMessage(ch.RoomVNum, "$n starts to write some mail.")
	sendToChar(ch, fmt.Sprintf("$n tells you, 'I'll take %d coins for the stamp.'\r\n"+
		"$n tells you, 'Write your message, use @ on a new line when done.'", StampPrice))
	ch.SetGold(ch.GetGold() - StampPrice)

	mailMu.Lock()
	MailPendingTo[ch.ID] = recipientID
	MailBuffer[ch.ID] = ""
	mailMu.Unlock()
	ch.SetPlrFlag(4, true)
}

func postmasterCheckMail(w *World, ch *Player, me *MobInstance, arg string) {
	if HasMail(int64(ch.ID)) {
		sendToChar(ch, "$n tells you, 'You have mail waiting.'")
	} else {
		sendToChar(ch, "$n tells you, 'Sorry, you don't have any mail waiting.'")
	}
}

func postmasterReceiveMail(w *World, ch *Player, me *MobInstance, arg string) {
	pid := int64(ch.ID)
	if !HasMail(pid) {
		sendToChar(ch, "$n tells you, 'Sorry, you don't have any mail waiting.'")
		return
	}

	for HasMail(pid) {
		proto := &parser.Obj{
			VNum:      -1,
			Keywords:  "mail paper letter",
			ShortDesc: "a piece of mail",
			LongDesc:  "Someone has left a piece of mail here.",
			TypeFlag:  ITEM_NOTE,
			Weight:    1,
			Cost:      30,
		}
		// ITEM_WEAR_TAKE = bit 15, ITEM_WEAR_HOLD = bit 2
		proto.WearFlags = [3]int{1 << 15, 1 << 2, 0}

		obj := NewObjectInstance(proto, 0)
		msg := ReadDelete(pid)
		if msg == "" {
			msg = "Mail system error - please report.  Error #11.\r\n"
		}
		proto.ActionDesc = msg

		_ = ch.Inventory.AddItem(obj)
		sendToChar(ch, "$n gives you a piece of mail.")
		w.roomMessage(ch.RoomVNum, "$N gives $n a piece of mail.")
	}
}

// HandleMailInput processes a line of mail text input from a player.
// Called externally when a player is in PLR_WRITING state.
// Returns true if the mail message is complete (line == "@").
func HandleMailInput(ch *Player, line string) bool {
	if line == "@" {
		mailMu.Lock()
		mailText := MailBuffer[ch.ID]
		recipientID := MailPendingTo[ch.ID]
		delete(MailPendingTo, ch.ID)
		delete(MailBuffer, ch.ID)
		mailMu.Unlock()

		ch.SetPlrFlag(4, false)

		if mailText != "" && recipientID > 0 {
			StoreMail(recipientID, int64(ch.ID), mailText)
			ch.SendMessage("Mail sent.\r\n")
		}
		return true
	}

	mailMu.Lock()
	buf := MailBuffer[ch.ID]
	if buf != "" {
		buf += "\r\n"
	}
	buf += line
	if len(buf) > MaxMailSize {
		buf = buf[:MaxMailSize]
	}
	MailBuffer[ch.ID] = buf
	mailMu.Unlock()

	return false
}

// InitMail is called at boot to initialize the mail system.
func InitMail() {
	mailMu.Lock()
	defer mailMu.Unlock()
	log.Println("   Scanning mail file...")
	ScanFile()
}

func init() {
	RegisterSpec("postmaster", postmaster)
}
