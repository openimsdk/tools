package field

import "testing"

func TestPath(t *testing.T) {
	testCases := []struct {
		op       func(*Path) *Path
		expected string
	}{
		{
			func(p *Path) *Path { return p },
			"root",
		},
		{
			func(p *Path) *Path { return p.Child("first") },
			"root.first",
		},
		{
			func(p *Path) *Path { return p.Child("second") },
			"root.first.second",
		},
		{
			func(p *Path) *Path { return p.Index(0) },
			"root.first.second[0]",
		},
		{
			func(p *Path) *Path { return p.Child("third") },
			"root.first.second[0].third",
		},
		{
			func(p *Path) *Path { return p.Index(93) },
			"root.first.second[0].third[93]",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root.first.second[0].third",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root.first.second[0]",
		},
		{
			func(p *Path) *Path { return p.Key("key") },
			"root.first.second[0][key]",
		},
	}

	root := NewPath("root")
	p := root
	for i, tc := range testCases {
		p = tc.op(p)
		if p.String() != tc.expected {
			t.Errorf("[%d] Expected %q, got %q", i, tc.expected, p.String())
		}
		if p.Root() != root {
			t.Errorf("[%d] Wrong root: %#v", i, p.Root())
		}
	}
}

func TestPathMultiArg(t *testing.T) {
	testCases := []struct {
		op       func(*Path) *Path
		expected string
	}{
		{
			func(p *Path) *Path { return p },
			"root.first",
		},
		{
			func(p *Path) *Path { return p.Child("second", "third") },
			"root.first.second.third",
		},
		{
			func(p *Path) *Path { return p.Index(0) },
			"root.first.second.third[0]",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root.first.second.third",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root.first.second",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root.first",
		},
		{
			func(p *Path) *Path { return p.parent },
			"root",
		},
	}

	root := NewPath("root", "first")
	p := root
	for i, tc := range testCases {
		p = tc.op(p)
		if p.String() != tc.expected {
			t.Errorf("[%d] Expected %q, got %q", i, tc.expected, p.String())
		}
		if p.Root() != root.Root() {
			t.Errorf("[%d] Wrong root: %#v", i, p.Root())
		}
	}
}
