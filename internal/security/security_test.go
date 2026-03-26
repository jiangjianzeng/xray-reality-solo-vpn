package security

import "testing"

func TestNormalizeClientName(t *testing.T) {
	got := NormalizeClientName("  MacBook \n\t Pro  ")
	if got != "MacBook Pro" {
		t.Fatalf("unexpected normalized name: %q", got)
	}
}

func TestValidateClientName(t *testing.T) {
	if err := ValidateClientName("A"); err == nil {
		t.Fatal("expected short name to be invalid")
	}
	if err := ValidateClientName("Office Mac"); err != nil {
		t.Fatalf("expected regular name to be valid, got: %v", err)
	}
	if err := ValidateClientName("bad\x00name"); err == nil {
		t.Fatal("expected control chars to be invalid")
	}
}

func TestYAMLQuote(t *testing.T) {
	if YAMLQuote("Alice's Mac") != "'Alice''s Mac'" {
		t.Fatalf("unexpected yaml quote output")
	}
}
