package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	chart "github.com/wcharczuk/go-chart"
)

// GithubStats represents the github repo information
type GithubStats struct {
	Stargazers  int `json:"stargazers_count"`
	Forks       int `json:"forks_count"`
	Watchers    int `json:"watchers"`
	Subscribers int `json:"subscribers_count"`
}

func (s GithubStats) csv() []string {
	return []string{
		strconv.Itoa(s.Stargazers),
		strconv.Itoa(s.Forks),
		strconv.Itoa(s.Watchers),
		strconv.Itoa(s.Subscribers),
	}
}

// DockerhubStats represents the dockerhub repo information
type DockerhubStats struct {
	Pulls int `json:"pull_count"`
	Stars int `json:"star_count"`
}

func (s DockerhubStats) csv() []string {
	return []string{
		strconv.Itoa(s.Pulls),
		strconv.Itoa(s.Stars),
	}
}

const (
	dateTime int = iota
	githubStars
	githubForks
	githubWatchers
	githubSubscribers
	dockerPulls
	dockerStars
)

// CsvRow represents one row in the CSV output
type CsvRow struct {
	DateTime string
	GithubStats
	DockerhubStats
}

func newRow(g GithubStats, d DockerhubStats) *CsvRow {
	return &CsvRow{
		time.Now().Format(time.RFC3339),
		g,
		d,
	}
}

// StatsCsv represents a CSV output file
type StatsCsv struct {
	FieldNames []string
	Fields     []CsvRow
}

func (csv *StatsCsv) appendRow(row CsvRow) {
	csv.Fields = append(csv.Fields, row)
}

// func newCsv(fieldNames []string) *StatsCsv {
// 	return &StatsCsv{
// 		FieldNames: []string{
// 			"DateTime",
// 			"GithubStars",
// 			"GithubForks",
// 			"GithubWatchers",
// 			"GithubSubscribers",
// 			"DockerPulls",
// 			"DockerStars",
// 		},
// 		Fields: []string{},
// 	}
// }

// const dateTime = 0
// const githubStar = 1
// const dockerPulls = 5

func buildCsv(gh GithubStats, dh DockerhubStats) []string {
	csv := []string{time.Now().Format(time.RFC3339)}
	csv = append(csv, gh.csv()...)
	return append(csv, dh.csv()...)
}

func appendToCsv(filename string, stats []string) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s file for append", filename)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err = w.Write(stats); err != nil {
		return fmt.Errorf("unable to append stats to %s", filename)
	}
	w.Flush()

	return w.Error()
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

// FIXME: should just be mapping of source to repo/users. Cannot use fixed user/repo combo
func getStats(client *http.Client, src *StatSource) ([]string, error) {
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

	return buildCsv(github, dockerhub), nil
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

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		s := scanner.Text()
		parts := strings.Split(s, ",")
		t, err := time.Parse(time.RFC3339, parts[dateTime])
		if err != nil {
			fmt.Printf("Unable to process date in %s\n", s)
		}

		v, err := strconv.ParseFloat(parts[githubStars], 64)
		if err != nil {
			fmt.Printf("Unable to process value in %s\n", s)
		}

		data.XValues = append(data.XValues, t)
		data.YValues = append(data.YValues, v)

		if len(parts) > dockerPulls {
			v, err := strconv.ParseFloat(parts[dockerPulls], 64)
			if err != nil {
				fmt.Printf("Unable to process secondary value in %s\n", s)
			}

			data.SecondaryY = append(data.SecondaryY, v)
		} else {
			data.SecondaryY = append(data.SecondaryY, 0)
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
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

	err = json.Unmarshal(j, result)
	if err != nil {
		return fmt.Errorf("can't unmarshall JSON %v", err)
	}

	return nil
}
