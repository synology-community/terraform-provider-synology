package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/kdomanski/iso9660"
)

const userDataFileName string = "user-data"
const metaDataFileName string = "meta-data"
const networkConfigFileName string = "network-config"

type CloudInit struct {
	MetaData      string `yaml:"meta_data"`
	UserData      string `yaml:"user_data"`
	NetworkConfig string `yaml:"network_config"`
}

func IsoFromFiles(ctx context.Context, isoName string, files map[string]string) (string, error) {
	writer, err := iso9660.NewWriter()

	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to create writer: %v", err))
	}

	for path, content := range files {
		tflog.Info(ctx, fmt.Sprintf("writing iso file for %s", path))

		if len(path) > 0 {
			if err = writer.AddFile(strings.NewReader(content), path); err != nil {
				tflog.Error(ctx, fmt.Sprintf("failed to add metadata file: %v", err))
				return "", err
			}
		}
	}

	var b bytes.Buffer
	err = writer.WriteTo(&b, isoName)
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to write ISO image: %s", err))
		return "", err
	}

	defer func() {
		_ = writer.Cleanup()
	}()

	return b.String(), nil
}

func IsoFromCloudInit(ctx context.Context, ci CloudInit) (string, error) {
	fileMap := map[string]string{}
	if ci.MetaData != "" {
		fileMap[metaDataFileName] = ci.MetaData
	} else {
		fileMap[metaDataFileName] = ""
	}
	if ci.UserData != "" {
		fileMap[userDataFileName] = ci.UserData
	}
	if ci.NetworkConfig != "" {
		fileMap[networkConfigFileName] = ci.NetworkConfig
	}

	return IsoFromFiles(ctx, "cidata", fileMap)
}

func removeTmpIsoDirectory(ctx context.Context, iso string) {

	err := os.RemoveAll(filepath.Dir(iso))
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("error while removing tmp directory holding the ISO file: %s", err))
	}
}

// tflog.Print("Creating ISO tmp directory")
// 	tmpDir, err := os.MkdirTemp("", "cloudinit")
// 	if err != nil {
// 		tflog.Fatalf("failed to create tmp directory: %s", err)
// 		return nil, err
// 	}

// 	outputFile, err := os.OpenFile(filepath.Join(tmpDir, fmt.Sprintf("%s.iso", ci.Name)), os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
// 	if err != nil {
// 		tflog.Fatalf("failed to create file: %s", err)
// 		return nil, err
// 	}
