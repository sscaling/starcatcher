package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
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
	statFile := "stats.csv"
	client := &http.Client{}
	stats, err := readStats(client, "wurstmeister", "kafka-docker")
	if err != nil {
		log.Fatalf("Unable to read stats: %v\n", err)
	}

	file, err := os.OpenFile(statFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		log.Fatalf("unable to open %s file for append\n", statFile)
	}
	defer file.Close()

	w := csv.NewWriter(file)
	if err = w.Write(stats.csv()); err != nil {
		log.Fatalf("unable to append stats to %s\n", statFile)
	}

	w.Flush()

	if err := w.Error(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("success")
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
