package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/xrash/smetrics"
	_ "io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	_ "os/exec"
	_ "path/filepath"
	"strconv"
	"strings"
)

var (
	url             = flag.String("url", "http://downloads.arduino.cc/packages/package_index.json", "The url of the file json containing the package index")
	coreName        = flag.String("core", "avr", "The core name: avr, arc32, etc")
	corePackager    = flag.String("packager", "arduino", "The packager name: arduino, Intel, etc")
	generateAntRule = flag.Bool("ant", false, "Set to true if you want to generate a string to copy in build.xml")
)

type core struct {
	Architecture string `json:"architecture"`
	Version      string `json:"version"`
	URL          string `json:"url"`
	Maintainer   string `json:"maintainer"`
	Name         string `json:"archiveFileName"`
	Checksum     string `json:"checksum"`
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
		Host     string `json:"host"`
		URL      string `json:"url"`
		Name     string `json:"archiveFileName"`
		Checksum string `json:"checksum"`
	} `json:"systems"`
	url         string
	destination string
}

type index struct {
	Packages []struct {
		Name       string `json:"name"`
		Maintainer string `json:"maintainer"`
		Platforms  []core `json:"platforms"`
		Tools      []tool `json:"tools"`
	} `json:"packages"`
}

var systems = map[string]string{
	"linuxamd64":  "x86_64-linux-gnu",
	"linux386":    "i686-linux-gnu",
	"darwinamd64": "apple-darwin",
	"windows386":  "i686-mingw32",
}

func isError(err error, context string) {
	if err != nil {
		if context != "" {
			log.Println(context)
		}
		log.Fatal(err.Error())
	}
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
			isError(err, "")

			_, ok := cores[p.Name+":"+a.Architecture]

			if !ok || cores[p.Name+":"+a.Architecture].Version < a.Version {
				cores[p.Name+":"+a.Architecture] = core{
					Version:      a.Version,
					Name:         a.Name,
					Architecture: a.Architecture,
					Maintainer:   p.Name,
					URL:          a.URL,
					Checksum:     strings.Split(a.Checksum, ":")[1],
					Dependencies: a.Dependencies,
				}
			}
		}
		for _, t := range p.Tools {
			tools[p.Name+":"+t.Name+":"+t.Version] = t
		}
	}

	if *generateAntRule {
		// Download cores and tools
		for _, c := range cores {
			if c.Maintainer == *corePackager && c.Architecture == *coreName {

				toolStr := "TOOL"
				toolIndex := 1

				for _, t := range c.Dependencies {

					tt := tools[t.Packager+":"+t.Name+":"+t.Version]

					if strings.Contains(t.Name, "gcc") || strings.Contains(tt.Systems[0].URL, "toolchain") {
						toolStr = "COMPILER"
					} else {
						toolStr = "TOOL" + strconv.Itoa(toolIndex)
						toolIndex++
					}

					fmt.Printf("<property name=\"BUNDLED_%s\" value=\"%s\"/>\n", toolStr, t.Name)
					fmt.Printf("<property name=\"BUNDLED_%s_VERSION\" value=\"%s\"/>\n", toolStr, t.Version)

					url := strings.Split(tt.Systems[0].URL, "/")
					fmt.Printf("<property name=\"BUNDLED_%s_BASEURL\" value=\"%s\"/>\n", toolStr, strings.Join(url[:len(url)-1], "/"))

					for _, s := range tt.Systems {

						cksum := strings.Split(s.Checksum, ":")[1]
						url_slc := strings.Split(s.URL, "/")
						url := url_slc[len(url_slc)-1]

						if smetrics.Jaro(s.Host, "x86_64-linux-gnu") > 0.9 {
							fmt.Printf("<property name=\"LINUX64_BUNDLED_%s_ARCHIVE\" value=\"%s\"/>\n", toolStr, url)
							fmt.Printf("<property name=\"LINUX64_BUNDLED_%s_SHA256\" value=\"%s\"/>\n", toolStr, cksum)
						} else if smetrics.Jaro(s.Host, "i686-linux-gnu") > 0.9 {
							fmt.Printf("<property name=\"LINUX32_BUNDLED_%s_ARCHIVE\" value=\"%s\"/>\n", toolStr, url)
							fmt.Printf("<property name=\"LINUX32_BUNDLED_%s_SHA256\" value=\"%s\"/>\n", toolStr, cksum)
						} else if smetrics.Jaro(s.Host, "arm-linux-gnueabihf") > 0.9 {
							fmt.Printf("<property name=\"LINUXARM_BUNDLED_%s_ARCHIVE\" value=\"%s\"/>\n", toolStr, url)
							fmt.Printf("<property name=\"LINUXARM_BUNDLED_%s_SHA256\" value=\"%s\"/>\n", toolStr, cksum)
						} else if strings.Contains(s.Host, "apple-darwin") {
							fmt.Printf("<property name=\"MACOSX_BUNDLED_%s_ARCHIVE\" value=\"%s\"/>\n", toolStr, url)
							fmt.Printf("<property name=\"MACOSX_BUNDLED_%s_SHA256\" value=\"%s\"/>\n", toolStr, cksum)
						} else if smetrics.Jaro(s.Host, "i686-mingw32") > 0.9 {
							fmt.Printf("<property name=\"WINDOWS_BUNDLED_%s_ARCHIVE\" value=\"%s\"/>\n", toolStr, url)
							fmt.Printf("<property name=\"WINDOWS_BUNDLED_%s_SHA256\" value=\"%s\"/>\n", toolStr, cksum)
						}
					}
				}
			}
		}
		fmt.Println()
		os.Exit(0)
	}

	// Download cores and tools
	for _, c := range cores {
		if c.Maintainer == *corePackager && c.Architecture == *coreName {
			fmt.Printf("-DBUNDLED_CORE=%s -DBUNDLED_CORE_MAINTAINER=%s ", *coreName, *corePackager)
			fmt.Printf("-DBUNDLED_CORE_VERSION=%s -DBUNDLED_CORE_ARCHIVE=%s ", c.Version, c.Name)
			fmt.Printf("-DBUNDLED_CORE_SHA256=%s -DBUNDLED_CORE_URL=%s ", c.Checksum, c.URL)

			toolStr := "TOOL"
			toolIndex := 1

			for _, t := range c.Dependencies {

				tt := tools[t.Packager+":"+t.Name+":"+t.Version]

				if strings.Contains(t.Name, "gcc") || strings.Contains(tt.Systems[0].URL, "toolchain") {
					toolStr = "COMPILER"
				} else {
					toolStr = "TOOL" + strconv.Itoa(toolIndex)
					toolIndex++
				}

				fmt.Printf("-DBUNDLED_%s=%s -DBUNDLED_%s_VERSION=%s ", toolStr, t.Name, toolStr, t.Version)

				url := strings.Split(tt.Systems[0].URL, "/")
				fmt.Printf("-DBUNDLED_%s_BASEURL=%s ", toolStr, strings.Join(url[:len(url)-1], "/"))

				for _, s := range tt.Systems {

					cksum := strings.Split(s.Checksum, ":")[1]
					url_slc := strings.Split(s.URL, "/")
					url := url_slc[len(url_slc)-1]

					if smetrics.Jaro(s.Host, "x86_64-linux-gnu") > 0.9 {
						fmt.Printf("-DLINUX64_BUNDLED_%s_ARCHIVE=%s ", toolStr, url)
						fmt.Printf("-DLINUX64_BUNDLED_%s_SHA256=%s ", toolStr, cksum)
					} else if smetrics.Jaro(s.Host, "i686-linux-gnu") > 0.9 {
						fmt.Printf("-DLINUX32_BUNDLED_%s_ARCHIVE=%s ", toolStr, url)
						fmt.Printf("-DLINUX32_BUNDLED_%s_SHA256=%s ", toolStr, cksum)
					} else if smetrics.Jaro(s.Host, "arm-linux-gnueabihf") > 0.9 {
						fmt.Printf("-DLINUXARM_BUNDLED_%s_ARCHIVE=%s ", toolStr, url)
						fmt.Printf("-DLINUXARM_BUNDLED_%s_SHA256=%s ", toolStr, cksum)
					} else if strings.Contains(s.Host, "apple-darwin") {
						fmt.Printf("-DMACOSX_BUNDLED_%s_ARCHIVE=%s ", toolStr, url)
						fmt.Printf("-DMACOSX_BUNDLED_%s_SHA256=%s ", toolStr, cksum)
					} else if smetrics.Jaro(s.Host, "i686-mingw32") > 0.9 {
						fmt.Printf("-DWINDOWS_BUNDLED_%s_ARCHIVE=%s ", toolStr, url)
						fmt.Printf("-DWINDOWS_BUNDLED_%s_SHA256=%s ", toolStr, cksum)
					}
				}
			}
		}
	}
	fmt.Println()
	os.Exit(0)
}
