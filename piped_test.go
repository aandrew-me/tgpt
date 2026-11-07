package main

import (
	"bufio"
	"errors"
	"strings"
	"testing"
)

func TestReadPipedKeepsNewlinesBetweenLines(t *testing.T) {
	got, err := readPiped(strings.NewReader("alpha\nbeta\ngamma\n"))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	if got != "alpha\nbeta\ngamma" {
		t.Fatalf("readPiped = %q, want lines separated by newlines", got)
	}
}

func TestReadPipedSingleLineUnchanged(t *testing.T) {
	got, err := readPiped(strings.NewReader("only"))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	if got != "only" {
		t.Fatalf("readPiped = %q, want %q", got, "only")
	}
}

func TestReadPipedEmpty(t *testing.T) {
	got, err := readPiped(strings.NewReader(""))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	if got != "" {
		t.Fatalf("readPiped = %q, want empty", got)
	}
}

func TestReadPipedPreservesBlankLine(t *testing.T) {
	got, err := readPiped(strings.NewReader("a\n\nb\n"))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	if got != "a\n\nb" {
		t.Fatalf("readPiped = %q, want a blank line preserved", got)
	}
}

func TestReadPipedCRLF(t *testing.T) {
	got, err := readPiped(strings.NewReader("a\r\nb\r\n"))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	if got != "a\nb" {
		t.Fatalf("readPiped = %q, want %q", got, "a\nb")
	}
}

func TestReadPipedLongLine(t *testing.T) {
	line := strings.Repeat("A", 80*1024)
	got, err := readPiped(strings.NewReader(line + "\nnext\n"))
	if err != nil {
		t.Fatalf("readPiped: %v", err)
	}
	want := line + "\nnext"
	if got != want {
		t.Fatalf("readPiped length %d, want %d", len(got), len(want))
	}
}

func TestReadPipedLineOver1MiB(t *testing.T) {
	line := strings.Repeat("B", 1024*1024+1)
	_, err := readPiped(strings.NewReader(line + "\n"))
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Fatalf("err = %v, want bufio.ErrTooLong", err)
	}
}
