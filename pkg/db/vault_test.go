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

func TestNewVault(t *testing.T) {
	testFailed := 0
	for i, test := range []struct {
		host       string
		inputFile  string
		key        string
		size       int
		keyFile    string
		shouldFail bool
		shouldErr  bool
	}{
		{
			host:       "ny-sw01",
			inputFile:  "../../assets/inventory/vault.yml",
			keyFile:    "../../assets/inventory/vault.key",
			size:       4,
			shouldFail: false,
			shouldErr:  false,
		},
		{
			host:       "ny-sw01",
			inputFile:  "../../assets/inventory/vault.yml",
			key:        "7f017fde-e88b-42c5-89df-a7c8f9de981d",
			size:       4,
			shouldFail: false,
			shouldErr:  false,
		},
	} {
		vlt := NewVault()
		var err error
		if test.key == "" {
			err = vlt.LoadPasswordFromFile(test.keyFile)
		} else {
			err = vlt.SetPassword(test.key)
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

		err = vlt.LoadFromFile(test.inputFile)
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

		hostCredentials, err := vlt.GetCredentials(test.host)
		if err != nil {
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
			for _, c := range hostCredentials {
				t.Logf("INFO: Test %d: host %s, credential: %v", i, test.host, c)
			}
		}

		if len(hostCredentials) != test.size {
			if !test.shouldFail {
				t.Logf("FAIL: Test %d: host %s: expected to pass, but failed due to len(credentials) mismatch %d (actual) vs. %d (expected)",
					i, test.host, len(hostCredentials), test.size)
				testFailed++
				continue
			}
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
