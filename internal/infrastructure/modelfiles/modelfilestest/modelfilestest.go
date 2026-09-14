// Package modelfilestest serves model files from a local server for the tests of the model files
// tool, recording what is asked for, so no test reaches the network.
package modelfilestest

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/modelfiles"
)

// Server answers each path it has been offered with that path's bytes; every other path it answers
// with Not Found.
type Server struct {
	server *httptest.Server
	mu     sync.Mutex
	bodies map[string][]byte
	cut    map[string]bool
	asked  []string
}

// Serve starts a server that closes when the test ends.
func Serve(t *testing.T) *Server {
	t.Helper()
	served := &Server{bodies: map[string][]byte{}, cut: map[string]bool{}}
	served.server = httptest.NewServer(http.HandlerFunc(served.answer))
	t.Cleanup(served.server.Close)
	return served
}

// Close stops the server early, so its addresses reach nothing.
func (s *Server) Close() { s.server.Close() }

// answer records the path asked for, then serves it.
func (s *Server) answer(writer http.ResponseWriter, request *http.Request) {
	s.mu.Lock()
	s.asked = append(s.asked, request.URL.Path)
	body, offered := s.bodies[request.URL.Path]
	cut := s.cut[request.URL.Path]
	s.mu.Unlock()
	if !offered {
		http.NotFound(writer, request)
		return
	}
	if cut {
		// Promising one byte more than is sent ends the answer early at the client.
		writer.Header().Set("Content-Length", strconv.Itoa(len(body)+1))
	}
	_, _ = writer.Write(body)
}

// Cut offers body at path as Offer does; the answer ends before it is whole, as a dropped
// connection would leave it.
func (s *Server) Cut(path string, body []byte) string {
	address := s.Offer(path, body)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cut[path] = true
	return address
}

// Offer serves body at path, replacing anything offered there before. It answers with the address.
func (s *Server) Offer(path string, body []byte) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bodies[path] = body
	return s.server.URL + path
}

// Entry offers body under name and answers with the list entry that matches it.
func (s *Server) Entry(name string, body []byte) modelfiles.File {
	return modelfiles.File{Name: name, Address: s.Offer("/"+name, body), Size: int64(len(body)), SHA256: Digest(body)}
}

// Client is the client that reaches the server.
func (s *Server) Client() *http.Client { return s.server.Client() }

// Asked lists the paths asked for, in the order they were asked.
func (s *Server) Asked() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.asked...)
}

// Digest is the SHA-256 of body written as hexadecimal, as the list gives it.
func Digest(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

// Archive is a zip archive holding each member under its path.
func Archive(t *testing.T, members map[string][]byte) []byte {
	t.Helper()
	var packed bytes.Buffer
	writer := zip.NewWriter(&packed)
	for path, body := range members {
		member, err := writer.Create(path)
		if err != nil {
			t.Fatalf("adding %s to the archive: %v", path, err)
		}
		if _, err := member.Write(body); err != nil {
			t.Fatalf("writing %s into the archive: %v", path, err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("closing the archive: %v", err)
	}
	return packed.Bytes()
}
