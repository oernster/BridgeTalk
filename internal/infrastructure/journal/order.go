// Which of two journal names is the newer, by the time each name states.
//
// The game has named its journals in two forms: a dashed one stating the year in full and a
// dashless one stating it in two digits. A name comparison orders each form within itself but not
// the two against each other ("Journal.21" sorts after "Journal.20"), so the time each states is
// read out of it instead. Reading the name keeps a poll from stat-ing every file in the folder.
package journal

import (
	"strconv"
	"strings"
	"time"
)

// The parts of a journal name around its stamp and its part number.
const (
	journalPrefix = "Journal."
	journalSuffix = ".log"
	partMark      = "."
)

// stampLayouts are the forms a journal name states its time in: dashed, then dashless.
var stampLayouts = []string{"2006-01-02T150405", "060102150405"}

// stamped is what a journal name states: when its session began, which part of that session it
// is and whether it stated a time at all.
type stamped struct {
	dated bool
	at    time.Time
	part  int
	name  string
}

// stampOf reads what a journal's base name states. A name stating no time in either form is
// undated, which is older than any dated one; a part that is no number reads as 0.
func stampOf(base string) stamped {
	stem := strings.TrimSuffix(strings.TrimPrefix(base, journalPrefix), journalSuffix)
	stamp, part := stem, ""
	if cut := strings.LastIndex(stem, partMark); cut >= 0 {
		stamp, part = stem[:cut], stem[cut+len(partMark):]
	}
	number, _ := strconv.Atoi(part)
	for _, layout := range stampLayouts {
		if at, err := time.Parse(layout, stamp); err == nil {
			return stamped{dated: true, at: at, part: number, name: base}
		}
	}
	return stamped{part: number, name: base}
}

// newerThan reports whether s names a later journal than other: a dated name before an undated
// one, then the later time, then the later part, then the later name.
func (s stamped) newerThan(other stamped) bool {
	if s.dated != other.dated {
		return s.dated
	}
	if !s.at.Equal(other.at) {
		return s.at.After(other.at)
	}
	if s.part != other.part {
		return s.part > other.part
	}
	return s.name > other.name
}
