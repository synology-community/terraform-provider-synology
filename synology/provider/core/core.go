package core

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func buildName(providerName, resourceName string) string {
	return providerName + "_core_" + resourceName
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		NewPackageResource,
		NewPackageFeedResource,
	}
}

func DataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		// NewPackagesDataSource,
	}
}
