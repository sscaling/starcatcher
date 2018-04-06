package main

import (
	"os"
	"testing"
)

func TestRead(t *testing.T) {
	f, err := os.Open("example.json")
	if err != nil {
		t.Fatalf("Can't open file %v", err)
	}

	res, err := ReadJSON(f)
	if err != nil {
		t.Fatalf("Couldn't read file %v", err)
	}

	if res.Stargazers != 1610 {
		t.Fail()
	}
}
