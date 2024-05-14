package http

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
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

func Check(buff *bytes.Buffer, format string) Request {
	data := newRequest(buff, format)
	payload, err := json.Marshal(data)
	if err != nil {
		fmt.Println(err)
	}

	url := os.Getenv("CV_URL")

	if url == "" {
		fmt.Println("Please set env variable named CV_URL")
		os.Exit(1)
	}

	res, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		fmt.Println(err)
	}
	defer res.Body.Close()
	var r Request
	if err = json.NewDecoder(res.Body).Decode(&r); err != nil {
		fmt.Println("Unable to decode response")
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
