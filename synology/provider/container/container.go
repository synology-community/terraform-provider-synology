package container

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func buildName(providerName, resourceName string) string {
	if resourceName == "container" {
		return providerName + "_container"
	}

	return providerName + "_container_" + resourceName
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		NewProjectResource,
		NewNetworkResource,
	}
}

func DataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{}
}
