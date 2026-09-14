// Package madelines keeps the lines made for machine voices: one folder a voice, each line a mono
// 16-bit FLAC file named by its key (FR-526), under the product's own data folder (FR-523).
package madelines

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"

	"github.com/oernster/bridge-talk/internal/domain/machinevoice"
	"github.com/oernster/bridge-talk/internal/infrastructure/appdata"
	"github.com/oernster/bridge-talk/internal/refusal"
)

const (
	// Folder names the made lines' folder inside the product's data folder.
	Folder = "Made lines"
	// SampleRate is the rate the model makes samples at: 24 kHz, measured from its output on
	// 2026-09-13.
	SampleRate = 24000
	// BitsPerSample is how finely each sample is stored (FR-526).
	BitsPerSample = 16
)

const (
	// extension names a made line's file; partSuffix follows it while the line is being written.
	extension  = ".flac"
	partSuffix = ".part"
	// blockSize is how many samples a frame holds, as measured at 56.7 percent of WAV's size.
	blockSize = 4096
	// predictorOrder is the order of the fixed predictor each frame is coded with.
	predictorOrder = 2
	// shortestPredicted is the fewest samples a frame is predicted over: one past the predictor's
	// warm-up. A shorter frame is written verbatim.
	shortestPredicted = predictorOrder + 1
	// maxRiceParameter is the largest Rice parameter the 4-bit field holds; 15 is FLAC's escape.
	maxRiceParameter = 14
	folderPerm       = 0o755
	filePerm         = 0o644
)

// Store keeps made lines under one folder.
type Store struct {
	dir string
}

// New keeps made lines under the folder given.
func New(dir string) Store { return Store{dir: dir} }

// Dir answers with the made lines' folder inside the product's local data folder (FR-523).
func Dir() (string, error) {
	base, err := appdata.Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, Folder), nil
}

// Keys lists the keys of a voice's made lines, sorted. A folder that cannot be read holds none;
// neither a part left by an interrupted write nor anything else in the folder is a made line.
func (s Store) Keys(voice machinevoice.Voice) []string {
	entries, err := os.ReadDir(s.voiceDir(voice))
	if err != nil {
		return nil
	}
	var keys []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != extension {
			continue
		}
		keys = append(keys, strings.TrimSuffix(entry.Name(), extension))
	}
	return keys
}

// Path returns where a made line is played from.
func (s Store) Path(voice machinevoice.Voice, key string) string {
	return filepath.Join(s.voiceDir(voice), key+extension)
}

// Write stores a made line whole or not at all (FR-517). It is written beside its place then
// renamed into it, so an interrupted write leaves a part, which Keys never lists; a refused write
// removes its part.
func (s Store) Write(voice machinevoice.Voice, key string, samples []float32) error {
	folder := s.voiceDir(voice)
	if err := os.MkdirAll(folder, folderPerm); err != nil {
		return fmt.Errorf("making %s: %w", folder, refusal.Reason(err))
	}
	line := s.Path(voice, key)
	part := line + partSuffix
	if err := os.WriteFile(part, encode(sixteenBits(samples)), filePerm); err != nil {
		os.Remove(part)
		return fmt.Errorf("writing %s: %w", part, refusal.Reason(err))
	}
	if err := os.Rename(part, line); err != nil {
		os.Remove(part)
		return fmt.Errorf("writing %s: %w", line, refusal.Reason(err))
	}
	return nil
}

// DeleteAllBut deletes the made lines of every voice except the one given (FR-527), going on past
// any that cannot be deleted and naming each (FR-530).
func (s Store) DeleteAllBut(voice machinevoice.Voice) error { return s.deleteExcept(voice.ID()) }

// DeleteAll deletes every made line (FR-527), naming any that cannot be deleted (FR-530).
func (s Store) DeleteAll() error { return s.deleteExcept(noFolder) }

// noFolder is a name no folder has, so deleting except it keeps nothing.
const noFolder = ""

// deleteExcept deletes everything under the store's folder but the folder named keep.
func (s Store) deleteExcept(keep string) error {
	entries, err := os.ReadDir(s.dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading %s: %w", s.dir, refusal.Reason(err))
	}
	var failed []error
	for _, entry := range entries {
		if entry.Name() == keep {
			continue
		}
		path := filepath.Join(s.dir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			failed = append(failed, fmt.Errorf("deleting %s: %w", path, refusal.Reason(err)))
		}
	}
	return errors.Join(failed...)
}

func (s Store) voiceDir(voice machinevoice.Voice) string { return filepath.Join(s.dir, voice.ID()) }

// sixteenBits turns the model's samples into FR-526's: clamped to between -1 and 1, scaled to 16
// bits and rounded.
func sixteenBits(samples []float32) []int32 {
	out := make([]int32, len(samples))
	for index, sample := range samples {
		out[index] = int32(math.Round(math.Max(-1, math.Min(1, float64(sample))) * math.MaxInt16))
	}
	return out
}

// encode writes samples as a mono FLAC stream in memory. The stream states its sample count up
// front, so nothing has to be written back into it once the frames are done.
//
// The library's errors are not handled because none can arise here. Read in its source on
// 2026-09-14, every error its encoder returns is a failed write (which memory never gives) or a
// frame shaped other than the mono 16-bit 24 kHz verbatim and fixed-predictor frames frameOf
// builds, each of which the tests round-trip.
func encode(samples []int32) []byte {
	var out bytes.Buffer
	info := &meta.StreamInfo{
		BlockSizeMin: blockSize, BlockSizeMax: blockSize, SampleRate: SampleRate,
		NChannels: 1, BitsPerSample: BitsPerSample, NSamples: uint64(len(samples)),
	}
	encoder, _ := flac.NewEncoder(&out, info)
	for number, start := 0, 0; start < len(samples); number, start = number+1, start+blockSize {
		block := samples[start:min(start+blockSize, len(samples))]
		_ = encoder.WriteFrame(frameOf(uint64(number), block))
	}
	_ = encoder.Close()
	return out.Bytes()
}

// frameOf codes one block: with the fixed predictor where it is long enough, verbatim otherwise.
func frameOf(number uint64, block []int32) *frame.Frame {
	sub := &frame.Subframe{Samples: block, NSamples: len(block)}
	sub.SubHeader = frame.SubHeader{Pred: frame.PredVerbatim}
	if len(block) >= shortestPredicted {
		sub.SubHeader = frame.SubHeader{
			Pred: frame.PredFixed, Order: predictorOrder,
			ResidualCodingMethod: frame.ResidualCodingMethodRice1,
			RiceSubframe: &frame.RiceSubframe{
				Partitions: []frame.RicePartition{{Param: riceParameter(block)}},
			},
		}
	}
	return &frame.Frame{
		Header: frame.Header{
			HasFixedBlockSize: true, BlockSize: uint16(len(block)), SampleRate: SampleRate,
			Channels: frame.ChannelsMono, BitsPerSample: BitsPerSample, Num: number,
		},
		Subframes: []*frame.Subframe{sub},
	}
}

// riceParameter chooses the Rice parameter from the mean size of the fixed predictor's errors. Any
// parameter decodes to the same samples; this one keeps the file small.
func riceParameter(block []int32) uint {
	var sum float64
	for index := predictorOrder; index < len(block); index++ {
		sum += math.Abs(float64(block[index] - 2*block[index-1] + block[index-2]))
	}
	mean := sum / float64(len(block)-predictorOrder)
	if mean < 1 {
		return 0
	}
	return uint(min(math.Floor(math.Log2(mean)), maxRiceParameter))
}
