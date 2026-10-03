package plugin_test

// B-12: a plugin's misbehaviour is reported for what it was. An answer within the cap on its bytes
// cannot make the reader allocate many times that before refusing it.

import (
	"encoding/binary"
	"runtime"
	"strings"
	"testing"

	"github.com/oernster/bridge-talk/internal/infrastructure/plugin"
)

// faultingVersion is a plugin whose Version faults. Describe and Takes are never reached.
type faultingVersion struct{}

func (faultingVersion) Version() int32                    { panic("the plugin's own fault") }
func (faultingVersion) Describe([]byte) int32             { return 0 }
func (faultingVersion) Takes(int32, []byte, []byte) int32 { return 0 }

// A fault inside Version was swallowed and read as version 0, so the refusal blamed the version
// the plugin was built against. It names the fault.
func TestAPluginWhoseVersionFaultsIsRefusedForTheFault(t *testing.T) {
	t.Parallel()
	set := plugin.Load(folder(t, "one.dll"), opening(map[string]plugin.Library{"one.dll": faultingVersion{}}, nil))
	defer set.Close()

	if len(set.Refusals) != 1 {
		t.Fatalf("refusals = %+v, want one", set.Refusals)
	}
	why := set.Refusals[0].Why
	if strings.Contains(why, "interface version 0") || !strings.Contains(why, "the plugin's own fault") {
		t.Errorf("refusal said %q, want the fault named rather than a version", why)
	}
}

// answerBytes is the size of the answer the takes test builds: the cap on a plugin's answer.
const answerBytes = 4 << 20

// int32Bytes is the width of one number in the layouts.
const int32Bytes = 4

// A takes answer of one take claiming as many parts as it has bytes left, a quarter of what each
// part needs at the least, once made the reader set aside room for every one of them before the
// first was read: 160 MB for a 4 MiB answer. It is refused with no more allocated than the answer
// itself.
func TestATakesAnswerClaimingMorePartsThanItCanHoldIsRefusedBeforeAllocating(t *testing.T) {
	data := make([]byte, answerBytes)
	binary.LittleEndian.PutUint32(data, 1)
	binary.LittleEndian.PutUint32(data[int32Bytes:], uint32((answerBytes-2*int32Bytes)/int32Bytes))

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	_, err := plugin.DecodeTakes(data)
	runtime.ReadMemStats(&after)

	if err == nil {
		t.Fatal("an answer claiming more parts than it holds was read")
	}
	if spent := after.TotalAlloc - before.TotalAlloc; spent > answerBytes {
		t.Errorf("refusing it allocated %d bytes, want no more than the %d of the answer", spent, answerBytes)
	}
}
