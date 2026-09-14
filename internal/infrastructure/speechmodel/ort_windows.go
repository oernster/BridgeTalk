//go:build windows

package speechmodel

// ONNX Runtime's C API called with cgo disabled. The library is loaded by its full path, so the
// older copy Windows keeps in System32 is never picked up in its place. Each function is read out of
// the OrtApi table by its position; every call answers a status that is nil on success.

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

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
//
// A library once loaded stays loaded until the process ends: nothing here unloads ONNX Runtime,
// since whether it can be unloaded safely while its threads may still run has not been measured.
func openFrom(runtimePath, modelPath string) (session, error) {
	library, err := windows.LoadDLL(runtimePath)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", runtimePath, loadReason(err))
	}
	base, err := library.FindProc("OrtGetApiBase")
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", runtimePath, loadReason(err))
	}
	baseAddress, _, _ := base.Call()
	getAPI := *(*uintptr)(pointerAt(baseAddress))
	table, _, _ := syscall.SyscallN(getAPI, apiVersion)
	if table == 0 {
		return nil, fmt.Errorf("loading %s: it does not answer ONNX Runtime API version %d", runtimePath, apiVersion)
	}
	loaded := &ortSession{api: pointerAt(table)}
	if err := loaded.start(modelPath); err != nil {
		loaded.release()
		return nil, err
	}
	return loaded, nil
}

// loadReason keeps what Windows said about a library it could not load, without the library's
// name, which the caller gives with its full path.
func loadReason(err error) error {
	var failed *windows.DLLError
	if errors.As(err, &failed) {
		return failed.Err
	}
	return err
}

// start makes ONNX Runtime's environment and memory description, then loads the model.
func (s *ortSession) start(modelPath string) error {
	name := append([]byte(logID), 0)
	if err := s.call(fnCreateEnv, loggingLevelError, uintptr(unsafe.Pointer(&name[0])), uintptr(unsafe.Pointer(&s.env))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	if err := s.call(fnCreateCpuMemoryInfo, deviceAllocator, defaultMemory, uintptr(unsafe.Pointer(&s.memory))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	var options uintptr
	if err := s.call(fnCreateSessionOptions, uintptr(unsafe.Pointer(&options))); err != nil {
		return fmt.Errorf("starting ONNX Runtime: %w", err)
	}
	defer syscall.SyscallN(s.function(fnReleaseSessionOptions), options)
	path, err := windows.UTF16PtrFromString(modelPath)
	if err != nil {
		return fmt.Errorf("loading %s: %w", modelPath, err)
	}
	if err := s.call(fnCreateSession, s.env, uintptr(unsafe.Pointer(path)), options, uintptr(unsafe.Pointer(&s.session))); err != nil {
		// ONNX Runtime writes the model's full path into its own message, twice for a missing file,
		// measured on 2026-09-14; the path is given once here and the file's name stands in there.
		said := strings.ReplaceAll(err.Error(), modelPath, voicefiles.ModelFile)
		return fmt.Errorf("loading %s: %s", modelPath, said)
	}
	return nil
}

// run makes one line. The numbers, the style row and the speed go in as tensors over Go's own
// memory, pinned while ONNX Runtime holds their addresses; the waveform is copied out.
func (s *ortSession) run(tokens []int64, style []float32) ([]float32, error) {
	speed := []float32{speakingRate}
	tokenShape := []int64{1, int64(len(tokens))}
	styleShape := []int64{1, int64(len(style))}
	speedShape := []int64{int64(len(speed))}
	names := make([][]byte, len(inputNames))
	namePointers := make([]uintptr, len(inputNames))
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
	for index, name := range inputNames {
		names[index] = append([]byte(name), 0)
		pinner.Pin(&names[index][0])
		namePointers[index] = uintptr(unsafe.Pointer(&names[index][0]))
	}
	outputPointer := uintptr(unsafe.Pointer(&output[0]))

	values := make([]uintptr, 0, len(inputNames))
	defer func() {
		for _, value := range values {
			syscall.SyscallN(s.function(fnReleaseValue), value)
		}
	}()
	for _, input := range []struct {
		data  unsafe.Pointer
		bytes int
		shape []int64
		kind  uintptr
	}{
		{unsafe.Pointer(&tokens[0]), len(tokens) * int64Bytes, tokenShape, elementInt64},
		{unsafe.Pointer(&style[0]), len(style) * float32Bytes, styleShape, elementFloat32},
		{unsafe.Pointer(&speed[0]), len(speed) * float32Bytes, speedShape, elementFloat32},
	} {
		var value uintptr
		if err := s.call(fnCreateTensorWithDataAsOrtValue, s.memory, uintptr(input.data), uintptr(input.bytes),
			uintptr(unsafe.Pointer(&input.shape[0])), uintptr(len(input.shape)), input.kind,
			uintptr(unsafe.Pointer(&value))); err != nil {
			return nil, fmt.Errorf("making a line: %w", err)
		}
		values = append(values, value)
	}

	var result uintptr
	if err := s.call(fnRun, s.session, 0, uintptr(unsafe.Pointer(&namePointers[0])), uintptr(unsafe.Pointer(&values[0])),
		uintptr(len(values)), uintptr(unsafe.Pointer(&outputPointer)), 1, uintptr(unsafe.Pointer(&result))); err != nil {
		return nil, fmt.Errorf("making a line: %w", err)
	}
	defer syscall.SyscallN(s.function(fnReleaseValue), result)
	return s.samples(result)
}

// samples copies a float32 tensor's elements out of ONNX Runtime's memory.
func (s *ortSession) samples(tensor uintptr) ([]float32, error) {
	var info uintptr
	if err := s.call(fnGetTensorTypeAndShape, tensor, uintptr(unsafe.Pointer(&info))); err != nil {
		return nil, fmt.Errorf("reading a made line: %w", err)
	}
	defer syscall.SyscallN(s.function(fnReleaseTensorTypeAndShapeInfo), info)
	var count uintptr
	if err := s.call(fnGetTensorShapeElementCount, info, uintptr(unsafe.Pointer(&count))); err != nil {
		return nil, fmt.Errorf("reading a made line: %w", err)
	}
	var data unsafe.Pointer
	if err := s.call(fnGetTensorMutableData, tensor, uintptr(unsafe.Pointer(&data))); err != nil {
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
			syscall.SyscallN(s.function(held.function), held.handle)
		}
	}
	s.session, s.memory, s.env = 0, 0, 0
}

// call calls the function at index with args, answering ONNX Runtime's message where it fails.
func (s *ortSession) call(index int, args ...uintptr) error {
	status, _, _ := syscall.SyscallN(s.function(index), args...)
	if status == 0 {
		return nil
	}
	message, _, _ := syscall.SyscallN(s.function(fnGetErrorMessage), status)
	text := windows.BytePtrToString((*byte)(pointerAt(message)))
	syscall.SyscallN(s.function(fnReleaseStatus), status)
	return errors.New(text)
}

// function reads the address of the function at index out of the OrtApi table.
func (s *ortSession) function(index int) uintptr {
	return *(*uintptr)(unsafe.Add(s.api, index*pointerWidth))
}

// pointerAt treats an address ONNX Runtime answered with as the pointer it is. The memory is ONNX
// Runtime's own, never Go's, so the collector has nothing there to move or free.
func pointerAt(address uintptr) unsafe.Pointer {
	return *(*unsafe.Pointer)(unsafe.Pointer(&address))
}
