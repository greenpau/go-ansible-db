// Copyright 2018 Paul Greenberg (greenpau@outlook.com)
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package db

import "sort"

type stringFloatMap struct {
	m map[string]float64
	s []string
}

func (sfm *stringFloatMap) Len() int {
	return len(sfm.m)
}

func (sfm *stringFloatMap) Less(i, j int) bool {
	return sfm.m[sfm.s[i]] > sfm.m[sfm.s[j]]
}

func (sfm *stringFloatMap) Swap(i, j int) {
	sfm.s[i], sfm.s[j] = sfm.s[j], sfm.s[i]
}

func sortStringFloatMap(m map[string]float64) []string {
	sfm := new(stringFloatMap)
	sfm.m = m
	sfm.s = make([]string, len(m))
	i := 0
	for k := range m {
		sfm.s[i] = k
		i++
	}
	sort.Sort(sfm)
	return sfm.s
}
