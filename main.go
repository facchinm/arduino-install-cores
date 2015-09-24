package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

var (
	url     = flag.String("url", "http://downloads.arduino.cc/packages/package_index.json", "The url of the file json containing the package index")
	arduino = flag.String("arduino", "/usr/src/arduino/arduino", "The path of the arduino executable")
	home    = flag.String("home", "/home/vagrant/.arduino15", "The path where the arduino executable downloads the cores")
)

type index struct {
	Packages []struct {
		Name      string `json:"name"`
		Platforms []struct {
			Architecture string `json:"architecture"`
		} `json:"platforms"`
	} `json:"packages"`
}

type couple struct {
	Platform     string
	Architecture string
}

func main() {
	resp, _ := http.Get(*url)

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data index

	var couples = make(map[string]*couple)

	json.Unmarshal(body, &data)

	for _, p := range data.Packages {
		for _, a := range p.Platforms {
			c := couple{Platform: p.Name, Architecture: a.Architecture}
			couples[c.Platform+":"+c.Architecture] = &c
		}
	}

	var cmd *exec.Cmd
	var err error
	var children []os.FileInfo
	var version string

	for label, c := range couples {
		// Launch the command to install the boards
		cmd = exec.Command("xvfb-run", *arduino, "--install-boards", label)
		log.Println(cmd.Args)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println(err.Error())
			// There's no need to exit if an architecture has already been installed
		}

		// Ensure that the correct folder exists
		hardwareFolder := filepath.Join("/usr/src/", c.Platform, "/hardware/")
		err = os.MkdirAll(hardwareFolder, 0777)

		if err != nil {
			log.Println("Error during creation of folder " + hardwareFolder)
			log.Println(err.Error())
			os.Exit(1)
		}

		archFolder := filepath.Join(hardwareFolder, c.Architecture)

		// remove the old folders
		err = os.RemoveAll(archFolder)

		if err != nil {
			log.Println("Error during removal of folder " + archFolder)
			log.Println(err.Error())
			os.Exit(1)
		}

		// Get the version
		installedFolder := filepath.Join(*home, "/packages/", c.Platform, "/hardware/", c.Architecture)

		children, err = ioutil.ReadDir(installedFolder)

		if err != nil {
			log.Println("Error during reading of folder " + installedFolder)
			log.Println(err.Error())
			os.Exit(1)
		}

		if len(children) > 0 {
			version = children[0].Name()
			err = os.Symlink(filepath.Join(installedFolder, string(version)), archFolder)
		}

		if err != nil {
			log.Println("Error during linking of folder " + filepath.Join(installedFolder, string(version)))
			log.Println(err.Error())
			os.Exit(1)
		}
	}

	os.Exit(0)
}
