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

import "fmt"

// Options is a mapping type for options we pass as-is to container runtime.
type Options map[string]string

func (d *Options) DecodeMapstructure(value any) error {
	switch v := value.(type) {
	case map[string]any:
		m := make(map[string]string)
		for key, e := range v {
			if e == nil {
				m[key] = ""
			} else {
				m[key] = fmt.Sprint(e)
			}
		}
		*d = m
	case map[string]string:
		*d = v
	default:
		return fmt.Errorf("invalid type %T for options", value)
	}
	return nil
}
