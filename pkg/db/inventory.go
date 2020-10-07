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

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"regexp"
	"strings"
	"sync/atomic"
)

// Inventory is the contents of Ansible inventory file.
type Inventory struct {
	Raw       []byte
	HostsRef  map[string]string
	GroupsRef map[string]bool
	Hosts     []*InventoryHost
	Groups    []*InventoryGroup
}

// InventoryHost is a host in Ansible inventory
type InventoryHost struct {
	Name        string
	Parent      string
	Variables   map[string]string
	Groups      []string
	GroupChains []string
}

// InventoryGroup is an group of InventoryHost instances.
type InventoryGroup struct {
	Name      string
	Ancestors []string
	Variables map[string]string
	Counters  struct {
		Hosts  uint64
		Groups uint64
	}
}

// NewInventory returns a pointer to Inventory.
func NewInventory() *Inventory {
	g := &InventoryGroup{
		Name:      "all",
		Variables: make(map[string]string),
		Ancestors: []string{},
	}
	inv := &Inventory{
		HostsRef:  make(map[string]string),
		GroupsRef: make(map[string]bool),
	}
	inv.GroupsRef["all"] = true
	inv.Groups = append(inv.Groups, g)
	return inv
}

// Size returns the number of hosts in the Inventory.
func (inv *Inventory) Size() uint64 {
	return uint64(len(inv.Hosts))
}

func (inv *Inventory) parseString(s string) error {
	// Sections are default (0), group (1), children (2), and variables (3)
	var sectionType int
	groupName := "all"
	lines := strings.Split(s, "\n")
	for lc, line := range lines {
		orig := line
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			line = strings.TrimRight(line, "]")
			line = strings.TrimLeft(line, "[")
			kv := strings.Split(line, ":")
			if len(kv) > 2 {
				return fmt.Errorf("invalid section: %s", orig)
			}
			groupName = kv[0]
			if err := inv.AddGroup(groupName, "all"); err != nil {
				return fmt.Errorf("AddGroup() failed: %s, line: %d", err, lc)
			}
			if len(kv) == 1 {
				sectionType = 1
				continue
			}
			switch kv[1] {
			case "children":
				sectionType = 2
			case "vars":
				sectionType = 3
			default:
				return fmt.Errorf("invalid section: %s", orig)
			}
			continue
		}

		switch sectionType {
		case 0:
			// default, group all
			if err := inv.AddHost(line, "all"); err != nil {
				return fmt.Errorf("AddHost() failed: %s, line: %d", err, lc)
			}
		case 1:
			// group section, contains individual hosts
			if err := inv.AddHost(line, groupName); err != nil {
				return fmt.Errorf("AddHost() failed: %s, line: %d", err, lc)
			}
		case 2:
			// children section
			if err := inv.AddGroup(line, groupName); err != nil {
				return fmt.Errorf("AddGroup() failed: %s, line: %d", err, lc)
			}
		case 3:
			// group variables
			if err := inv.AddVariable(line, groupName); err != nil {
				return fmt.Errorf("AddVariable() failed: %s, line: %d", err, lc)
			}
		default:
			return fmt.Errorf("invalid section type: %d", sectionType)
		}
	}

	for _, h := range inv.Hosts {
		groupChains, groups, err := inv.GetParentGroupChains(h.Parent)
		if err != nil {
			return fmt.Errorf("the search for parent group chains for host '%s' erred: %s", h.Name, err)
		}
		if len(groupChains) < 1 {
			return fmt.Errorf("parent group for host '%s' not found", h.Name)
		}
		for _, g := range groups {
			if err := inv.AddGroupMemberCounter("host", g); err != nil {
				return fmt.Errorf("failed updating counters for the parent group '%s' of host '%s': %s", g, h.Name, err)
			}
		}
		h.GroupChains = groupChains
		h.Groups = groups
	}

	for _, g := range inv.Groups {
		if g.Counters.Hosts < 1 {
			return fmt.Errorf("inventory group '%s' has no hosts", g.Name)
		}
		for _, a := range g.Ancestors {
			if err := inv.AddGroupMemberCounter("group", a); err != nil {
				return fmt.Errorf("failed updating counters for '%s' group: %s", a, err)
			}
		}
	}

	// inherit variables from parent groups
	for _, h := range inv.Hosts {
		m := make(map[string]string)
		for _, g := range h.Groups {
			group, err := inv.GetGroup(g)
			if err != nil {
				return err
			}
			for k, v := range group.Variables {
				m[k] = v
			}
		}
		for k, v := range m {
			if _, exists := h.Variables[k]; !exists {
				h.Variables[k] = v
			}
		}
	}

	return nil
}

// AddGroupMemberCounter increments group membership counters for hosts and
// sub-groups.
func (inv *Inventory) AddGroupMemberCounter(counterType, groupName string) error {
	if _, exists := inv.GroupsRef[groupName]; !exists {
		return fmt.Errorf("group %s does not exist", groupName)
	}
	for _, g := range inv.Groups {
		if g.Name == groupName {
			if counterType == "host" {
				atomic.AddUint64(&g.Counters.Hosts, 1)
				return nil
			}
			atomic.AddUint64(&g.Counters.Groups, 1)
			return nil
		}
	}
	return fmt.Errorf("group %s was not found", groupName)
}

// LoadFromBytes loads inventory data from an array of bytes.
func (inv *Inventory) LoadFromBytes(b []byte) error {
	s := string(b[:])
	return inv.parseString(s)
}

// LoadFromFile loads inventory data from a file.
func (inv *Inventory) LoadFromFile(s string) error {
	b, err := ioutil.ReadFile(s)
	if err != nil {
		return err
	}
	s = string(b[:])
	return inv.parseString(s)
}

// GetHosts returns a list of InventoryHost instances.
func (inv *Inventory) GetHosts() ([]*InventoryHost, error) {
	return inv.Hosts, nil
}

// AddGroup adds a group to the Inventory.
func (inv *Inventory) AddGroup(s, p string) error {
	for _, g := range inv.Groups {
		if g.Name == s {
			for _, a := range g.Ancestors {
				if a == p {
					return nil
				}
			}
			g.Ancestors = append(g.Ancestors, p)
			return nil
		}
	}
	g := &InventoryGroup{
		Name:      s,
		Variables: make(map[string]string),
	}
	g.Ancestors = append(g.Ancestors, p)
	inv.Groups = append(inv.Groups, g)
	inv.GroupsRef[s] = true
	return nil
}

func getKeyValuePairs(s string) (map[string]string, error) {
	s = strings.TrimSpace(s)
	m := make(map[string]string)
	size := len(s)
	var k string

	x := 10000
	for {
		x--
		if x == 0 {
			break
		}
		i := strings.Index(s, "=")
		if i < 0 {
			break
		}
		k = s[:i]
		if i >= size {
			break
		}
		s = s[i+1:]
		nx := strings.Index(s, "=")
		if nx < 0 {
			// no more key value pairs
			m[k] = strings.TrimSpace(s)
			break
		} else {
			v := s[:nx]
			vIndex := strings.LastIndex(v, " ")
			v = s[:vIndex]
			m[k] = strings.TrimSpace(v)
			s = s[vIndex+1:]
		}
	}
	return m, nil
}

// AddHost adds a host to the Inventory.
func (inv *Inventory) AddHost(s, groupName string) error {
	if _, exists := inv.GroupsRef[groupName]; !exists {
		return fmt.Errorf("the group %s for host %s does not exist", groupName, s)
	}
	n := strings.Split(s, " ")[0]
	kv, err := getKeyValuePairs(s[len(n):])
	if err != nil {
		return err
	}
	if g, exists := inv.HostsRef[n]; exists {
		if g != groupName {
			return fmt.Errorf("host %s exist in multiple groups: %s, %s", n, g, groupName)
		}
	}
	h := &InventoryHost{
		Name:      n,
		Parent:    groupName,
		Variables: kv,
	}
	inv.HostsRef[n] = groupName
	inv.Hosts = append(inv.Hosts, h)
	return nil
}

// AddVariable adds a variable to an InventoryGroup.
func (inv *Inventory) AddVariable(s, groupName string) error {
	if _, exists := inv.GroupsRef[groupName]; !exists {
		return fmt.Errorf("the group %s does not exist", groupName)
	}
	kvPairs, err := getKeyValuePairs(s)
	if err != nil {
		return err
	}
	for _, g := range inv.Groups {
		if g.Name == groupName {
			for k, v := range kvPairs {
				g.Variables[k] = v
			}
			break
		}
	}
	return nil
}

// GetParentGroupChains gets parent inventory groups recursively for the provided one.
func (inv *Inventory) GetParentGroupChains(s string) ([]string, []string, error) {
	var x, max int
	outputs := make(map[string]bool)
	groups := make(map[string]bool)
	groups[s] = false
	max = 10000
	x = max
	for {
		x--
		if x == 0 {
			return []string{}, []string{}, fmt.Errorf("failed to get parent groups: exceeded %d (max) iterations", max)
		}
		breakOut := true
		for k, completed := range groups {
			if completed {
				continue
			}
			parentGroups, err := inv.GetParentGroup(k)
			if err != nil {
				return []string{}, []string{}, err
			}
			groups[k] = true
			for _, g := range parentGroups {
				if _, exists := groups[g]; !exists {
					groups[g] = false
					breakOut = false
				}
				if g != "all" {
					out := fmt.Sprintf("%s,%s", g, k)
					if _, exists := outputs[out]; !exists {
						outputs[out] = true
					}
				}
			}
		}
		if breakOut {
			break
		}
	}

	max = 10000
	x = max
	for {
		x--
		if x == 0 {
			return []string{}, []string{}, fmt.Errorf("failed to assemble group chains: exceeded %d (max) iterations", max)
		}
		delElements := []string{}
		continueNow := false
		for g1 := range outputs {
			g1arr := strings.Split(g1, ",")
			for g2 := range outputs {
				if g1 == g2 {
					continue
				}
				g2arr := strings.Split(g2, ",")
				// check whether the first element is last in the other outputs
				if g1arr[0] == g2arr[len(g2arr)-1] {
					var output string
					if g2arr[len(g2arr)-1] == g1arr[1] {
						output = fmt.Sprintf("%s,%s", g2arr[len(g2arr)-1], g1arr[1])
					} else {
						output = fmt.Sprintf("%s,%s", g2, g1arr[1])
					}
					delElements = append(delElements, g2)
					outputs[output] = true
					continueNow = true
					break
				}
			}
			if continueNow {
				break
			}
		}
		if len(delElements) == 0 {
			break
		}
		for _, e := range delElements {
			delete(outputs, e)
		}
	}

	chains := []string{}
	chains = append(chains, "all")
	for g := range outputs {
		// skip the group if the first element is not a top one or that the last
		// element is not a leaf
		groups := strings.Split(g, ",")
		fg, err := inv.GetGroup(groups[0])
		if err != nil {
			return []string{}, []string{}, err
		}
		if len(fg.Ancestors) > 1 {
			continue
		}
		chains = append(chains, g)
	}

	// sort the array such that group chains with the most members appear last.
	rc := []string{}
	max = 1000
	x = max
	for {
		x--
		if x == 0 {
			return []string{}, []string{}, fmt.Errorf("failed to sort group chains: exceeded %d (max) iterations", max)
		}
		k := 0
		v := 10000
		for i, chain := range chains {
			j := len(strings.Split(chain, ","))
			if j < v {
				k = i
				v = j
			}
		}
		rc = append(rc, chains[k])
		chains[k] = chains[len(chains)-1]
		chains[len(chains)-1] = ""
		chains = chains[:len(chains)-1]
		if len(chains) == 0 {
			break
		}
	}

	// create a list of unique groups
	groupChains := make([]string, len(rc))
	copy(groupChains, rc)
	processedGroups := make(map[string]float64)
	max = 10000
	x = max
	for {
		x--
		if x == 0 {
			return []string{}, []string{}, fmt.Errorf("failed create a list of unique groups: exceeded %d (max) iterations", max)
		}
		for i, chain := range groupChains {
			groups := strings.Split(chain, ",")
			if len(groups) < 2 && groups[0] == "" {
				continue
			}
			processedGroups[groups[0]] = float64(x)
			if groupChains[i] == "" {
				continue
			}
			x--
			groupChains[i] = strings.Join(groups[1:], ",")
		}

		isEmpty := true
		for _, chain := range groupChains {
			if chain != "" {
				isEmpty = false
				break
			}
		}
		if isEmpty {
			break
		}
	}

	rg := sortStringFloatMap(processedGroups)

	if len(rg) == 1 {
		if rg[0] == "all" && s != "all" {
			rg = append(rg, s)
			rc = append(rc, s)
		}
	}
	return rc, rg, nil
}

// GetParentGroup gets parent inventory groups for the provided one.
func (inv *Inventory) GetParentGroup(s string) ([]string, error) {
	groups := make(map[string]bool)
	if _, exists := inv.GroupsRef[s]; !exists {
		return []string{}, fmt.Errorf("group %s does not exist in the inventory", s)
	}
	for _, g := range inv.Groups {
		if g.Name == s {
			for _, a := range g.Ancestors {
				if _, exists := groups[a]; !exists {
					groups[a] = false
				}
			}
			break
		}
	}
	r := []string{}
	for g := range groups {
		r = append(r, g)
	}
	return r, nil
}

// GetHost returns an instance of InventoryHost.
func (inv *Inventory) GetHost(s string) (*InventoryHost, error) {
	if _, exists := inv.HostsRef[s]; !exists {
		return nil, fmt.Errorf("host %s does not exist in the inventory", s)
	}
	for _, h := range inv.Hosts {
		if h.Name == s {
			return h, nil
		}
	}
	return nil, fmt.Errorf("host %s not found", s)
}

// GetGroup returns an instance of InventoryGroup.
func (inv *Inventory) GetGroup(s string) (*InventoryGroup, error) {
	if _, exists := inv.GroupsRef[s]; !exists {
		return nil, fmt.Errorf("Group %s does not exist in the inventory", s)
	}
	for _, g := range inv.Groups {
		if g.Name == s {
			return g, nil
		}
	}
	return nil, fmt.Errorf("Group %s not found", s)
}

// GetHostsWithFilter returns a list of InventoryHost instances filtered by
// input host and group patterns. Returns the host matching the patterns only.
func (inv *Inventory) GetHostsWithFilter(hostFilter, groupFilter interface{}) ([]*InventoryHost, error) {
	if hostFilter == nil && groupFilter == nil {
		return inv.Hosts, nil
	}
	hosts := []*InventoryHost{}
	for _, host := range inv.Hosts {
		hostMatched := false
		if hostFilter != nil {
			var filters []string
			// see if a host matches the pattern or patterns
			switch hostFilter.(type) {
			case string:
				filters = append(filters, hostFilter.(string))
			case []string:
				filters = hostFilter.([]string)
			default:
				return hosts, fmt.Errorf("unsupporter host filter type: %T", hostFilter)
			}
			for _, filter := range filters {
				filterPattern, err := regexp.Compile(filter)
				if err != nil {
					return hosts, fmt.Errorf("filter contains invalid pattern: %s, error: %s", filter, err)
				}
				if filterPattern.MatchString(host.Name) {
					hostMatched = true
					break
				}
			}
		}

		if groupFilter != nil {
			var filters []string
			switch groupFilter.(type) {
			case string:
				filters = append(filters, groupFilter.(string))
			case []string:
				filters = groupFilter.([]string)
			default:
				return hosts, fmt.Errorf("unsupporter group filter type: %T", groupFilter)
			}
			for _, filter := range filters {
				if hostMatched {
					break
				}
				filterPattern, err := regexp.Compile(filter)
				if err != nil {
					return hosts, fmt.Errorf("filter contains invalid pattern: %s, error: %s", filter, err)
				}

				for _, group := range host.Groups {
					if filterPattern.MatchString(group) {
						hostMatched = true
						break
					}
				}
			}
		}

		if hostMatched {
			hosts = append(hosts, host)
		}
	}
	return hosts, nil
}
