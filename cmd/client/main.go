package main

import (
	log "github.com/sirupsen/logrus"

	"github.com/appkins/terraform-provider-synology/synology/util"
)

func main() {
	cloudInit := util.CloudInit{}

	b, err := util.IsoFromCloudInit(cloudInit)
	if err != nil {
		panic(err)
	}

	log.Info(string(b))

}
