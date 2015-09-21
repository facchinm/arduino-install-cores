package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
)

var (
	url     = flag.String("url", "http://downloads.arduino.cc/packages/package_index.json", "The url of the file json containing the package index")
	arduino = flag.String("arduino", "/usr/src/arduino/arduino", "The path of the arduino executable")
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

	for label, c := range couples {
		// Launch the command to install the boards
		cmd := exec.Command("xvfb-run", *arduino, "--install-boards", label)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}

		// Ensure that the correct folder exists
		cmd = exec.Command("mkdir", "-p /usr/src/"+c.Platform+"/hardware/")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}

		archFolder := "-rf /usr/src/" + c.Platform + "/hardware/" + c.Architecture

		// remove the old folders
		cmd = exec.Command("rm", archFolder)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}

		// Link the new folders
		installedFolder := "/home/vagrant/.arduino15/packages/" + c.Platform + "/hardware/" + c.Architecture
		cmd = exec.Command("ln", "-sf "+installedFolder+" "+archFolder)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Println(err.Error())
			os.Exit(1)
		}
	}

	os.Exit(0)
}
