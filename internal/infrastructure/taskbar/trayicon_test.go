package taskbar

// The picture the Linux tray hands the desktop survives the tray library's conversion (FR-814).

import (
	"bytes"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/iconfile"
)

// systrayARGB is how fyne.io/systray v1.12.2 turns a decoded picture into the bytes it publishes,
// copied from its argbForImage: the low byte of each 16 bit premultiplied channel.
func systrayARGB(picture image.Image) []byte {
	bounds := picture.Bounds()
	out := make([]byte, 0, bounds.Dx()*bounds.Dy()*4)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := picture.At(x, y).RGBA()
			out = append(out, byte(a), byte(r), byte(g), byte(b))
		}
	}
	return out
}

// decodedNRGBA reads a PNG into straight, unpremultiplied pixels.
func decodedNRGBA(t *testing.T, frame []byte) *image.NRGBA {
	t.Helper()
	decoded, err := png.Decode(bytes.NewReader(frame))
	if err != nil {
		t.Fatalf("decoding: %v", err)
	}
	picture := image.NewNRGBA(decoded.Bounds())
	draw.Draw(picture, picture.Bounds(), decoded, decoded.Bounds().Min, draw.Src)
	return picture
}

// The committed icon's tray frame, once made into the tray's picture, reaches the desktop as the
// artwork's own colours: every byte the library publishes is the pixel it came from.
func TestTheTraysPictureReachesTheDesktopAsTheArtworksColours(t *testing.T) {
	ico, err := os.ReadFile(filepath.Join("..", "..", "..", "assets", "application-icon.ico"))
	if err != nil {
		t.Fatalf("reading the committed icon: %v", err)
	}
	frame, err := iconfile.Frame(ico, linuxTraySide)
	if err != nil {
		t.Fatalf("frame: %v", err)
	}
	picture, err := trayPicture(frame)
	if err != nil {
		t.Fatalf("tray picture: %v", err)
	}
	decoded, err := png.Decode(bytes.NewReader(picture))
	if err != nil {
		t.Fatalf("decoding the tray picture: %v", err)
	}
	sent := systrayARGB(decoded)
	artwork := decodedNRGBA(t, frame).Pix
	drawn := 0
	for at := 0; at < len(artwork); at += 4 {
		alpha := sent[at]
		if alpha != 0 && alpha != math.MaxUint8 {
			t.Fatalf("pixel %d reaches the desktop at coverage %d, neither drawn nor clear", at/4, alpha)
		}
		if alpha == 0 {
			continue
		}
		drawn++
		if got, want := sent[at+1:at+4], artwork[at:at+3]; !bytes.Equal(got, want) {
			t.Fatalf("pixel %d reaches the desktop as %v, the artwork has %v", at/4, got, want)
		}
	}
	if drawn == 0 {
		t.Fatal("the tray picture draws nothing")
	}
}

// A pixel at least half covered is drawn fully opaque; one less than half covered is left clear.
func TestTheTraysPictureDrawsWhatIsAtLeastHalfCovered(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	source.Pix = []byte{10, 20, 30, opaqueFrom, 40, 50, 60, opaqueFrom - 1}
	var frame bytes.Buffer
	if err := png.Encode(&frame, source); err != nil {
		t.Fatalf("encoding: %v", err)
	}
	picture, err := trayPicture(frame.Bytes())
	if err != nil {
		t.Fatalf("tray picture: %v", err)
	}
	got := decodedNRGBA(t, picture).Pix
	if got[3] != math.MaxUint8 || !bytes.Equal(got[:3], source.Pix[:3]) {
		t.Errorf("a half covered pixel came out %v, want %v fully opaque", got[:4], source.Pix[:3])
	}
	if got[7] != 0 {
		t.Errorf("a less than half covered pixel came out at coverage %d, want clear", got[7])
	}
}

// Bytes that are not a PNG are refused rather than handed on.
func TestTheTraysPictureRefusesWhatIsNotAPNG(t *testing.T) {
	if _, err := trayPicture([]byte("not a picture")); err == nil {
		t.Error("bytes that are not a PNG were made into a tray picture")
	}
}
