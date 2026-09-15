package game

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestFixedMailTextUsesFirstNULTerminator(t *testing.T) {
	fullField := bytes.Repeat([]byte{'F'}, MailHeaderDataSize)
	tests := []struct {
		name  string
		field []byte
		want  string
	}{
		{name: "empty_text", field: []byte{0, 0, 0}, want: ""},
		{name: "short_text_with_zero_padding", field: []byte{'s', 'h', 'o', 'r', 't', 0, 0, 0}, want: "short"},
		{name: "embedded_terminator", field: []byte{'l', 'e', 'f', 't', 0, 'r', 'i', 'g', 'h', 't'}, want: "left"},
		{name: "full_field_without_terminator", field: fullField, want: string(fullField)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := fixedMailText(tt.field); got != tt.want {
				t.Fatalf("fixedMailText() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestMailReceiveConvertsShortSingleBlockText proves the native Go receive
// path turns one fixed header field into semantic text without changing the
// existing deleted-block write.
func TestMailReceiveConvertsShortSingleBlockText(t *testing.T) {
	setupNativeMailReadFixture(t)

	const body = "short native body"
	header := mailHeader{
		BlockType: MailBlockHeader,
		To:        mailLifecycleRecipientID,
		From:      mailLifecycleSenderID,
		MailTime:  0,
		NextBlock: MailBlockLast,
	}
	copy(header.Text[:], body)
	before := marshalMailHeader(&header)
	writeToFile(before, MailBlockSize, 0)
	indexMail(header.To, 0)

	got := readDelete(header.To)
	if !strings.HasSuffix(got, body) {
		t.Fatalf("received mail = %q, want suffix %q", got, body)
	}
	if strings.IndexByte(got, 0) >= 0 {
		t.Fatalf("received mail contains NUL padding: %q", strings.ReplaceAll(got, "\x00", "<NUL>"))
	}

	afterHeader := header
	afterHeader.BlockType = MailBlockDeleted
	wantAfter := marshalMailHeader(&afterHeader)
	after, err := os.ReadFile(MailFile)
	if err != nil {
		t.Fatalf("read deleted native mail fixture: %v", err)
	}
	if !bytes.Equal(after, wantAfter) {
		t.Fatalf("deleted single-block fixture changed beyond marker")
	}
}

// TestMailReceiveJoinsBoundedFullMultiBlockText proves that the read path
// joins a complete header field and one complete continuation field without
// trimming legitimate bytes or carrying fixed-block padding into the object.
func TestMailReceiveJoinsBoundedFullMultiBlockText(t *testing.T) {
	setupNativeMailReadFixture(t)

	headerText := strings.Repeat("H", MailHeaderDataSize)
	continuationText := strings.Repeat("C", MailDataBlockSize)
	header := mailHeader{
		BlockType: MailBlockHeader,
		To:        mailLifecycleRecipientID,
		From:      mailLifecycleSenderID,
		MailTime:  0,
		NextBlock: MailBlockSize,
	}
	copy(header.Text[:], headerText)
	data := mailData{BlockType: MailBlockLast}
	copy(data.Text[:], continuationText)
	writeToFile(marshalMailHeader(&header), MailBlockSize, 0)
	writeToFile(marshalMailData(&data), MailBlockSize, MailBlockSize)
	indexMail(header.To, 0)

	got := readDelete(header.To)
	wantText := headerText + continuationText
	if !strings.HasSuffix(got, wantText) {
		t.Fatalf("received multi-block suffix bytes = %d, want %d", len(got), len(wantText))
	}
	if strings.IndexByte(got, 0) >= 0 {
		t.Fatalf("received multi-block mail contains NUL padding")
	}

	deletedHeader := header
	deletedHeader.BlockType = MailBlockDeleted
	deletedData := data
	deletedData.BlockType = MailBlockDeleted
	wantAfter := append(marshalMailHeader(&deletedHeader), marshalMailData(&deletedData)...)
	after, err := os.ReadFile(MailFile)
	if err != nil {
		t.Fatalf("read deleted multi-block fixture: %v", err)
	}
	if !bytes.Equal(after, wantAfter) {
		t.Fatalf("deleted multi-block fixture changed beyond markers")
	}
}

// TestMailReceiveStopsContinuationAtFirstNUL proves that readDelete applies
// the fixed-field C string contract to continuation blocks, including when a
// corrupt-looking nonzero tail follows the first terminator.
func TestMailReceiveStopsContinuationAtFirstNUL(t *testing.T) {
	setupNativeMailReadFixture(t)

	headerText := "header text"
	continuationPrefix := "continuation text"
	continuationTail := "nonzero padding"
	header := mailHeader{
		BlockType: MailBlockHeader,
		To:        mailLifecycleRecipientID,
		From:      mailLifecycleSenderID,
		MailTime:  0,
		NextBlock: MailBlockSize,
	}
	copy(header.Text[:], headerText)
	data := mailData{BlockType: MailBlockLast}
	copy(data.Text[:], continuationPrefix)
	data.Text[len(continuationPrefix)] = 0
	copy(data.Text[len(continuationPrefix)+1:], continuationTail)

	before := append(marshalMailHeader(&header), marshalMailData(&data)...)
	writeToFile(marshalMailHeader(&header), MailBlockSize, 0)
	writeToFile(marshalMailData(&data), MailBlockSize, MailBlockSize)
	indexMail(header.To, 0)

	got := readDelete(header.To)
	wantText := headerText + continuationPrefix
	if !strings.HasSuffix(got, wantText) {
		t.Fatalf("received continuation text = %q, want suffix %q", got, wantText)
	}
	if strings.Contains(got, continuationTail) || strings.IndexByte(got, 0) >= 0 {
		t.Fatalf("received continuation leaked bytes after first NUL: %q", got)
	}

	deletedHeader := header
	deletedHeader.BlockType = MailBlockDeleted
	deletedData := data
	deletedData.BlockType = MailBlockDeleted
	wantAfter := append(marshalMailHeader(&deletedHeader), marshalMailData(&deletedData)...)
	after, err := os.ReadFile(MailFile)
	if err != nil {
		t.Fatalf("read deleted continuation fixture: %v", err)
	}
	if !bytes.Equal(after, wantAfter) {
		t.Fatalf("deleted continuation fixture changed beyond block-type markers")
	}
	if bytes.Equal(before, after) {
		t.Fatalf("mail fixture did not record the existing deletion-marker transitions")
	}
}

// TestMailReceiveProducesSemanticRuntimeText proves the native Go mail path
// places the C-semantic text in the delivered object before persistence.
func TestMailReceiveProducesSemanticRuntimeText(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0o700); err != nil {
		t.Fatalf("create mail data directory: %v", err)
	}

	oldIndex, oldFreeList, oldFileEndPos := mailIndex, freeList, fileEndPos
	oldDisabled := mailDisabled
	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	t.Cleanup(func() {
		mailIndex, freeList, fileEndPos = oldIndex, oldFreeList, oldFileEndPos
		mailDisabled = oldDisabled
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
	})

	w, sender, postmasterFn, postmasterMob, _ := newMailLifecycleWorld(t)
	if !InitMailSystem(mailLifecycleNameByID, mailLifecycleIDByName) {
		t.Fatal("mail initialization failed")
	}

	postmasterFn(w, sender, postmasterMob, "mail", "Recipient body")
	if HandleMailInput(sender, mailLifecycleBody) {
		t.Fatal("mail body line unexpectedly completed composition")
	}
	if !HandleMailInput(sender, "@") {
		t.Fatal("@ did not complete mail composition")
	}

	recipient := NewPlayer(mailLifecycleRecipientID, "Recipient", 1001)
	if err := w.AddPlayer(recipient); err != nil {
		t.Fatalf("add recipient: %v", err)
	}
	postmasterFn(w, recipient, postmasterMob, "receive", "")

	items := recipient.Inventory.FindItems("")
	if len(items) != 1 {
		t.Fatalf("received inventory items = %d, want 1", len(items))
	}
	got := items[0].Runtime.MailText
	if got == "" || !strings.HasSuffix(got, mailLifecycleBody) {
		t.Fatalf("runtime mail text does not end with body: bytes=%d text=%q", len(got), got)
	}
	if strings.IndexByte(got, 0) >= 0 {
		t.Fatalf("runtime mail text contains fixed-block NUL padding: bytes=%d text=%q", len(got), got)
	}
	t.Logf("mail_runtime_text_bytes=%d mail_runtime_text_nul_count=0 body=%q", len(got), mailLifecycleBody)
}

func setupNativeMailReadFixture(t *testing.T) {
	t.Helper()
	t.Chdir(t.TempDir())
	if err := os.Mkdir("data", 0o700); err != nil {
		t.Fatalf("create mail data directory: %v", err)
	}

	oldIndex, oldFreeList, oldFileEndPos := mailIndex, freeList, fileEndPos
	oldDisabled := mailDisabled
	oldNameFunc, oldIDFunc := worldNameFunc, worldIDFunc
	mailIndex, freeList, fileEndPos = nil, nil, 0
	mailDisabled = false
	worldNameFunc, worldIDFunc = nil, nil
	t.Cleanup(func() {
		mailIndex, freeList, fileEndPos = oldIndex, oldFreeList, oldFileEndPos
		mailDisabled = oldDisabled
		worldNameFunc, worldIDFunc = oldNameFunc, oldIDFunc
	})
}
