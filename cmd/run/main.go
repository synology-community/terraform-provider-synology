package main

import (
	"encoding/json"

	client "github.com/appkins/terraform-provider-synology/synology-go"
	"github.com/appkins/terraform-provider-synology/synology-go/api/filestation"
)

func main() {

	host := "appkins.synology.me:5001" // os.Getenv("SYNOLOGY_HOST")
	user := "terraform"                // os.Getenv("SYNOLOGY_USER")
	password := "ach2vzw*dnx5BPV9njr"  // os.Getenv("SYNOLOGY_PASSWORD")

	client, err := client.New(host, true)

	if err != nil {
		panic(err)
	}

	err = client.Login(user, password, "webui")

	if err != nil {
		panic(err)
	}

	infoRequest := filestation.NewFileStationInfoRequest(2)
	infoResponse := filestation.FileStationInfoResponse{}

	err = client.Do(infoRequest, &infoResponse)

	if err != nil {
		panic(err)
	}

	println(infoResponse.Hostname)
	println(infoResponse.Supportsharing)

	listGuestResp, err := client.ListGuests()

	if err != nil {
		panic(err)
	}

	listGuestRespBytes, _ := json.Marshal(listGuestResp)

	println(string(listGuestRespBytes))

	for _, guest := range listGuestResp.Guests {
		println(guest.Name)
	}

	createFolder(client)
}

func createFolder(client client.Client) {
	resp, err := client.CreateFolder("/volume1/my_folder", "my_folder", true)

	if err != nil {
		panic(err)
	}

	for _, folder := range resp.Folders {
		println(folder.Path)
		println(folder.Name)
		println(folder.IsDir)
	}
}
