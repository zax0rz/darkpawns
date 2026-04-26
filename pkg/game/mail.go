// mail.go — Ported from src/mail.c
//
// MUD mail system: store, read, and delete mail messages for players.
// Postmaster NPC spec_proc handled elsewhere.

package game

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// ---------------------------------------------------------------------------
// Constants from mail.h
// ---------------------------------------------------------------------------

const (
	MailBlockSize      = 512
	MailFile           = "data/mail"
	MailMinLevel       = 5
	MailStampPrice     = 50
	MailMaxSize        = 4096

	MailBlockHeader   = 1
	MailBlockDeleted  = 2
	MailBlockLast     = -2

	MailHeaderDataSize = MailBlockSize - 8 - 8 - 4 - 4 // to, from, time, next, block_type
	MailDataBlockSize  = MailBlockSize - 4              // block_type + text
)

// ---------------------------------------------------------------------------
// Mail data structures
// ---------------------------------------------------------------------------

type MailIndex struct {
	Recipient int
	ListStart *MailPosList
	Next      *MailIndex
}

type MailPosList struct {
	Position int
	Next     *MailPosList
}

// headerBlockType and dataBlockType map to the C header_block_type/data_block_type unions
type mailHeader struct {
	BlockType int
	To        int
	From      int
	MailTime  int64
	NextBlock int
	Text      [MailHeaderDataSize]byte
}

type mailData struct {
	BlockType int
	Text      [MailDataBlockSize]byte
}

// ---------------------------------------------------------------------------
// Global mail state
// ---------------------------------------------------------------------------

var (
	mailIndex     *MailIndex
	freeList      *MailPosList
	fileEndPos    int64
	noMail        bool
	worldNameFunc func(id int) string
	worldIDFunc   func(name string) int
)

// InitMailSystem initializes mail from disk.
func InitMailSystem(nameFunc func(id int) string, idFunc func(name string) int) {
	worldNameFunc = nameFunc
	worldIDFunc = idFunc
	scanFile()
}

func GetNameByID(id int) string {
	if worldNameFunc != nil {
		return worldNameFunc(id)
	}
	return fmt.Sprintf("Player(%d)", id)
}

func GetIDByName(name string) int {
	if worldIDFunc != nil {
		return worldIDFunc(name)
	}
	return -1
}

// ---------------------------------------------------------------------------
// Mail file operations
// ---------------------------------------------------------------------------

func pushFreeList(pos int) {
	freeList = &MailPosList{Position: pos, Next: freeList}
}

func popFreeList() int {
	if freeList == nil {
		return int(fileEndPos)
	}
	pos := freeList.Position
	freeList = freeList.Next
	return pos
}

func findCharInIndex(recipient int) *MailIndex {
	if recipient < 0 {
		log.Printf("SYSERR: Mail system -- non fatal error #1 (searchee == %d)", recipient)
		return nil
	}
	for tmp := mailIndex; tmp != nil; tmp = tmp.Next {
		if tmp.Recipient == recipient {
			return tmp
		}
	}
	return nil
}

func writeToFile(buf []byte, size int, filepos int) {
	if filepos%MailBlockSize != 0 {
		log.Printf("SYSERR: Mail system -- fatal error #2!!! (invalid file position %d)", filepos)
		noMail = true
		return
	}
	f, err := os.OpenFile(MailFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("SYSERR: Unable to open mail file '%s'.", MailFile)
		noMail = true
		return
	}
	defer f.Close()
	if _, err := f.Seek(int64(filepos), 0); err != nil {
		log.Printf("SYSERR: Seek error in mail file: %v", err)
		noMail = true
		return
	}
	if _, err := f.Write(buf[:size]); err != nil {
		log.Printf("SYSERR: Write error in mail file: %v", err)
		noMail = true
		return
	}
	// Find end of file
	stat, err := f.Stat()
	if err == nil {
		fileEndPos = stat.Size()
	}
}

func readFromFile(buf []byte, size int, filepos int) {
	if filepos%MailBlockSize != 0 {
		log.Printf("SYSERR: Mail system -- fatal error #3!!! (invalid filepos read %d)", filepos)
		noMail = true
		return
	}
	f, err := os.OpenFile(MailFile, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		log.Printf("SYSERR: Unable to open mail file '%s'.", MailFile)
		noMail = true
		return
	}
	defer f.Close()
	if _, err := f.Seek(int64(filepos), 0); err != nil {
		log.Printf("SYSERR: Seek error in mail file read: %v", err)
		noMail = true
		return
	}
	if _, err := f.Read(buf[:size]); err != nil {
		log.Printf("SYSERR: Read error in mail file: %v", err)
		noMail = true
		return
	}
}

func indexMail(idToIndex int, pos int) {
	if idToIndex < 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #4. (id_to_index == %d)", idToIndex)
		return
	}
	ni := findCharInIndex(idToIndex)
	if ni == nil {
		ni = &MailIndex{Recipient: idToIndex}
		ni.Next = mailIndex
		mailIndex = ni
	}
	ni.ListStart = &MailPosList{Position: pos, Next: ni.ListStart}
}

func scanFile() bool {
	f, err := os.Open(MailFile)
	if err != nil {
		log.Print("   Mail file non-existant... creating new file.")
		os.WriteFile(MailFile, []byte{}, 0644)
		return true
	}
	defer f.Close()

	var nextBlock mailHeader
	totalMessages := 0
	blockNum := 0

	for {
		n, err := f.Read(marshalMailHeader(&nextBlock))
		if n == 0 || err != nil {
			break
		}
		if nextBlock.BlockType == MailBlockHeader {
			indexMail(nextBlock.To, blockNum*MailBlockSize)
			totalMessages++
		} else if nextBlock.BlockType == MailBlockDeleted {
			pushFreeList(blockNum * MailBlockSize)
		}
		blockNum++
	}

	stat, _ := f.Stat()
	fileEndPos = stat.Size()
	log.Printf("   %d bytes read.", fileEndPos)
	if fileEndPos%int64(MailBlockSize) != 0 {
		log.Print("SYSERR: Error booting mail system -- Mail file corrupt!")
		log.Print("SYSERR: Mail disabled!")
		return false
	}
	log.Printf("   Mail file read -- %d messages.", totalMessages)
	return true
}

// ---------------------------------------------------------------------------
// Core mail operations
// ---------------------------------------------------------------------------

func hasMail(recipient int) bool {
	return findCharInIndex(recipient) != nil
}

func storeMail(to, from int, message string) {
	log.Printf("SYSERR: Mail system -- non-fatal error #5. (from == %d, to == %d)", from, to)
	if from < 0 || to < 0 || message == "" {
		return
	}

	msgBytes := []byte(message)
	totalLength := len(msgBytes)

	// Build header block
	header := mailHeader{
		BlockType: MailBlockHeader,
		NextBlock: MailBlockLast,
		From:      from,
		To:        to,
		MailTime:  time.Now().Unix(),
	}
	copy(header.Text[:], msgBytes)
	headerBytes := marshalMailHeader(&header)

	targetAddress := popFreeList()
	indexMail(to, targetAddress)
	writeToFile(headerBytes, MailBlockSize, targetAddress)

	if len(msgBytes) <= MailHeaderDataSize {
		return
	}

	bytesWritten := MailHeaderDataSize
	msgBytes = msgBytes[MailHeaderDataSize:]

	// Link header to first data block
	lastAddress := targetAddress
	targetAddress = popFreeList()
	header.NextBlock = targetAddress
	headerBytes = marshalMailHeader(&header)
	writeToFile(headerBytes, MailBlockSize, lastAddress)

	// Write first data block
	data := mailData{BlockType: MailBlockLast}
	dataSize := len(msgBytes)
	if dataSize > MailDataBlockSize {
		dataSize = MailDataBlockSize
	}
	copy(data.Text[:], msgBytes[:dataSize])
	dataBytes := marshalMailData(&data)
	writeToFile(dataBytes, MailBlockSize, targetAddress)
	bytesWritten += dataSize
	msgBytes = msgBytes[dataSize:]

	// Write remaining data blocks with chaining
	for bytesWritten < totalLength {
		lastAddress = targetAddress
		targetAddress = popFreeList()

		data.BlockType = targetAddress
		dataBytes = marshalMailData(&data)
		writeToFile(dataBytes, MailBlockSize, lastAddress)

		data.BlockType = MailBlockLast
		dataSize = len(msgBytes)
		if dataSize > MailDataBlockSize {
			dataSize = MailDataBlockSize
		}
		copy(data.Text[:], msgBytes[:dataSize])
		dataBytes = marshalMailData(&data)
		writeToFile(dataBytes, MailBlockSize, targetAddress)

		bytesWritten += dataSize
		msgBytes = msgBytes[dataSize:]
	}
}

func readDelete(recipient int) string {
	if recipient < 0 {
		log.Printf("SYSERR: Mail system -- non-fatal error #6. (recipient: %d)", recipient)
		return ""
	}

	mp := findCharInIndex(recipient)
	if mp == nil {
		log.Print("SYSERR: Mail system -- post office spec_proc error?  Error #7.")
		return ""
	}

	pp := mp.ListStart
	if pp == nil {
		log.Print("SYSERR: Mail system -- non-fatal error #8.")
		return ""
	}

	var mailAddress int
	if pp.Next == nil {
		mailAddress = pp.Position
		if mailIndex == mp {
			mailIndex = mp.Next
		} else {
			prev := mailIndex
			for prev != nil && prev.Next != mp {
				prev = prev.Next
			}
			if prev != nil {
				prev.Next = mp.Next
			}
		}
	} else {
		for pp.Next.Next != nil {
			pp = pp.Next
		}
		mailAddress = pp.Next.Position
		pp.Next = nil
	}

	var header mailHeader
	headerBytes := make([]byte, MailBlockSize)
	readFromFile(headerBytes, MailBlockSize, mailAddress)
	unmarshalMailHeader(&header, headerBytes)

	if header.BlockType != MailBlockHeader {
		log.Printf("SYSERR: Oh dear. (Header block %d != %d)", header.BlockType, MailBlockHeader)
		noMail = true
		log.Print("SYSERR: Mail system disabled!  -- Error #9.")
		return ""
	}

	tm := time.Unix(header.MailTime, 0).Format("Mon Jan 2 15:04:05 2006")
	fromName := GetNameByID(header.From)
	toName := GetNameByID(recipient)

	var sb strings.Builder
	sb.WriteString(" * * * * Dark Pawns Mail System * * * *\r\n")
	sb.WriteString(fmt.Sprintf("Date: %s\r\n", tm))
	sb.WriteString(fmt.Sprintf("  To: %s\r\n", toName))
	sb.WriteString(fmt.Sprintf("From: %s\r\n\r\n", fromName))
	sb.WriteString(string(header.Text[:]))
	message := sb.String()

	followingBlock := header.NextBlock

	// Mark header block deleted
	header.BlockType = MailBlockDeleted
	headerBytes = marshalMailHeader(&header)
	writeToFile(headerBytes, MailBlockSize, mailAddress)
	pushFreeList(mailAddress)

	for followingBlock != MailBlockLast {
		var data mailData
		dataBytes := make([]byte, MailBlockSize)
		readFromFile(dataBytes, MailBlockSize, followingBlock)
		unmarshalMailData(&data, dataBytes)

		message += string(data.Text[:])
		mailAddress = followingBlock
		followingBlock = data.BlockType

		data.BlockType = MailBlockDeleted
		dataBytes = marshalMailData(&data)
		writeToFile(dataBytes, MailBlockSize, mailAddress)
		pushFreeList(mailAddress)
	}

	return message
}

// ---------------------------------------------------------------------------
// Postmaster spec_proc helpers
// ---------------------------------------------------------------------------

func (w *World) PostmasterSendMail(ch *Player, mailman *MobInstance, arg string) {
	if ch.GetLevel() < MailMinLevel {
		ch.SendMessage(fmt.Sprintf("$n tells you, 'Sorry, you have to be level %d to send mail!'\r\n", MailMinLevel))
		return
	}

	arg = strings.TrimSpace(arg)
	spaceIdx := strings.Index(arg, " ")
	if spaceIdx == -1 {
		ch.SendMessage("$n tells you, 'You need to specify an addressee!'\r\n")
		return
	}
	name := arg[:spaceIdx]

	if ch.Gold < MailStampPrice {
		ch.SendMessage(fmt.Sprintf("$n tells you, 'A stamp costs %d coins.'\r\n$n tells you, '...which I see you can't afford.'\r\n", MailStampPrice))
		return
	}

	recipient := GetIDByName(name)
	if recipient < 0 {
		ch.SendMessage("$n tells you, 'No one by that name is registered here!'\r\n")
		return
	}

	ch.Gold -= MailStampPrice
	if ch.Gold < 0 {
		ch.Gold = 0
	}

	// In C this sets PLR_MAILING flag and starts string_write.
	// For now, store a stub message.
	ch.SendMessage(fmt.Sprintf("$n tells you, 'I'll take %d coins for the stamp.'\r\n$n tells you, 'Write your message, use @ on a new line when done.'\r\n", MailStampPrice))

	// STUB: Mail message entry via string writer not yet ported.
	// For now, log the intent and store a placeholder.
	log.Printf("Mail: %s writing to %s (ID %d)", ch.GetName(), name, recipient)
}

func (w *World) PostmasterCheckMail(ch *Player, mailman *MobInstance) {
	if hasMail(ch.ID) {
		ch.SendMessage("$n tells you, 'You have mail waiting.'\r\n")
	} else {
		ch.SendMessage("$n tells you, 'Sorry, you don't have any mail waiting.'\r\n")
	}
}

func (w *World) PostmasterReceiveMail(ch *Player, mailman *MobInstance) {
	if !hasMail(ch.ID) {
		ch.SendMessage("$n tells you, 'Sorry, you don't have any mail waiting.'\r\n")
		return
	}

	for hasMail(ch.ID) {
		mailText := readDelete(ch.ID)
		if mailText == "" {
			mailText = "Mail system error - please report.  Error #11.\r\n"
		}

		obj := w.CreateMailObject(ch, mailText)
		if obj != nil {
			w.GiveObjectToChar(obj, ch)
		}

		ch.SendMessage("$n gives you a piece of mail.\r\n")
	}
}

// CreateMailObject creates a note object containing mail text.
func (w *World) CreateMailObject(ch *Player, mailText string) *ObjectInstance {
	// Create a note object with mail text as action_description.
	// Uses explicit field setting since CreateObject from prototype may not exist.
	obj := &ObjectInstance{
		VNum:     -1, // no prototype
		CanPickUp: true,
	}
	obj.Runtime.MailText = mailText
	return obj
}

// GiveObjectToChar adds an object to the player's inventory.
func (w *World) GiveObjectToChar(obj *ObjectInstance, ch *Player) {
	if ch.Inventory != nil {
		ch.Inventory.AddItem(obj)
	}
}

// ---------------------------------------------------------------------------
// Serialization helpers (packed binary)
// ---------------------------------------------------------------------------

func marshalMailHeader(h *mailHeader) []byte {
	buf := make([]byte, MailBlockSize)
	// Simple binary layout matching C struct packing
	int32Bytes(buf, 0, int32(h.BlockType))
	int32Bytes(buf, 4, int32(h.NextBlock))
	int64Bytes(buf, 8, h.MailTime)
	int32Bytes(buf, 16, int32(h.From))
	int32Bytes(buf, 20, int32(h.To))
	copy(buf[24:], h.Text[:])
	return buf
}

func unmarshalMailHeader(h *mailHeader, buf []byte) {
	h.BlockType = int(readInt32(buf, 0))
	h.NextBlock = int(readInt32(buf, 4))
	h.MailTime = readInt64(buf, 8)
	h.From = int(readInt32(buf, 16))
	h.To = int(readInt32(buf, 20))
	copy(h.Text[:], buf[24:24+MailHeaderDataSize])
}

func marshalMailData(d *mailData) []byte {
	buf := make([]byte, MailBlockSize)
	int32Bytes(buf, 0, int32(d.BlockType))
	copy(buf[4:], d.Text[:])
	return buf
}

func unmarshalMailData(d *mailData, buf []byte) {
	d.BlockType = int(readInt32(buf, 0))
	copy(d.Text[:], buf[4:4+MailDataBlockSize])
}

func int32Bytes(buf []byte, off int, v int32) {
	buf[off] = byte(v)
	buf[off+1] = byte(v >> 8)
	buf[off+2] = byte(v >> 16)
	buf[off+3] = byte(v >> 24)
}

func int64Bytes(buf []byte, off int, v int64) {
	int32Bytes(buf, off, int32(v))
	int32Bytes(buf, off+4, int32(v>>32))
}

func readInt32(buf []byte, off int) int32 {
	return int32(buf[off]) | int32(buf[off+1])<<8 | int32(buf[off+2])<<16 | int32(buf[off+3])<<24
}

func readInt64(buf []byte, off int) int64 {
	return int64(readInt32(buf, off)) | int64(readInt32(buf, off+4))<<32
}
