package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
)

type index struct {
	Packages []struct {
		Name      string `json:"name"`
		Platforms []struct {
			Architecture string `json:"architecture"`
		} `json:"platforms"`
	} `json:"packages"`
}

func main() {
	resp, _ := http.Get("http://downloads.arduino.cc/packages/package_index.json")

	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)

	var data index

	var couples = make(map[string]bool)

	json.Unmarshal(body, &data)

	for _, p := range data.Packages {
		for _, a := range p.Platforms {
			couples[p.Name+" "+a.Architecture] = true
		}
	}

	for c := range couples {
		cmd := exec.Command("/home/vagrant/ide_add_boards.sh", c)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	}

}
