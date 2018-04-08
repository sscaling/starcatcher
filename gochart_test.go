package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/wcharczuk/go-chart"
)

func readCsv(filename string) (*chart.TimeSeries, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	data := chart.TimeSeries{
		XValues: []time.Time{},
		YValues: []float64{},
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		// fmt.Printf(">> %s\n", s)
		parts := strings.Split(s, ",")
		t, err := time.Parse(time.RFC3339, parts[0])
		if err != nil {
			fmt.Printf("Unable to process date in %s\n", s)
		}

		v, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			fmt.Printf("Unable to process value in %s\n", s)
		}

		data.XValues = append(data.XValues, t)
		data.YValues = append(data.YValues, v)
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	return &data, nil
}
func TestGoChart(t *testing.T) {

	// FIXME: handle error
	kafka, _ := readCsv("stats.csv")
	// jmxexporter, _ := readCsv("jmxexporter.csv")

	// Max value in range
	var m float64
	for _, y := range kafka.YValues {
		m = math.Max(m, y)
	}

	fmt.Printf("Max Y value for kafka-docker is %f\n", m)

	ticks := []chart.Tick{}
	for i := 0; i < int(m)+500; i = i + 500 {
		ticks = append(ticks, chart.Tick{Value: float64(i), Label: strconv.Itoa(i)})
	}

	fmt.Println(ticks)

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style: chart.Style{
				Show: true,
			},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{
				Show: true,
			},
			// What ticks to have on Y-axis
			Ticks: ticks,
			// The range of the Y-axis
			Range: &chart.ContinuousRange{
				Min: 0,
				Max: m,
			},
		},
		Series: []chart.Series{
			kafka,
			// jmxexporter,
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		fmt.Println(err)
		t.FailNow()
	}

	f := "/tmp/chart.png"
	err = ioutil.WriteFile(f, buffer.Bytes(), 0644)
	if err != nil {
		t.Fatal(err)
	}

}
