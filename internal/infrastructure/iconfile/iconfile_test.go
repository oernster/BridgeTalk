package iconfile

// Reading the committed .ico: the frame the Linux tray is handed, every side the flatpak installs and
// a file that lies about itself refused (FR-814, FR-810).

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// traySide is the side the Linux tray is handed.
const traySide = 64

// committedIcon is the application's one icon file, read from the repository.
func committedIcon(t *testing.T) []byte {
	t.Helper()
	ico, err := os.ReadFile(filepath.Join("..", "..", "..", "assets", "application-icon.ico"))
	if err != nil {
		t.Fatalf("reading the committed icon: %v", err)
	}
	return ico
}

// The committed icon holds a 64 pixel PNG, the side the Linux tray is handed.
func TestTheCommittedIconHoldsTheTraysPicture(t *testing.T) {
	frame, err := Frame(committedIcon(t), traySide)
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	if !bytes.HasPrefix(frame, pngSignature) {
		t.Fatal("the frame answered is not a PNG")
	}
	width := binary.BigEndian.Uint32(frame[16:20])
	height := binary.BigEndian.Uint32(frame[20:24])
	if width != traySide || height != traySide {
		t.Errorf("the frame is %dx%d, want %d square", width, height, traySide)
	}
}

// A side the icon does not hold is said so; the full side, stored as zero, is found.
func TestAFrameIsFoundBySideWithZeroStandingForTheFullSide(t *testing.T) {
	ico := committedIcon(t)
	if _, err := Frame(ico, 65); !errors.Is(err, ErrNoFrame) {
		t.Errorf("a side the icon lacks answered %v, want ErrNoFrame", err)
	}
	if _, err := Frame(ico, fullSide); err != nil {
		t.Errorf("the full side stored as zero was not found: %v", err)
	}
}

// Nothing the file says about itself is trusted: a short file, an entry past the end, a frame past
// the end and a frame that is not a PNG are each refused rather than read.
func TestAnIconThatLiesAboutItselfIsRefused(t *testing.T) {
	header := func(count uint16) []byte {
		out := make([]byte, headerSize)
		binary.LittleEndian.PutUint16(out[countOffset:], count)
		return out
	}
	entry := func(side byte, length, start uint32) []byte {
		out := make([]byte, entrySize)
		out[0], out[1] = side, side
		binary.LittleEndian.PutUint32(out[lengthOffset:], length)
		binary.LittleEndian.PutUint32(out[startOffset:], start)
		return out
	}
	withEntry := func(length, start uint32, data []byte) []byte {
		out := append(header(1), entry(traySide, length, start)...)
		return append(out, data...)
	}
	dataStart := uint32(headerSize + entrySize)
	cases := map[string][]byte{
		"shorter than a header":   {0, 0},
		"an entry past the end":   header(3),
		"a frame past the end":    withEntry(1<<31, dataStart, pngSignature),
		"a frame that is not PNG": withEntry(8, dataStart, []byte("GIF89a..")),
	}
	for name, ico := range cases {
		if frame, err := Frame(ico, traySide); err == nil {
			t.Errorf("%s: answered %d bytes, want a refusal", name, len(frame))
		}
	}
}

// Sides lists every square frame of the committed icon, which is what the flatpak installs.
func TestSidesListsEveryFrameOfTheCommittedIcon(t *testing.T) {
	sides, err := Sides(committedIcon(t))
	if err != nil {
		t.Fatalf("sides: %v", err)
	}
	if want := []int{16, 24, 32, 48, 64, 128, 256}; !reflect.DeepEqual(sides, want) {
		t.Errorf("sides = %v, want %v", sides, want)
	}
	if _, err := Sides([]byte{0}); err == nil {
		t.Error("a file too short to be an icon was read for its sides")
	}
}
