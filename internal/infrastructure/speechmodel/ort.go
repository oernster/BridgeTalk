//go:build windows || linux

package speechmodel

// ONNX Runtime's C API called with cgo disabled, through nativelib. The library is loaded by its full
// path. Each function is read out of the OrtApi table by its position; every call answers a status
// that is nil on success.
//
// Every address is converted to uintptr in the argument list of nativelib.Call itself. Only there does
// Go keep the variable where ONNX Runtime was told it is: converted any earlier, a goroutine stack that
// moved before the call left ONNX Runtime reading and writing the old copy, measured on 2026-09-14 as
// panics, empty lines and "NULL input" refusals (TestAddressesAreConvertedOnlyWhereTheCallIsMade in
// tests/structural).

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"unsafe"

	"github.com/oernster/bridge-talk/internal/infrastructure/nativelib"
	"github.com/oernster/bridge-talk/internal/infrastructure/voicefiles"
)

// apiVersion is the OrtApi table version read: the one ONNX Runtime 1.23 answers, measured working
// with release 1.23.2 on 2026-09-14.
const apiVersion = 23

// The positions in the OrtApi table at apiVersion of the functions called, counted in
// onnxruntime_c_api.h. A position never moves within a version.
const (
	fnGetErrorMessage                = 2
	fnCreateEnv                      = 3
	fnCreateSession                  = 7
	fnRun                            = 9
	fnCreateSessionOptions           = 10
	fnCreateTensorWithDataAsOrtValue = 49
	fnGetTensorMutableData           = 51
	fnGetTensorShapeElementCount     = 64
	fnGetTensorTypeAndShape          = 65
	fnCreateCpuMemoryInfo            = 69
	fnReleaseEnv                     = 92
	fnReleaseStatus                  = 93
	fnReleaseMemoryInfo              = 94
	fnReleaseSession                 = 95
	fnReleaseValue                   = 96
	fnReleaseTensorTypeAndShapeInfo  = 99
	fnReleaseSessionOptions          = 100
)

// apiBaseFunction is the one function ONNX Runtime exports by name; everything else is reached
// through the table it answers.
const apiBaseFunction = "OrtGetApiBase"

// Values of the C API's enumerations.
const (
	// loggingLevelError keeps ONNX Runtime's log to errors, which the application has no console for.
	loggingLevelError = 3
	// deviceAllocator and defaultMemory ask for plain processor memory for the tensors made below.
	deviceAllocator = 0
	defaultMemory   = 0
	// elementFloat32 and elementInt64 are the tensor element types the model reads and writes.
	elementFloat32 = 1
	elementInt64   = 7
)

// The model's own names for what it reads and writes, read from the model on 2026-09-13: the line's
// numbers as int64 [1, n], the style row as float32 [1, 256], the speed as float32 [1]; out comes
// the waveform, float32 samples at 24 kHz.
var (
	inputNames = []string{"input_ids", "style", "speed"}
	outputName = "waveform"
)

// speakingRate is the model's own speed: each line at the pace the voice was trained at.
const speakingRate float32 = 1

// logID names this caller in ONNX Runtime's log.
const logID = "speechmodel"

// pointerWidth is how many bytes one entry of the OrtApi table takes.
const pointerWidth = int(unsafe.Sizeof(uintptr(0)))

// float32Bytes and int64Bytes are how many bytes one element of each tensor takes.
const (
	float32Bytes = 4
	int64Bytes   = 8
)

// ortSession is the model loaded through ONNX Runtime. Every handle is ONNX Runtime's own memory,
// held as a number Go never reads through.
type ortSession struct {
	api     unsafe.Pointer
	env     uintptr
	memory  uintptr
	session uintptr
}

// openSession loads ONNX Runtime from dir, then the model beside it.
func openSession(dir string) (session, error) {
	return openFrom(filepath.Join(dir, voicefiles.RuntimeFile), filepath.Join(dir, voicefiles.ModelFile))
}

// openFrom loads ONNX Runtime from runtimePath, then the model at modelPath. A file that cannot be
// loaded is named once in the reason (FR-237).
func openFrom(runtimePath, modelPath string) (session, error) {
	library, err := nativelib.Open(runtimePath)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", runtimePath, err)
	}
	base, err := library.Function(apiBaseFunction)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", runtimePath, err)
	}
	getAPI := *(*uintptr)(nativelib.Pointer(nativelib.Call(base)))
	table := nativelib.Call(getAPI, apiVersion)
	if table == 0 {
		return nil, fmt.Errorf("loading %s: it does not answer ONNX Runtime API version %d", runtimePath, apiVersion)
	}
	loaded := &ortSession{api: nativelib.Pointer(table)}
	if err := loaded.start(modelPath); err != nil {
		loaded.release()
		return nil, err
	}
	return loaded, nil
}

// start makes ONNX Runtime's environment and memory description, then loads the model.
func (s *ortSession) start(modelPath string) error {
	name := append([]byte(logID), 0)
	if err := s.checked(nativelib.Call(s.function(fnCreateEnv), loggingLevelError,
		uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&s.env)))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	if err := s.checked(nativelib.Call(s.function(fnCreateCpuMemoryInfo), deviceAllocator, defaultMemory,
		uintptr(unsafe.Pointer(&s.memory)))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	var options uintptr
	if err := s.checked(nativelib.Call(s.function(fnCreateSessionOptions),
		uintptr(unsafe.Pointer(&options)))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	defer nativelib.Call(s.function(fnReleaseSessionOptions), options)
	path, err := modelPathFor(modelPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", modelPath, err)
	}
	if err := s.checked(nativelib.Call(s.function(fnCreateSession), s.env, uintptr(unsafe.Pointer(path)),
		options, uintptr(unsafe.Pointer(&s.session)))); err != nil {
		// ONNX Runtime writes the model's full path into its own message, twice for a missing file,
		// measured on 2026-09-14; the path is given once here and the file's name stands in there.
		said := strings.ReplaceAll(err.Error(), modelPath, voicefiles.ModelFile)
		return fmt.Errorf("loading %s: %s", modelPath, said)
	}
	return nil
}

// run makes one line. The numbers, the style row and the speed go in as tensors over Go's own
// memory, pinned while ONNX Runtime holds their addresses, as are the lists of names and values the
// run reads; the waveform is copied out.
func (s *ortSession) run(tokens []int64, style []float32) ([]float32, error) {
	speed := []float32{speakingRate}
	tokenShape := []int64{1, int64(len(tokens))}
	styleShape := []int64{1, int64(len(style))}
	speedShape := []int64{int64(len(speed))}
	output := append([]byte(outputName), 0)

	var pinner runtime.Pinner
	defer pinner.Unpin()
	pinner.Pin(&tokens[0])
	pinner.Pin(&style[0])
	pinner.Pin(&speed[0])
	pinner.Pin(&tokenShape[0])
	pinner.Pin(&styleShape[0])
	pinner.Pin(&speedShape[0])
	pinner.Pin(&output[0])
	names := make([]*byte, len(inputNames))
	for index, name := range inputNames {
		terminated := append([]byte(name), 0)
		pinner.Pin(&terminated[0])
		names[index] = &terminated[0]
	}
	pinner.Pin(&names[0])
	outputs := []*byte{&output[0]}
	pinner.Pin(&outputs[0])
	values := make([]uintptr, len(inputNames))
	pinner.Pin(&values[0])

	defer func() {
		for _, value := range values {
			if value != 0 {
				nativelib.Call(s.function(fnReleaseValue), value)
			}
		}
	}()
	for index, input := range []struct {
		data  unsafe.Pointer
		bytes int
		shape []int64
		kind  uintptr
	}{
		{unsafe.Pointer(&tokens[0]), len(tokens) * int64Bytes, tokenShape, elementInt64},
		{unsafe.Pointer(&style[0]), len(style) * float32Bytes, styleShape, elementFloat32},
		{unsafe.Pointer(&speed[0]), len(speed) * float32Bytes, speedShape, elementFloat32},
	} {
		if err := s.checked(nativelib.Call(s.function(fnCreateTensorWithDataAsOrtValue), s.memory,
			uintptr(input.data), uintptr(input.bytes), uintptr(unsafe.Pointer(&input.shape[0])),
			uintptr(len(input.shape)), input.kind, uintptr(unsafe.Pointer(&values[index])))); err != nil {
			return nil, fmt.Errorf("making a line: %w", err)
		}
	}

	var result uintptr
	if err := s.checked(nativelib.Call(s.function(fnRun), s.session, 0, uintptr(unsafe.Pointer(&names[0])),
		uintptr(unsafe.Pointer(&values[0])), uintptr(len(values)), uintptr(unsafe.Pointer(&outputs[0])),
		uintptr(len(outputs)), uintptr(unsafe.Pointer(&result)))); err != nil {
		return nil, fmt.Errorf("making a line: %w", err)
	}
	defer nativelib.Call(s.function(fnReleaseValue), result)
	return s.samples(result)
}

// samples copies a float32 tensor's elements out of ONNX Runtime's memory.
func (s *ortSession) samples(tensor uintptr) ([]float32, error) {
	var info uintptr
	if err := s.checked(nativelib.Call(s.function(fnGetTensorTypeAndShape), tensor,
		uintptr(unsafe.Pointer(&info)))); err != nil {
		return nil, fmt.Errorf("reading a made line: %w", err)
	}
	defer nativelib.Call(s.function(fnReleaseTensorTypeAndShapeInfo), info)
	var count uintptr
	if err := s.checked(nativelib.Call(s.function(fnGetTensorShapeElementCount), info,
		uintptr(unsafe.Pointer(&count)))); err != nil {
		return nil, fmt.Errorf("reading a made line: %w", err)
	}
	var data unsafe.Pointer
	if err := s.checked(nativelib.Call(s.function(fnGetTensorMutableData), tensor,
		uintptr(unsafe.Pointer(&data)))); err != nil {
		return nil, fmt.Errorf("reading a made line: %w", err)
	}
	return append([]float32(nil), unsafe.Slice((*float32)(data), count)...), nil
}

// release gives back whatever was made, the model first.
func (s *ortSession) release() {
	for _, held := range []struct {
		handle   uintptr
		function int
	}{{s.session, fnReleaseSession}, {s.memory, fnReleaseMemoryInfo}, {s.env, fnReleaseEnv}} {
		if held.handle != 0 {
			nativelib.Call(s.function(held.function), held.handle)
		}
	}
	s.session, s.memory, s.env = 0, 0, 0
}

// checked answers ONNX Runtime's message where status, a call's answer, says it failed.
func (s *ortSession) checked(status uintptr) error {
	if status == 0 {
		return nil
	}
	text := nativelib.String(nativelib.Call(s.function(fnGetErrorMessage), status))
	nativelib.Call(s.function(fnReleaseStatus), status)
	return errors.New(text)
}

// function reads the address of the function at index out of the OrtApi table.
func (s *ortSession) function(index int) uintptr {
	return *(*uintptr)(unsafe.Add(s.api, index*pointerWidth))
}
