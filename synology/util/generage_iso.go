package util

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/kdomanski/iso9660"
)

const userDataFileName string = "user-data"
const metaDataFileName string = "meta-data"
const networkConfigFileName string = "network-config"

type CloudInit struct {
	Name          string
	PoolName      string
	MetaData      string `yaml:"meta_data"`
	UserData      string `yaml:"user_data"`
	NetworkConfig string `yaml:"network_config"`
}

func IsoFromCloudInit(ci CloudInit) ([]byte, error) {
	writer, err := iso9660.NewWriter()

	if err != nil {
		log.Errorf("failed to create writer: %s", err)
	}

	if len(ci.MetaData) > 0 {
		if err = writer.AddFile(strings.NewReader(ci.MetaData), metaDataFileName); err != nil {
			log.Fatalf("failed to add metadata file: %s", err)
			return nil, err
		}
	}

	if len(ci.UserData) > 0 {
		if err = writer.AddFile(strings.NewReader(ci.UserData), userDataFileName); err != nil {
			log.Fatalf("failed to add metadata file: %s", err)
			return nil, err
		}
	}

	if len(ci.NetworkConfig) > 0 {
		if err = writer.AddFile(strings.NewReader(ci.NetworkConfig), networkConfigFileName); err != nil {
			log.Fatalf("failed to add network config file: %s", err)
			return nil, err
		}
	}

	var b bytes.Buffer
	err = writer.WriteTo(&b, fmt.Sprintf("vol%s", ci.Name))
	if err != nil {
		log.Fatalf("failed to write ISO image: %s", err)
		return nil, err
	}

	defer func() {
		_ = writer.Cleanup()
	}()

	return b.Bytes(), nil
}

func removeTmpIsoDirectory(iso string) {

	err := os.RemoveAll(filepath.Dir(iso))
	if err != nil {
		log.Printf("error while removing tmp directory holding the ISO file: %s", err)
	}
}

// log.Print("Creating ISO tmp directory")
// 	tmpDir, err := os.MkdirTemp("", "cloudinit")
// 	if err != nil {
// 		log.Fatalf("failed to create tmp directory: %s", err)
// 		return nil, err
// 	}

// 	outputFile, err := os.OpenFile(filepath.Join(tmpDir, fmt.Sprintf("%s.iso", ci.Name)), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
// 	if err != nil {
// 		log.Fatalf("failed to create file: %s", err)
// 		return nil, err
// 	}
