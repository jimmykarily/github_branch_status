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
var statuses map[string]map[string]string
var client *http.Client

func init() {
	statuses = map[string]map[string]string{}
}

type Conf struct {
	GithubRepo                     string
	Branch                         string
	UpdateIntervalSeconds, Timeout int
	Port                           string
	GitHubToken                    string
}

func handler(w http.ResponseWriter, r *http.Request) {
	keys, ok := r.URL.Query()["context"]
	if !ok || len(keys[0]) < 1 {
		log.Println("Url Param 'context' is missing")
		return
	}

	// Query()["key"] will return an array of items,
	// we only want the single item.
	key := keys[0]

	if status, ok := statuses[string(key)]; ok {
		http.ServeFile(w, r, "images/"+status["state"]+".svg")
	} else {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, "Context not known")
	}
}

func parseConf() Conf {
	var err error
	var exists bool
	var interval, timeout string

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

	if timeout, exists = os.LookupEnv("CLIENT_TIMEOUT"); !exists {
		timeout = "30" // Seconds
	}

	if conf.Port, exists = os.LookupEnv("PORT"); !exists {
		conf.Port = "8080"
	}
	conf.UpdateIntervalSeconds, err = strconv.Atoi(interval)
	if err != nil {
		panic("GITHUB_STATUS_UPDATE_INTERVAL should be an integer: " + err.Error())
	}

	conf.Timeout, err = strconv.Atoi(timeout)
	if err != nil {
		panic("CLIENT_TIMEOUT should be an integer: " + err.Error())
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
	uri := "https://api.github.com/repos/" + conf.GithubRepo + "/commits/" + commitSha + "/status"
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

	var responseBody map[string]interface{}
	json.Unmarshal([]byte(body), &responseBody)

	for _, status := range responseBody["statuses"].([]interface{}) {
		statusMap := status.(map[string]interface{})

		state := statusMap["state"].(string)
		targetUrl := statusMap["target_url"].(string)
		context := statusMap["context"].(string)
		description := statusMap["description"].(string)

		statuses[context] = map[string]string{
			"state":       state,
			"targetUrl":   targetUrl,
			"context":     context,
			"description": description,
		}
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
	client = &http.Client{
		Timeout: time.Second * time.Duration(conf.Timeout),
	}

	go pollStatuses()

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+conf.Port, nil))
}
