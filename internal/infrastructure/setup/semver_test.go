package setup

import "testing"

func TestCompareOrdersVersions(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		a, b string
		want Relation
	}{
		{"a later patch is newer", "1.2.4", "1.2.3", Newer},
		{"an earlier patch is older", "1.2.3", "1.2.4", Older},
		{"a later minor beats a later patch", "1.3.0", "1.2.9", Newer},
		{"a later major beats everything below it", "2.0.0", "1.9.9", Newer},
		{"identical versions are the same", "1.2.3", "1.2.3", Same},
		{"surrounding space is ignored", " 1.2.3 ", "1.2.3", Same},
		{"a pre-release suffix does not count", "1.2.3-rc1", "1.2.3", Same},
		{"a missing patch field reads as zero", "1.2", "1.2.0", Same},
		{"a missing minor field reads as zero", "1", "1.0.0", Same},
		{"an unreadable field reads as zero", "1.x.3", "1.0.3", Same},
		{"an empty version is the earliest", "", "0.0.1", Older},
		{"extra fields beyond the patch are ignored", "1.2.3.9", "1.2.3", Same},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			t.Parallel()
			if got := Compare(item.a, item.b); got != item.want {
				t.Errorf("Compare(%q, %q) is %d, want %d", item.a, item.b, got, item.want)
			}
		})
	}
}

// TestCompareIsAntisymmetric checks the relation reverses when the arguments do,
// which is the property the setup program leans on when it decides between offering
// an update and offering a downgrade.
func TestCompareIsAntisymmetric(t *testing.T) {
	t.Parallel()
	pairs := [][2]string{{"1.2.3", "1.2.4"}, {"2.0.0", "1.9.9"}, {"0.1.0", "0.1.0"}}
	for _, pair := range pairs {
		forward, backward := Compare(pair[0], pair[1]), Compare(pair[1], pair[0])
		if forward != -backward {
			t.Errorf("Compare(%q, %q) is %d but the reverse is %d", pair[0], pair[1], forward, backward)
		}
	}
}
