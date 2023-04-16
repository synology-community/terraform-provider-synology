Synology API Go client
======================

Synology-go is a Go library for accessing [Synology DSM](https://www.synology.com/en-eu/support/developer#tool)
system via HTTP API.

# Install

```bash
go get github.com/maksym-nazarenko/terraform-provider-synology/synology-go@v0.0.1
```

or
```bash
go get github.com/maksym-nazarenko/terraform-provider-synology/synology-go
```
for the latest version.

# Usage

```go
package main

import (
	"log"

	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api"
	"github.com/maksym-nazarenko/terraform-provider-synology/synology-go/api/filestation"
)

func main() {
	skipCertificateVerification := true
	c, err := New("synology-server:5001", skipCertificateVerification)
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Login("api-client", "password", "webui"); err != nil {
		log.Fatal(err)
	}

	req := filestation.NewCreateFolderRequest(2).
		WithFolderPath("/test-folder").
		WithName("folder_name")

	resp := filestation.CreateFolderResponse{}
	err = c.Do(r, &resp)
	if err != nil {
		log.Fatal(err)
	}

	if !resp.Success() {
		log.Fatal(resp.GetError())
	}

	log.Printf("Created %d folder(s)\n", len(resp.Folders))
}
```

# Supported APIs

|API|Min version|Method|Description|
|---|---|---|---|
|SYNO.FileStation.CreateFolder|2|`create`|Create folders|
|SYNO.FileStation.Info|2|`get`|Provide File Station information|
|SYNO.FileStation.Rename|2|`rename`|Rename a file/folder|
