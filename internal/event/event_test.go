package event

import "testing"

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

func TestLocationAndTitle(t *testing.T) {
	ev := Event{Type: "TypeError", File: "A.php", Line: 3}
	if ev.Location() != "A.php:3" {
		t.Fatalf("location = %q", ev.Location())
	}
	if ev.Title() != "TypeError" {
		t.Fatalf("title = %q", ev.Title())
	}
}
