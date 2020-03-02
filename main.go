package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var conf Conf
var statuses map[string]string
var client *http.Client

func init() {
	statuses = map[string]string{}
}

type Conf struct {
	GithubRepo            string
	Branch                string
	UpdateIntervalSeconds int
	Port                  string
	GitHubToken           string
}

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hi there, I love %s!", r.URL.Path[1:])
}

func parseConf() Conf {
	var err error
	var exists bool
	var interval string

	if conf.GitHubToken, exists = os.LookupEnv("GITHUB_TOKEN"); !exists {
		panic("GITHUB_TOKEN not set!")
	}
	if conf.GithubRepo, exists = os.LookupEnv("GITHUB_REPO"); !exists {
		panic("GITHUB_REPO not set!")
	}
	if conf.Branch, exists = os.LookupEnv("GIT_BRANCH"); !exists {
		conf.Branch = "master"
	}
	if interval, exists = os.LookupEnv("GITHUB_STATUS_UPDATE_INTERVAL"); !exists {
		interval = "30" // Seconds
	}
	if conf.Port, exists = os.LookupEnv("PORT"); !exists {
		conf.Port = "8080"
	}
	conf.UpdateIntervalSeconds, err = strconv.Atoi(interval)
	if err != nil {
		panic("GITHUB_STATUS_UPDATE_INTERVAL should be an integer: " + err.Error())
	}

	return conf
}

// Gets the commit sha of the tip of the GIT_BRANCH
func getBranchTip() (string, error) {
	uri := "https://api.github.com/repos/" + conf.GithubRepo + "/branches/" + conf.Branch
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return "", err
	}
	req.Header.Add("Authorization", "token "+conf.GitHubToken)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)

	commit, ok := result["commit"].(map[string]interface{})
	if !ok {
		log.Fatal("Could not unmarshal key 'commit' in response: ", string(body))
	}
	commitSha, ok := commit["sha"].(string)

	return commitSha, nil
}

// Get's the commit statuses (all contexts) of the tip of GIT_BRANCH
func updateStatuses() error {
	commitSha, err := getBranchTip()
	if err != nil {
		return err
	}
	uri := "https://api.github.com/repos/" + conf.GithubRepo + "/commits/" + commitSha + "/statuses"
	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}
	req.Header.Add("Authorization", "token "+conf.GitHubToken)
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)

	var newStatuses []map[string]string
	json.Unmarshal([]byte(body), &newStatuses)

	for _, status := range newStatuses {
		state := status["state"]
		context := status["context"]
		statuses[context] = state
	}

	return nil
}

func pollStatuses() {
	log.Print("I will start polling loop now")
	for {
		err := updateStatuses()
		if err != nil {
			log.Fatal("An error occured: ", err.Error())
		}
		log.Print(statuses)
		time.Sleep(time.Duration(conf.UpdateIntervalSeconds) * time.Second)
	}
}

func main() {
	parseConf()
	client = &http.Client{}

	go pollStatuses()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+conf.Port, nil))
}
