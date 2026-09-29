package main

import "testing"

func TestParseSums(t *testing.T) {
	txt := "ABC123  cinema.exe\nffff *setup.exe\n"
	if got := parseSums(txt, "cinema.exe"); got != "abc123" {
		t.Fatalf("got %q", got)
	}
	if got := parseSums(txt, "setup.exe"); got != "ffff" {
		t.Fatalf("got %q", got)
	}
	if got := parseSums(txt, "x"); got != "" {
		t.Fatalf("got %q", got)
	}
}

func TestFind(t *testing.T) {
	r := release{Assets: []asset{{Name: "a"}, {Name: "cinema.exe", URL: "u"}}}
	if a, ok := r.find("cinema.exe"); !ok || a.URL != "u" {
		t.Fatal("asset não encontrado")
	}
	if _, ok := r.find("nada"); ok {
		t.Fatal("achou o que não existe")
	}
}
