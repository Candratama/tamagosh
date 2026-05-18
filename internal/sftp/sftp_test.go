package sftp

import "testing"

func TestParent(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"/", "/"},
		{"/home", "/"},
		{"/home/fauziah", "/home"},
		{"/home/fauziah/uploads", "/home/fauziah"},
		{"", "/"},
	}
	for _, c := range cases {
		got := Parent(c.in)
		if got != c.want {
			t.Fatalf("Parent(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestJoin(t *testing.T) {
	if got := Join("/home", "fauziah"); got != "/home/fauziah" {
		t.Fatalf("got %q", got)
	}
	if got := Join("/", "uploads"); got != "/uploads" {
		t.Fatalf("got %q", got)
	}
	if got := Join("/home/", "/fauziah"); got != "/home/fauziah" {
		t.Fatalf("got %q", got)
	}
}
