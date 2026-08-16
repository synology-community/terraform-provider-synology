package util

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/kdomanski/iso9660"
)

const (
	userDataFileName      string = "user-data"
	metaDataFileName      string = "meta-data"
	networkConfigFileName string = "network-config"
)

type CloudInit struct {
	MetaData      string `yaml:"meta_data"`
	UserData      string `yaml:"user_data"`
	NetworkConfig string `yaml:"network_config"`
}

func IsoFromFiles(
	ctx context.Context,
	volumeIdentifier string,
	files map[string]string,
) ([]byte, error) {
	writer, err := iso9660.NewWriter()
	if err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to create writer: %v", err))
	}

	for path, content := range files {
		tflog.Info(ctx, fmt.Sprintf("writing iso file for %s", path))

		if len(path) > 0 {
			if err = writer.AddFile(strings.NewReader(content), path); err != nil {
				tflog.Error(ctx, fmt.Sprintf("failed to add metadata file: %v", err))
				return nil, err
			}
		}
	}

	defer func() {
		_ = writer.Cleanup()
	}()

	var b bytes.Buffer
	if err = writer.WriteTo(&b, volumeIdentifier); err != nil {
		tflog.Error(ctx, fmt.Sprintf("failed to write ISO image: %s", err))
		return nil, err
	}

	return b.Bytes(), nil
}

func IsoFromCloudInit(ctx context.Context, ci CloudInit) ([]byte, error) {
	fileMap := map[string]string{}
	if ci.MetaData != "" {
		fileMap[metaDataFileName] = ci.MetaData
	} else {
		fileMap[metaDataFileName] = ""
	}
	if ci.UserData != "" {
		// cloud-init identifies its user-data format from a magic first line:
		// "#cloud-config" for a single YAML document, or a MIME multipart
		// message (e.g. produced by Terraform's cloudinit_config data source,
		// which always renders multipart/mixed even for a single part) that
		// already declares its own "Content-Type"/"MIME-Version" header.
		// Blindly prepending "#cloud-config" to an already-multipart document
		// corrupts it: cloud-init then reads the whole MIME envelope as one
		// YAML doc and silently skips user-data processing entirely.
		if match, _ := regexp.MatchString(`(?i)^(#cloud-config|content-type:|mime-version:)`, ci.UserData); !match {
			ci.UserData = fmt.Sprintf("#cloud-config\n%s", ci.UserData)
		}

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
		tflog.Error(
			ctx,
			fmt.Sprintf("error while removing tmp directory holding the ISO file: %s", err),
		)
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
