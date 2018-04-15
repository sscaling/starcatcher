package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gocarina/gocsv"
	chart "github.com/wcharczuk/go-chart"
)

// GithubStats represents the github repo information
type GithubStats struct {
	Stargazers  float64 `json:"stargazers_count" csv:"gh:stars"`
	Forks       int     `json:"forks_count" csv:"gh:forks"`
	Watchers    int     `json:"watchers" csv:"gh:watchers"`
	Subscribers int     `json:"subscribers_count" csv:"gh:subscribers"`
}

// DockerhubStats represents the dockerhub repo information
type DockerhubStats struct {
	Pulls float64 `json:"pull_count" csv:"dh:pulls"`
	Stars int     `json:"star_count" csv:"dh:stars"`
}

// CsvRow represents one row in the CSV output
type CsvRow struct {
	DateTime time.Time `csv:"time"`
	GithubStats
	DockerhubStats
}

func appendToCsv(filename string, stats *CsvRow) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s file for append", filename)
	}
	defer file.Close()

	return gocsv.MarshalWithoutHeaders([]*CsvRow{stats}, file)
}

type statRequest struct {
	URI     string
	Headers map[string]string
}

func newStatRequest(uri string) *statRequest {
	return &statRequest{
		URI:     uri,
		Headers: make(map[string]string),
	}
}

func main() {
	fmt.Println("Star-catcher")

	if len(os.Args) < 3 {
		log.Fatalf("Usage: main <csv filename> <png filename>")
	}

	csvFilename := os.Args[1]
	pngFilename := os.Args[2]

	log.Printf("Generating %s from %s\n", pngFilename, csvFilename)

	// 1. read stats from github
	src := StatSource{
		GithubURI:    "https://api.github.com/repos/wurstmeister/kafka-docker",
		DockerhubURI: "https://hub.docker.com/v2/repositories/wurstmeister/kafka/",
	}
	stats, err := getStats(&http.Client{}, &src)
	if err != nil {
		log.Fatalf("Unable to read stats: %v\n", err)
	}

	// 2. append stats to CSV
	if err = appendToCsv(csvFilename, stats); err != nil {
		log.Fatalf("Unable to append stats to CSV %v\n", err)
	}

	// 3. Read in CSV file and generate a timeseries
	data, err := CsvToTimeSeries(csvFilename)
	if err != nil {
		log.Fatalf("Unable to load data from CSV and generate time series: %v\n", err)
	}

	// 4. Render a graph
	if err = RenderGraph(pngFilename, data); err != nil {
		log.Fatalf("Unable to render graph: %v", err)
	}

	fmt.Println("success")
}

// StatSource defines config for various sources for one project
// Currently this supports github and dockerhub
type StatSource struct {
	GithubURI    string
	DockerhubURI string
}

func getStats(client *http.Client, src *StatSource) (*CsvRow, error) {
	var github GithubStats
	req := newStatRequest(src.GithubURI)
	req.Headers = map[string]string{"Accept": "application/vnd.github.v3+json"}
	if err := readStats(client, req, &github); err != nil {
		return nil, err
	}

	var dockerhub DockerhubStats
	if err := readStats(client, newStatRequest(src.DockerhubURI), &dockerhub); err != nil {
		return nil, err
	}

	return &CsvRow{time.Now(), github, dockerhub}, nil
}

func readStats(client *http.Client, sr *statRequest, stats interface{}) error {
	req, err := http.NewRequest("GET", sr.URI, nil)
	for k, v := range sr.Headers {
		req.Header.Add(k, v)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("can't make request : %v", err)
	}
	defer resp.Body.Close()

	// b, _ := ioutil.ReadAll(resp.Body)
	// fmt.Println(string(b))

	if err := ReadJSON(resp.Body, stats); err != nil {
		return fmt.Errorf("can't read response %v", err)
	}

	fmt.Printf("stats:%#v\n", stats)

	return nil
}

type graphData struct {
	chart.TimeSeries
	SecondaryY []float64
}

// CsvToTimeSeries reads a stats CSV and produces a go-chart chart.TimeSeries object
func CsvToTimeSeries(filename string) (*graphData, error) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	chart := chart.TimeSeries{
		XValues: []time.Time{},
		YValues: []float64{},
	}

	data := graphData{
		TimeSeries: chart,
		SecondaryY: []float64{},
	}

	csv := []*CsvRow{}
	if err := gocsv.UnmarshalFile(file, &csv); err != nil {
		return nil, err
	}

	for _, d := range csv {
		data.XValues = append(data.XValues, d.DateTime)
		data.YValues = append(data.YValues, d.Stargazers)
		data.SecondaryY = append(data.SecondaryY, d.Pulls)
	}

	return &data, nil
}

// RenderGraph takes a chart.TimeSeries and renders it to the specified filename
func RenderGraph(filename string, data *graphData) error {
	// Try and lock the range to start -/+50 of min/max values
	var max float64
	min := math.MaxFloat64
	for _, y := range data.YValues {
		max = math.Max(max, y+math.Mod(y, 50))
		min = math.Min(min, y-math.Mod(y, 50))
	}

	ticks := []chart.Tick{}
	for i := int(min) - 100; i < int(max)+100; i = i + 100 {
		ticks = append(ticks, chart.Tick{Value: float64(i), Label: strconv.Itoa(i)})
	}

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
				Min: min,
				Max: max,
			},
		},
		YAxisSecondary: chart.YAxis{
			Style: chart.Style{
				Show: true, //enables / displays the secondary y-axis
			},
			// FIXME: calculate this
			Range: &chart.ContinuousRange{
				Min: data.SecondaryY[0] - 1000000,
				Max: data.SecondaryY[len(data.SecondaryY)-1] + 1000000,
			},
		},
		Series: []chart.Series{
			data.TimeSeries,
			chart.TimeSeries{
				YAxis:   chart.YAxisSecondary,
				XValues: data.XValues,
				YValues: data.SecondaryY,
			},
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return fmt.Errorf("unable to render PNG chart from stats: %v", err)
	}

	return ioutil.WriteFile(filename, buffer.Bytes(), 0644)
}

// ReadJSON reads input from io.Reader and produces a GithubStats
func ReadJSON(r io.Reader, result interface{}) error {
	j, err := ioutil.ReadAll(r)
	if err != nil {
		return fmt.Errorf("can't read source bytes: %v", err)
	}

	return json.Unmarshal(j, result)
}
