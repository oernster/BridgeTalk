//go:build !windows

package voicefiles

// RuntimeFile is ONNX Runtime off Windows, as models.toml names Linux's.
const RuntimeFile = "libonnxruntime.so"
