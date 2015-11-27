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
	url        = flag.String("url", "http://downloads.arduino.cc/packages/package_index.json", "The url of the file json containing the package index")
	coreFolder = flag.String("coreFolder", "/opt/cores", "The folder where to put the downloaded cores")
	toolFolder = flag.String("toolFolder", "/opt/tools", "The folder where to put the downloaded tools")
)

type core struct {
	Architecture string `json:"architecture"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	Name         string `json:"archiveFileName"`
	destination  string
	Dependencies []struct {
		Packager string `json:"packager"`
		Name     string `json:"name"`
		Version  string `json:"version"`
	} `json:"toolsDependencies"`
}

type tool struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Systems []struct {
		Host string `json:"host"`
		URL  string `json:"url"`
		Name string `json:"archiveFileName"`
	} `json:"systems"`
	url         string
	destination string
}

type index struct {
	Packages []struct {
		Name      string `json:"name"`
		Platforms []core `json:"platforms"`
		Tools     []tool `json:"tools"`
	} `json:"packages"`
}

func isError(err error, context string) {
	if err != nil {
		if context != "" {
			log.Println(context)
		}
		log.Fatal(err.Error())
	}
}

func download(url string, destination string) {
	err := os.MkdirAll(filepath.Dir(destination), 0755)
	isError(err, "")
	out, err := os.Create(destination)
	isError(err, "")
	resp, err := http.Get(url)
	isError(err, "")
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	isError(err, "")
}

func cleanup(directory string) {
	temp, err := ioutil.TempDir("", "")
	isError(err, "")
	files, err := ioutil.ReadDir(directory)
	isError(err, "")
	if 1 == len(files) {
		folder := filepath.Join(directory, files[0].Name())
		// Move to a tmp directory
		files, err = ioutil.ReadDir(folder)
		isError(err, "")
		for _, file := range files {
			var cmd *exec.Cmd
			cmd = exec.Command("mv", file.Name(), temp)
			cmd.Dir = folder
			output, err := cmd.CombinedOutput()
			isError(err, string(output))
		}
		err = os.RemoveAll(folder)
		isError(err, "")
		// move to the directory
		files, err = ioutil.ReadDir(temp)
		isError(err, "")
		for _, file := range files {
			var cmd *exec.Cmd
			cmd = exec.Command("mv", file.Name(), directory)
			cmd.Dir = temp
			output, err := cmd.CombinedOutput()
			isError(err, string(output))
		}
	}
}

func unpack(file string) {
	var cmd *exec.Cmd
	if strings.HasSuffix(file, "zip") {
		cmd = exec.Command("unzip", "-qq", filepath.Base(file))
	} else {
		cmd = exec.Command("tar", "xf", filepath.Base(file))
	}
	cmd.Dir = filepath.Dir(file)
	output, err := cmd.CombinedOutput()
	isError(err, string(output))
	os.Remove(file)
}

func main() {

	flag.Parse()

	resp, err := http.Get(*url)
	isError(err, "")

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	isError(err, "")

	var data index

	json.Unmarshal(body, &data)

	// Setup the packages to download
	cores := make(map[string]core)
	tools := make(map[string]tool)

	for _, p := range data.Packages {
		for _, a := range p.Platforms {
			destination, err := filepath.Abs(filepath.Join(*coreFolder, p.Name, a.Architecture, a.Name))
			isError(err, "")

			_, ok := cores[p.Name+":"+a.Architecture]

			if !ok || cores[p.Name+":"+a.Architecture].Version < a.Version {
				cores[p.Name+":"+a.Architecture] = core{
					Version:      a.Version,
					URL:          a.URL,
					destination:  destination,
					Dependencies: a.Dependencies,
				}
			}
		}
		for _, t := range p.Tools {
			tools[p.Name+":"+t.Name+":"+t.Version] = t
		}
	}

	// Download cores and tools
	for name, c := range cores {
		log.Printf("Downloading %s:%s in %s", name, c.Version, filepath.Dir(c.destination))
		download(c.URL, c.destination)
		log.Printf("Unpacking %s:%s in %s", name, c.Version, filepath.Dir(c.destination))
		unpack(c.destination)
		log.Printf("Cleanup %s", filepath.Dir(c.destination))
		cleanup(filepath.Dir(c.destination))
		for _, t := range c.Dependencies {
			tt := tools[t.Packager+":"+t.Name+":"+t.Version]
			for _, s := range tt.Systems {
				if "x86_64-linux-gnu" == s.Host || "x86_64-pc-linux-gnu" == s.Host {
					tt.destination, err = filepath.Abs(filepath.Join(*toolFolder, t.Name, t.Version, s.Name))
					isError(err, "")
					log.Printf("Downloading %s:%s in %s", t.Name, t.Version, filepath.Dir(tt.destination))
					download(s.URL, tt.destination)
					log.Printf("Unpacking %s:%s in %s", t.Name, t.Version, filepath.Dir(tt.destination))
					unpack(tt.destination)
					log.Printf("Cleanup %s", filepath.Dir(tt.destination))
					cleanup(filepath.Dir(tt.destination))
				}
			}
		}
		log.Println("------------------")
	}
	os.Exit(0)
}
