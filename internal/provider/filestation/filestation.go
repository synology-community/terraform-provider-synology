package filestation

func buildName(providerName, resourceName string) string {
	return providerName + "_filestation_" + resourceName
}
