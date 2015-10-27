package main

import (
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var (
	url    = flag.String("url", "http://downloads.arduino.cc/packages/package_index.json", "The url of the file json containing the package index")
	folder = flag.String("folder", "/opt/cores", "The folder where to put the downloaded cores")
)

type index struct {
	Packages []struct {
		Name      string `json:"name"`
		Platforms []struct {
			Architecture string `json:"architecture"`
			Version      string `json:"version"`
			URL          string `json:"url"`
			Name         string `json:"archiveFileName"`
		} `json:"platforms"`
	} `json:"packages"`
}

func isError(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func download(url string, destination string) {
	err := os.MkdirAll(filepath.Dir(destination), 0755)
	isError(err)
	out, err := os.Create(destination)
	isError(err)
	resp, err := http.Get(url)
	isError(err)
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	isError(err)
}

func unpack(file string) {
	var cmd *exec.Cmd
	if strings.HasSuffix(file, "zip") {
		cmd = exec.Command("unzip", "-qq", filepath.Base(file))
	} else {
		cmd = exec.Command("tar", "xf", filepath.Base(file))
	}
	cmd.Dir = filepath.Dir(file)
	err := cmd.Run()
	isError(err)
	os.Remove(file)
}

func main() {

	flag.Parse()

	resp, _ := http.Get(*url)

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data index

	json.Unmarshal(body, &data)

	for _, p := range data.Packages {
		for _, a := range p.Platforms {
			destination, _ := filepath.Abs(filepath.Join(*folder, p.Name, a.Architecture, a.Name))
			log.Printf("Downloading %s:%s:%s in %s", p.Name, a.Architecture, a.Version, filepath.Dir(destination))
			download(a.URL, destination)
			unpack(destination)
		}
	}

	os.Exit(0)
}
