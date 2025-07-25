/*
   Copyright 2020 The Compose Specification Authors.

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package composetypes

// Services is a map of ServiceConfig.
type Services map[string]ServiceConfig

// GetProfiles retrieve the profiles implicitly enabled by explicitly targeting selected services.
func (s Services) GetProfiles() []string {
	set := map[string]struct{}{}
	for _, service := range s {
		for _, p := range service.Profiles {
			set[p] = struct{}{}
		}
	}
	var profiles []string
	for k := range set {
		profiles = append(profiles, k)
	}
	return profiles
}

func (s Services) Filter(predicate func(ServiceConfig) bool) Services {
	services := Services{}
	for name, service := range s {
		if predicate(service) {
			services[name] = service
		}
	}
	return services
}
