package virtualization

import (
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

func buildName(providerName, resourceName string) string {
	return providerName + "_virtualization_" + resourceName
}

func Resources() []func() resource.Resource {
	return []func() resource.Resource{
		NewImageResource,
		NewGuestResource,
	}
}

func DataSources() []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewGuestDataSource,
		NewGuestsDataSource,
	}
}
