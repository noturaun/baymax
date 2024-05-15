package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
)

const (
	MAVEN = "maven"
	NPM   = "npm"
)

type Request struct {
	Components []Components `json:"components"`
}

type Components struct {
	ComponentIdentifier ComponentIdentifier `json:"componentIdentifier"`
}

type ComponentIdentifier struct {
	Format      string  `json:"format"`
	Coordinates Package `json:"coordinates"`
	Status      string  `json:"status"`
	Detection   string  `json:"detection"`
	Notes       string  `json:"notes"`
	ThreadLevel string  `json:"threadLevel"`
}

type Package struct {
	GroupId    string `json:"groupId"`
	ArtifactId string `json:"artifactId"`
	PackageId  string `json:"packageId"`
	Extension  string `json:"extension"`
	Version    string `json:"version"`
}

func Check(buff *bytes.Buffer, format string, path string, withProxy bool) Request {
	data := newRequest(buff, format)
	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}

	var proxyUrl string
	var transport *http.Transport
	if withProxy {
		proxyUrl = os.Getenv("BAYMAX_PROXY_URL")
		if proxyUrl != "" {
			uri, err := url.Parse(proxyUrl)
			if err != nil {
				fmt.Println("Error when parsing url")
				os.Exit(1)
			}
			transport = &http.Transport{
				Proxy: http.ProxyURL(uri),
			}
		}
	}

	cvUrl := os.Getenv("CV_URL")
	if cvUrl == "" {
		fmt.Println("Please set env variable named CV_URL")
		os.Exit(1)
	}

	var client *http.Client
	if withProxy {
		client = &http.Client{
			Transport: transport,
		}
	} else {
		client = http.DefaultClient
	}

	req, err := http.NewRequest("POST", cvUrl, bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Unable to request")
		os.Exit(1)
	}

	var r Request
	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		fmt.Println("Unable to decode response")
		os.Exit(1)
	}

	jsonData, err := json.MarshalIndent(r, "", "    ")

	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		os.Exit(1)
	}

	dir := path + "/.cakra"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.Mkdir(dir, 0755)
		if err != nil {
			fmt.Println("Error creating directory:", err)
			os.Exit(1)
		}
	}

	fn := dir + "/data.json"
	err = os.WriteFile(fn, jsonData, 0644)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		os.Exit(1)
	}

	return r
}

func newRequest(buff *bytes.Buffer, format string) Request {

	var comps []Components
	out := bufio.NewReader(buff)
	for {
		line, err := out.ReadString('\n')
		if err != nil {
			break
		}
		line = strings.Replace(line, "\n", "", 1)
		sp := strings.Split(line, ":")
		if len(sp) == 4 && format == MAVEN {
			ci := ComponentIdentifier{
				Format: format,
				Coordinates: Package{
					GroupId:    sp[0],
					ArtifactId: sp[1],
					Extension:  sp[2],
					Version:    sp[3],
				},
			}

			comps = append(comps, Components{
				ComponentIdentifier: ci,
			})
		} else if len(sp) == 2 && format == NPM {
			ci := ComponentIdentifier{
				Format: format,
				Coordinates: Package{
					PackageId: sp[0],
					Version:   sp[3],
				},
			}

			comps = append(comps, Components{
				ComponentIdentifier: ci,
			})
		}
	}
	return Request{comps}
}
