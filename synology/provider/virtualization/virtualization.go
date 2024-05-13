package virtualization

func buildName(providerName, resourceName string) string {
	return providerName + "_virtualization_" + resourceName
}
