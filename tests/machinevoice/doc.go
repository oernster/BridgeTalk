// Package machinevoice measures making a complete script with the real model, the real store and
// the real making service (NFR-P-203, NFR-C-502). Its test takes minutes, so it carries the
// benchmarks build tag: it runs with ./test.ps1 -Benchmarks and on every build, never in the
// everyday gate (Oliver, 2026-09-14). This file holds nothing else; it keeps the package visible to
// go list without the tag.
package machinevoice
