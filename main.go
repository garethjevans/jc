package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	job string
)

func main() {
	flag.StringVar(&job, "job", "", "Name of the jenkins job")
	flag.Parse()

	args := flag.Args()

	jenkinsHost := os.Getenv("JENKINS_HOST_URL")
	jenkinsUsername := os.Getenv("JENKINS_USERNAME")
	jenkinsApiToken := os.Getenv("JENKINS_API_TOKEN")

	argsMap := make(map[string]string)
	for _, v := range args {
		parts := strings.SplitN(v, "=", 2)
		argsMap[parts[0]] = parts[1]
	}

	jar, err := cookiejar.New(nil)
	if err != nil {
		panic(err)
	}

	client := http.Client{
		Jar: jar,
	}

	credentials := Credentials{
		Username: jenkinsUsername,
		ApiToken: jenkinsApiToken,
	}

	c, err := GetCrumb(client, jenkinsHost, credentials)
	if err != nil {
		panic(err)
	}

	credentials.Crumb = c

	buildNumber := TriggerBuild(client, jenkinsHost, credentials, job, argsMap)

	moreData := true
	var textSize int
	var buildInfo BuildInfo

	for {
		buildInfo, err = GetBuildInfo(client, jenkinsHost, credentials, job, buildNumber)
		if err != nil {
			panic(err)
		}

		moreData, textSize, err = GetBuildLog(client, jenkinsHost, credentials, job, buildNumber, textSize)
		if err != nil {
			panic(err)
		}

		if !moreData && !buildInfo.Building {
			break
		}

		time.Sleep(2 * time.Second)
	}

	if buildInfo.Result != "SUCCESS" {
		log.Fatalf("Unexpected build result %s", buildInfo.Result)
	}
}

func TriggerBuild(client http.Client, jenkinsHost string, credentials Credentials, jenkinsJob string, args map[string]string) int {
	params := url.Values{}
	for k, v := range args {
		params.Add(k, v)
	}
	paramsString := params.Encode()

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/job/%s/buildWithParameters?%s", jenkinsHost, jenkinsJob, paramsString), nil)
	if err != nil {
		panic(err)
	}

	resp, err := DoRequest(client, req, credentials)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	queueInfo, err := GetQueueInfo(client, credentials, resp.Header.Get("Location"))
	if err != nil {
		panic(err)
	}
	return queueInfo.QueueExecutable.Number
}

func GetQueueInfo(client http.Client, credentials Credentials, location string) (QueueInfo, error) {
	queueInfo := QueueInfo{}

	req, err := http.NewRequest("GET", location+"/api/json", nil)
	if err != nil {
		return queueInfo, err
	}

	resp, err := DoRequest(client, req, credentials)
	if err != nil {
		return queueInfo, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return queueInfo, err
	}

	err = json.Unmarshal(body, &queueInfo)
	if err != nil {
		return queueInfo, err
	}

	return queueInfo, err
}

func GetBuildLog(client http.Client, jenkinsHost string, credentials Credentials, jenkinsJob string, buildNumber int, start int) (bool, int, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/job/%s/%d/logText/progressiveText?start=%d", jenkinsHost, jenkinsJob, buildNumber, start), nil)
	if err != nil {
		return false, 0, err
	}

	resp, err := DoRequest(client, req, credentials)
	if err != nil {
		return false, 0, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, 0, err
	}

	if strings.TrimSpace(string(body)) != "" {
		fmt.Println(string(body))
	}

	moreDataString := resp.Header.Get("X-More-Data")
	textSizeString := resp.Header.Get("X-Text-Size")

	textSize, err := strconv.Atoi(textSizeString)
	if err != nil {
		return false, 0, err
	}
	var moreData bool
	if moreDataString != "" {
		moreData, err = strconv.ParseBool(moreDataString)
		if err != nil {
			return false, 0, err
		}
	} else {
		moreData = false
	}

	return moreData, textSize, nil
}

func GetBuildInfo(client http.Client, jenkinsHost string, credentials Credentials, jenkinsJob string, buildNumber int) (BuildInfo, error) {
	buildInfo := BuildInfo{}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/job/%s/%d/api/json", jenkinsHost, jenkinsJob, buildNumber), nil)
	if err != nil {
		return buildInfo, err
	}

	resp, err := DoRequest(client, req, credentials)
	if err != nil {
		return buildInfo, err
	}

	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return buildInfo, err
	}

	err = json.Unmarshal(body, &buildInfo)
	if err != nil {
		return buildInfo, err
	}

	return buildInfo, nil
}

func GetCrumb(client http.Client, jenkinsHost string, credentials Credentials) (Crumb, error) {
	crb := Crumb{}

	url := fmt.Sprintf("%s/crumbIssuer/api/json", jenkinsHost)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return crb, err
	}

	resp, err := DoRequest(client, req, credentials)
	if err != nil {
		return crb, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return crb, err
	}

	err = json.Unmarshal(body, &crb)
	if err != nil {
		return crb, err
	}

	return crb, nil
}

func DoRequest(client http.Client, req *http.Request, credentials Credentials) (*http.Response, error) {
	if credentials.Crumb.Crumb != "" && credentials.Crumb.CrumbRequestField != "" {
		req.Header.Set(credentials.Crumb.CrumbRequestField, credentials.Crumb.Crumb)
	}

	req.SetBasicAuth(credentials.Username, credentials.ApiToken)

	resp, err := client.Do(req)
	if err != nil {
		return resp, fmt.Errorf("unable to perform request: %v", err)
	}

	return resp, err
}

type Crumb struct {
	Crumb             string `json:"crumb"`
	CrumbRequestField string `json:"crumbRequestField"`
}

type QueueInfo struct {
	QueueExecutable QueueExecutable `json:"executable"`
}

type QueueExecutable struct {
	Number int `json:"number"`
}

type BuildInfo struct {
	Building bool   `json:"building"`
	Result   string `json:"result"`
}

type Credentials struct {
	Username string
	ApiToken string
	Crumb    Crumb
}
