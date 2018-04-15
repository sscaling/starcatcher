package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/gocarina/gocsv"
)

// func TestRead(t *testing.T) {
// 	f, err := os.Open("example.json")
// 	if err != nil {
// 		t.Fatalf("Can't open file %v", err)
// 	}

// 	res, err := ReadJSON(f)
// 	if err != nil {
// 		t.Fatalf("Couldn't read file %v", err)
// 	}

// 	if res.Stargazers != 1610 {
// 		t.Fail()
// 	}
// }
// Convert the internal date as CSV string

func TestReadCsv(t *testing.T) {
	f, err := os.OpenFile("stats.csv", os.O_RDONLY, os.ModePerm)
	if err != nil {
		t.Fatalf("Couldn't read stats.csv")
	}
	defer f.Close()

	data := []*CsvRow{}
	if err := gocsv.UnmarshalFile(f, &data); err != nil {
		t.Fatalf("couldn't unmarshal CSV")
	}

	for i, d := range data {
		fmt.Printf("%d: date?%v\n", i, d)
	}
}
