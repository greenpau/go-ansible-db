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
	//"fmt"
	//"io/ioutil"
	"testing"
)

func TestNewInventory(t *testing.T) {
	testFailed := 0
	for i, test := range []struct {
		input      []byte
		inputFile  string
		host       string
		size       uint64
		shouldFail bool
		shouldErr  bool
	}{
		{
			input:      []byte(`ny-sw01 os=cisco_nxos`),
			host:       "ny-sw01",
			size:       1,
			shouldFail: false,
			shouldErr:  false,
		},
		{
			input:      []byte(`ny-sw02 os=cisco_nxos`),
			host:       "ny-sw03",
			size:       1,
			shouldFail: true,
			shouldErr:  false,
		},
		{
			inputFile:  "../../assets/inventory/hosts",
			host:       "ny-sw05",
			size:       1,
			shouldFail: true,
			shouldErr:  false,
		},
		{
			inputFile:  "../../assets/inventory/hosts",
			host:       "ny-sw01",
			size:       5,
			shouldFail: false,
			shouldErr:  false,
		},
		{
			inputFile:  "../../assets/inventory/hosts5",
			host:       "ny-sw10",
			size:       1,
			shouldFail: false,
			shouldErr:  false,
		},
		{
			inputFile:  "../../assets/inventory/hosts6",
			host:       "p1s10.dcauxbozotron.com",
			size:       1,
			shouldFail: false,
			shouldErr:  false,
		},
	} {
		inv := NewInventory()
		var err error
		if test.inputFile == "" {
			err = inv.LoadFromBytes(test.input)
		} else {
			err = inv.LoadFromFile(test.inputFile)
		}
		if err != nil {
			if !test.shouldErr {
				t.Logf("FAIL: Test %d: expected to pass, but threw error: %v", i, err)
				testFailed++
				continue
			}
		} else {
			if test.shouldErr {
				t.Logf("FAIL: Test %d: expected to throw error, but passed", i)
				testFailed++
				continue
			}
		}

		if (inv.Size() != test.size) && !test.shouldFail {
			t.Logf("FAIL: Test %d: expected to pass, but failed due to inventory size: '%d' (actual) vs. '%d' (expected)",
				i, inv.Size(), test.size)
			testFailed++
			continue
		}

		if host, err := inv.GetHost(test.host); err != nil {
			if !test.shouldFail {
				t.Logf("FAIL: Test %d: host %s: expected to pass, but failed due to: %s", i, test.host, err)
				testFailed++
				continue
			}
		} else {
			if test.shouldFail {
				t.Logf("FAIL: Test %d: host %s: expected to fail, but passed", i, test.host)
				testFailed++
				continue
			}
			t.Logf("INFO: Test %d: host %s\n%s", i, test.host, host)
		}

		if test.shouldFail {
			t.Logf("PASS: Test %d: host '%s', expected to fail, failed", i, test.host)
		} else {
			t.Logf("PASS: Test %d: host '%s', expected to pass, passed", i, test.host)
		}
	}
	if testFailed > 0 {
		t.Fatalf("Failed %d tests", testFailed)
	}
}

func TestGetHost(t *testing.T) {
	invFile := "../../assets/inventory/hosts"
	vltFile := "../../assets/inventory/vault.yml"
	vltKeyFile := "../../assets/inventory/vault.key"
	// Create a new inventory file.
	inv := NewInventory()
	// Load the contents of the inventory from an input file.
	if err := inv.LoadFromFile(invFile); err != nil {
		t.Fatalf("error reading inventory: %s", err)
	}
	// Create a new vault file.
	vlt := NewVault()
	// Read the password for the vault file from an input file.
	if err := vlt.LoadPasswordFromFile(vltKeyFile); err != nil {
		t.Fatalf("error reading vault key file: %s", err)
	}
	// Load the contents of the vault from an input file.
	if err := vlt.LoadFromFile(vltFile); err != nil {
		t.Fatalf("error reading vault: %s", err)
	}

	for i, test := range []struct {
		host        string
		vars        int
		groups      int
		groupChains int
		credentials int
	}{
		{
			host:        "ny-sw01",
			vars:        7,
			groups:      6,
			groupChains: 3,
			credentials: 4,
		},
		{
			host:        "ny-sw02",
			vars:        7,
			groups:      6,
			groupChains: 3,
			credentials: 4,
		},
		{
			host:        "ny-sw03",
			vars:        7,
			groups:      6,
			groupChains: 3,
			credentials: 4,
		},
		{
			host:        "ny-sw04",
			vars:        7,
			groups:      6,
			groupChains: 3,
			credentials: 4,
		},
		{
			host:        "controller",
			vars:        2,
			groups:      1,
			groupChains: 1,
			credentials: 2,
		},
	} {
		// Get host variables for a specific host.
		host, err := inv.GetHost(test.host)
		if err != nil {
			t.Fatalf("error getting host %s from inventory: %s", test.host, err)
		}
		if len(host.Variables) != test.vars {
			t.Fatalf("the number of variables for host %s is not %d, but %d", host.Name, test.vars, len(host.Variables))
		}
		// Validate the number of group memberships for a specific host
		if len(host.Groups) != test.groups {
			t.Fatalf("the number of groups for host %s is not %d, but %d", host.Name, test.groups, len(host.Groups))
		}
		// Validate the number of group chains associated with a specific host
		if len(host.GroupChains) != test.groupChains {
			t.Fatalf("the number of group chains for host %s is not %d, but %d", host.Name, test.groupChains, len(host.GroupChains))
		}
		// Get credentials for a specific host.
		creds, err := vlt.GetCredentials(host.Name)
		if err != nil {
			t.Fatalf("error getting credentials for host %s: %s", host.Name, err)
		}
		if len(creds) != test.credentials {
			t.Fatalf("the number of credentials for host %s is not %d, but %d", host.Name, test.credentials, len(creds))
		}
		// Display host summary
		t.Logf("PASS: Test %d, Host '%s' found, parent group: %s", i, host.Name, host.Parent)
		t.Logf("Credentials:")
		for _, c := range creds {
			t.Logf("  - %s", c)
		}
		t.Logf("Variables:")
		for k, v := range host.Variables {
			t.Logf("  - %s: %s", k, v)
		}
		t.Logf("Groups:")
		for _, g := range host.Groups {
			t.Logf("  - %s", g)
		}
		t.Logf("Group Chains:")
		for _, g := range host.GroupChains {
			t.Logf("  - %s", g)
		}
	}
}
