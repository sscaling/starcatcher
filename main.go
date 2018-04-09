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

// RepoResponse represents the github repo information
type RepoResponse struct {
	Stargazers int `json:"stargazers_count"`
}

func (r *RepoResponse) csv() []string {
	return []string{time.Now().Format(time.RFC3339), strconv.Itoa(r.Stargazers)}
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
	client := &http.Client{}
	stats, err := readStats(client, "wurstmeister", "kafka-docker")
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

func readStats(client *http.Client, user, repo string) (*RepoResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", user, repo)
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't make request : %v", err)
	}
	defer resp.Body.Close()

	res, err := ReadJSON(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read response %v", err)
	}

	fmt.Printf("stargazers:%d\n", res.Stargazers)

	return res, nil
}

func appendToCsv(filename string, stats *RepoResponse) error {
	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("unable to open %s file for append", filename)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err = w.Write(stats.csv()); err != nil {
		return fmt.Errorf("unable to append stats to %s", filename)
	}
	w.Flush()

	return w.Error()
}

// CsvToTimeSeries reads a stats CSV and produces a go-chart chart.TimeSeries object
func CsvToTimeSeries(filename string) (*chart.TimeSeries, error) {
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

// RenderGraph takes a chart.TimeSeries and renders it to the specified filename
func RenderGraph(filename string, data *chart.TimeSeries) error {
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
		Series: []chart.Series{
			data,
		},
	}

	buffer := bytes.NewBuffer([]byte{})
	err := graph.Render(chart.PNG, buffer)
	if err != nil {
		return fmt.Errorf("unable to render PNG chart from stats: %v", err)
	}

	return ioutil.WriteFile(filename, buffer.Bytes(), 0644)
}

// ReadJSON reads input from io.Reader and produces a RepoResponse
func ReadJSON(r io.Reader) (*RepoResponse, error) {
	j, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("can't read source bytes: %v", err)
	}

	result := &RepoResponse{}
	err = json.Unmarshal(j, result)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshall JSON %v", err)
	}

	return result, nil
}
