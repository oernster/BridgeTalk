package cue

// GroupCategory answers the category an audition group belongs to: the one holding the most of the
// group's cues, the earlier in the table's order where two hold as many; none where no cue of the group
// has a category, as for the cue from the application (FR-749).
func (t Table) GroupCategory(group string) string {
	held := make(map[string]int, len(t.categories))
	for _, item := range t.all {
		if item.id.Group() == group && item.category != "" {
			held[item.category]++
		}
	}
	chosen, most := "", 0
	for _, name := range t.categories {
		if held[name] > most {
			chosen, most = name, held[name]
		}
	}
	return chosen
}
