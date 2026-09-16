package event

import (
	"fmt"
	"testing"
)

func TestFingerprintStableAndDistinct(t *testing.T) {
	a := Fingerprint("laravel", "TypeError", "boom", "PaymentController.php", 81)
	b := Fingerprint("laravel", "TypeError", "boom", "PaymentController.php", 81)
	c := Fingerprint("laravel", "TypeError", "boom", "PaymentController.php", 82)
	if a != b {
		t.Fatal("expected stable fingerprint")
	}
	if a == c {
		t.Fatal("expected different line to change fingerprint")
	}
}

func TestFingerprintNormalizesVolatileBits(t *testing.T) {
	a := Fingerprint("app", "Error", `user 123e4567-e89b-12d3-a456-426614174000 failed for "alice" id 9001`, "a.go", 1)
	b := Fingerprint("app", "Error", `user 999e4567-e89b-12d3-a456-426614174000 failed for "bob" id 4242`, "a.go", 1)
	if a != b {
		t.Fatalf("expected volatile values to group: %s vs %s", a, b)
	}
	c := Fingerprint("app", "Error", "something else entirely", "a.go", 1)
	if a == c {
		t.Fatal("expected different wording to stay distinct")
	}
}

func TestPushSampleCapsAndDedups(t *testing.T) {
	var s []string
	s = PushSample(s, "one")
	s = PushSample(s, "one")
	s = PushSample(s, "two")
	if len(s) != 2 || s[0] != "one" || s[1] != "two" {
		t.Fatalf("%v", s)
	}
	for i := 0; i < 8; i++ {
		s = PushSample(s, fmt.Sprintf("m%d", i))
	}
	if len(s) != 5 {
		t.Fatalf("len=%d", len(s))
	}
}

func TestLocationAndTitle(t *testing.T) {
	ev := Event{Type: "TypeError", File: "A.php", Line: 3}
	if ev.Location() != "A.php:3" {
		t.Fatalf("location = %q", ev.Location())
	}
	if ev.Title() != "TypeError" {
		t.Fatalf("title = %q", ev.Title())
	}
}
