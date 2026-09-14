package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"os"

	"github.com/oernster/bridge-talk/internal/infrastructure/madelines"
	"github.com/oernster/bridge-talk/internal/refusal"
)

// The shape of the plain PCM WAV a line is written as for the finder.
const (
	// wavChannels is how many channels a line holds: the model makes mono.
	wavChannels = 1
	// wavPCM is the format code of plain integer samples.
	wavPCM = 1
	// bitsPerByte turns bits per sample into bytes.
	bitsPerByte = 8
)

// wavHeader is a plain PCM WAV header as it is written, field by field, little-endian.
type wavHeader struct {
	Riff       [4]byte
	Size       uint32
	Wave       [4]byte
	Format     [4]byte
	FormatSize uint32
	Code       uint16
	Channels   uint16
	Rate       uint32
	ByteRate   uint32
	BlockAlign uint16
	Bits       uint16
	Data       [4]byte
	DataSize   uint32
}

// formatChunkBytes is the length of the format chunk after its size: the fields from Code to Bits.
var formatChunkBytes = binary.Size(wavHeader{}.Code) + binary.Size(wavHeader{}.Channels) +
	binary.Size(wavHeader{}.Rate) + binary.Size(wavHeader{}.ByteRate) +
	binary.Size(wavHeader{}.BlockAlign) + binary.Size(wavHeader{}.Bits)

// riffPrefixBytes is what the RIFF size leaves out: the RIFF mark and the size itself.
var riffPrefixBytes = binary.Size(wavHeader{}.Riff) + binary.Size(wavHeader{}.Size)

// writeWAV writes a line's samples at path as a mono WAV at the model's rate, each sample turned into
// 16 bits as a made line's are (FR-526).
func writeWAV(path string, samples []float32) error {
	sixteen := madelines.SixteenBits(samples)
	narrowed := make([]int16, len(sixteen))
	for at, sample := range sixteen {
		narrowed[at] = int16(sample)
	}
	blockAlign := wavChannels * madelines.BitsPerSample / bitsPerByte
	dataBytes := len(narrowed) * blockAlign
	header := wavHeader{
		Riff: [4]byte{'R', 'I', 'F', 'F'}, Size: uint32(binary.Size(wavHeader{}) - riffPrefixBytes + dataBytes),
		Wave: [4]byte{'W', 'A', 'V', 'E'}, Format: [4]byte{'f', 'm', 't', ' '}, FormatSize: uint32(formatChunkBytes),
		Code: wavPCM, Channels: wavChannels, Rate: madelines.SampleRate, ByteRate: uint32(madelines.SampleRate * blockAlign),
		BlockAlign: uint16(blockAlign), Bits: madelines.BitsPerSample,
		Data: [4]byte{'d', 'a', 't', 'a'}, DataSize: uint32(dataBytes),
	}
	var written bytes.Buffer
	// Writing fixed-size values to memory never fails, so there is no error to handle.
	_ = binary.Write(&written, binary.LittleEndian, header)
	_ = binary.Write(&written, binary.LittleEndian, narrowed)
	if err := os.WriteFile(path, written.Bytes(), fileMode); err != nil {
		return fmt.Errorf("writing %s: %w", path, refusal.Reason(err))
	}
	return nil
}
