package plugin

import (
	"strings"
	"testing"
)

// A plugin is somebody else's code reached through a raw call, so a panic on the way into it is
// a thing that can happen. It must end that call and nothing else: without the guard on the
// plugin thread the whole application goes, window and all, over one voice that misbehaved.
func TestAPanicOnThePluginThreadEndsThatCallAlone(t *testing.T) {
	on := newRunner()
	defer on.close()

	on.do(func() { panic("a plugin went wrong") })

	answered := false
	on.do(func() { answered = true })
	if !answered {
		t.Error("the plugin thread was not there for the next call")
	}
}

// sized is a plugin that names one size for every answer, whatever it is asked.
type sized struct {
	size  int32
	asked int
}

func (p *sized) Version() int32 { return ABIVersion }

func (p *sized) Describe(buffer []byte) int32 {
	p.asked++
	return p.size
}

func (p *sized) Takes(int32, []byte, []byte) int32 { return 0 }

// The size a plugin names is acted on before a byte of its answer can be read, since the buffer
// is made to fit first. A size field is 32 bits wide, so a plugin answering garbage can ask for
// two gigabytes. An allocation that large is not an error a program recovers from; it is the
// application ending with nothing said. So the cap is held exactly: the largest honest answer is
// taken and the first byte over it is refused by the number it asked for.
func TestAnAnswerOverTheCapIsRefusedByItsSize(t *testing.T) {
	for _, each := range []struct {
		name    string
		size    int32
		refused bool
		calls   int
	}{
		{"the largest answer that may be set aside", maxAnswer, false, 2},
		{"one byte more than that", maxAnswer + 1, true, 1},
	} {
		t.Run(each.name, func(t *testing.T) {
			library := &sized{size: each.size}
			on := newRunner()
			defer on.close()
			subject := &Plugin{library: library, on: on}

			answer, err := subject.ask(func(buffer []byte) int32 { return library.Describe(buffer) })

			if each.refused {
				if err == nil {
					t.Fatalf("a plugin asking for %d bytes was obliged with %d", each.size, len(answer))
				}
				if !strings.Contains(err.Error(), "4194305") {
					t.Errorf("refused with %q, want the size it asked for named", err)
				}
			} else if err != nil || int32(len(answer)) != each.size {
				t.Fatalf("answered %d bytes, %v; want the whole of an honest answer", len(answer), err)
			}
			// A refusal happens before the second call, which is the call that would have been
			// handed the buffer that could not be made.
			if library.asked != each.calls {
				t.Errorf("the plugin was called %d times, want %d", library.asked, each.calls)
			}
		})
	}
}
