package util

// import (
// 	"context"
// 	"fmt"
// 	"log"

// 	"github.com/compose-spec/compose-go/v2/cli"
// )

// func createCompose() {
// 	composeFilePath := "docker-compose.yml"
// 	projectName := "my_project"
// 	ctx := context.Background()

// 	options, err := cli.NewProjectOptions(
// 		[]string{composeFilePath},
// 		cli.WithOsEnv,
// 		cli.WithDotEnv,
// 		cli.WithName(projectName),
// 	)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	project, err := cli.ProjectFromOptions(ctx, options)
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	// Use the MarshalYAML method to get YAML representation
// 	projectYAML, err := project.MarshalYAML()
// 	if err != nil {
// 		log.Fatal(err)
// 	}

// 	fmt.Println(string(projectYAML))
// }
