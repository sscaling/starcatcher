package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
)

// RepoResponse represents the github repo information
type RepoResponse struct {
	Stargazers int32 `json:"stargazers_count"`
}

// ReadJSON reads input from io.Reader and produces a RepoResponse
func ReadJSON(r io.Reader) (*RepoResponse, error) {
	jsonBlob, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, fmt.Errorf("can't read source bytes: %v", err)
	}

	result := &RepoResponse{}
	err = json.Unmarshal(jsonBlob, result)
	if err != nil {
		return nil, fmt.Errorf("can't unmarshall JSON %v", err)
	}

	return result, nil
}

func main() {
	fmt.Println("Star-catcher")

	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.github.com/repos/wurstmeister/kafka-docker", nil)
	req.Header.Add("Accept", "application/vnd.github.v3+json")
	resp, err := client.Do(req)
	if err != nil {
		log.Fatalf("Can't make request : %v\n", err)
	}
	defer resp.Body.Close()

	res, err := ReadJSON(resp.Body)
	if err != nil {
		log.Fatalf("Can't read response %v\n", err)
	}

	fmt.Printf("stargazers:%d\n", res.Stargazers)
}
