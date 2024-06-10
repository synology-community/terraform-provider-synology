package main

import (
	"fmt"
	"log"

	"github.com/compose-spec/compose-go/v2/types"
)

func main() {
	// projectName := "my_project"
	// ctx := context.Background()

	// cli.ProjectFromOptions(ctx, nil)

	project := types.Project{
		Configs: types.Configs{
			"config1": types.ConfigObjConfig{
				Name: "config1",
				File: "config1",
			},
			"config2": types.ConfigObjConfig{
				Name: "config2",
				File: "config2",
			},
		},
		Networks: types.Networks{
			"network1": types.NetworkConfig{
				Name: "network1",
			},
		},
		Services: types.Services{
			"service1": types.ServiceConfig{
				Name:  "service1",
				Image: "image1",
			},
		},
	}
	// project.MarshalYAML()

	// options, err := cli.NewProjectOptions(
	// 	[]string{},
	// 	cli.WithOsEnv,
	// 	cli.WithDotEnv,
	// 	cli.WithName(projectName),
	// )
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// project, err := cli.ProjectFromOptions(ctx, options)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Use the MarshalYAML method to get YAML representation
	projectYAML, err := project.MarshalYAML()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(string(projectYAML))
}
