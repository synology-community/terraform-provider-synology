package main

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	client "github.com/synology-community/synology-api/package"
	"github.com/synology-community/synology-api/package/util/form"
)

type Options struct {
	Query   string `url:"q"`
	ShowAll bool   `url:"all"`
	Page    int    `url:"page"`
	Pages   []int  `url:"pages" del:","`
}

type File struct {
	Name    string `form:"name" url:"name"`
	Content string `form:"content" url:"content"`
}

type FileTest struct {
	Api     string `form:"api" url:"api"`
	Version string `form:"version" url:"version"`
	Method  string `form:"method" url:"method"`

	File File `form:"file" kind:"file"`
}

func main() {

	log.SetFormatter(&log.TextFormatter{DisableTimestamp: true, ForceColors: true})

	log.Info("Starting")

	host := "https://appkins.synology.me:5001" // os.Getenv("SYNOLOGY_HOST")
	user := "terraform"                        // os.Getenv("SYNOLOGY_USER")
	password := "ach2vzw*dnx5BPV9njr"          // os.Getenv("SYNOLOGY_PASSWORD")

	client, err := client.New(host, true)

	if err != nil {
		panic(err)
	}

	_, err = client.Login(user, password, "webui")

	if err != nil {
		panic(err)
	}

	_, err = client.FileStationAPI().Download("/projects", "WireGuard-v1000-1.0.20220627.spk")

	if err != nil {
		panic(err)
	}

	_, err = client.FileStationAPI().Upload("/data/foo/bar", &form.File{Name: "main.go", Content: "package main"}, true, true)

	if err != nil {
		panic(err)
	}

	if _, err := client.FileStationAPI().Upload("/data/foo/bar", &form.File{Name: "main.go", Content: "package main"}, true, true); err != nil {
		panic(err)
	}

	lgr, err := client.VirtualizationAPI().ListGuests()
	if err != nil {
		log.Error(err)
	}

	for _, guest := range lgr.Guests {
		var gm map[string]interface{}
		gb, _ := json.Marshal(&guest)
		err = json.Unmarshal(gb, &gm)
		if err != nil {
			log.Error(err)
		}
		log.WithFields(gm).Infof("Guest %s", guest.Name)

	}

	createFolder(client)
}

func createFolder(client client.SynologyClient) {
	resp, err := client.FileStationAPI().CreateFolder([]string{"/data/foo"}, []string{"bar"}, true)

	if err != nil {
		panic(err)
	}

	for _, folder := range resp.Folders {
		println(folder.Path)
		println(folder.Name)
		println(folder.IsDir)
	}
}
