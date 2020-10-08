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
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"golang.org/x/crypto/pbkdf2"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	vaultOperations                 = 10000
	vaultKeyLength                  = 32
	vaultInitializationVectorLength = 16
	vaultSaltLength                 = 32
)

// Vault is the contents of Ansible vault file.
type Vault struct {
	Header      VaultHeader        `xml:"-" json:"-" yaml:"-"`
	Body        VaultBody          `xml:"-" json:"-" yaml:"-"`
	Key         VaultKey           `xml:"-" json:"-" yaml:"-"`
	Password    []byte             `xml:"-" json:"-" yaml:"-"`
	Payload     []byte             `xml:"-" json:"-" yaml:"-"`
	Credentials []*VaultCredential `xml:"credentials" json:"credentials" yaml:"credentials"`
}

// VaultHeader is the header of a Vault.
type VaultHeader struct {
	Format  string `xml:"-" json:"-" yaml:"-"`
	Version string `xml:"-" json:"-" yaml:"-"`
	Cipher  string `xml:"-" json:"-" yaml:"-"`
}

// VaultBody is the body of a Vault.
type VaultBody struct {
	Salt []byte `xml:"-" json:"-" yaml:"-"`
	HMAC []byte `xml:"-" json:"-" yaml:"-"`
	Data []byte `xml:"-" json:"-" yaml:"-"`
}

// VaultKey is the key for a Vault
type VaultKey struct {
	Cipher               []byte `xml:"-" json:"-" yaml:"-"`
	HMAC                 []byte `xml:"-" json:"-" yaml:"-"`
	InitializationVector []byte `xml:"-" json:"-" yaml:"-"`
}

// VaultCredential is a decoded credential from a Vault.
type VaultCredential struct {
	Description     string `xml:"description,omitempty" json:"description,omitempty" yaml:"description,omitempty"`
	Regex           string `xml:"regex,omitempty" json:"regex,omitempty" yaml:"regex,omitempty"`
	Username        string `xml:"username,omitempty" json:"username,omitempty" yaml:"username,omitempty"`
	Password        string `xml:"password,omitempty" json:"password,omitempty" yaml:"password,omitempty"`
	EnabledPassword string `xml:"password_enable,omitempty" json:"password_enable,omitempty" yaml:"password_enable,omitempty"`
	Priority        int    `xml:"priority,omitempty" json:"priority,omitempty" yaml:"priority,omitempty"`
	Default         bool   `xml:"default,omitempty" json:"default,omitempty" yaml:"default,omitempty"`
}

// NewVault returns a pointer to Vault.
func NewVault() *Vault {
	v := &Vault{}
	return v
}

func (v *Vault) readVault(b []byte) error {
	if v.Password == nil {
		return fmt.Errorf("vault password not found")
	}
	lines := strings.Split(string(b[:]), "\n")
	if len(lines) < 2 {
		return fmt.Errorf("invalid vault payload")
	}
	header := strings.Split(strings.TrimSpace(lines[0]), ";")
	if len(header) != 3 {
		return fmt.Errorf("invalid vault header: %s", lines[0])
	}
	// Capture vault header
	v.Header.Format = header[0]
	v.Header.Version = header[1]
	v.Header.Cipher = header[2]
	if v.Header.Version != "1.1" {
		return fmt.Errorf("unsupported vault version: %s", v.Header.Version)
	}

	if v.Header.Cipher != "AES256" {
		return fmt.Errorf("unsupported vault cipher: %s", v.Header.Cipher)
	}
	// Capture vault body
	var bb strings.Builder
	for _, line := range lines[1:] {
		bb.WriteString(strings.TrimSpace(line))
	}
	body, err := hex.DecodeString(bb.String())
	if err != nil {
		return fmt.Errorf("vault hex decoding error: %s", err)
	}
	// Split the body into 3 parts: Salt, HMAC, and Data
	parts := strings.SplitN(string(body[:]), "\n", 3)
	if len(parts) != 3 {
		return fmt.Errorf("invalid vault body")
	}
	saltPart, err := hex.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("invalid vault body (salt): %s", err)
	}
	v.Body.Salt = saltPart
	hmacPart, err := hex.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("invalid vault body (hmac): %s", err)
	}
	v.Body.HMAC = hmacPart
	dataPart, err := hex.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("invalid vault body (data): %s", err)
	}
	v.Body.Data = dataPart
	// Generate a decryption key
	key := pbkdf2.Key(v.Password, v.Body.Salt, vaultOperations, 2*vaultKeyLength*vaultInitializationVectorLength, sha256.New)
	v.Key.Cipher = key[:vaultKeyLength]
	v.Key.HMAC = key[vaultKeyLength:(vaultKeyLength * 2)]
	v.Key.InitializationVector = key[(vaultKeyLength * 2) : (vaultKeyLength*2)+vaultInitializationVectorLength]
	// Valudate the password
	keyHash := hmac.New(sha256.New, v.Key.HMAC)
	keyHash.Write(v.Body.Data)
	if !hmac.Equal(keyHash.Sum(nil), v.Body.HMAC) {
		return fmt.Errorf("invalid vault vault password")
	}
	// Decrypt the vault
	cphr, err := aes.NewCipher(v.Key.Cipher)
	if err != nil {
		return fmt.Errorf("error opening the vault: %s", err)
	}
	plainText := make([]byte, len(v.Body.Data))
	encrBlock := cipher.NewCTR(cphr, v.Key.InitializationVector)
	encrBlock.XORKeyStream(plainText, v.Body.Data)
	output, err := unpadBytes(plainText)
	if err != nil {
		return fmt.Errorf("error opening the vault: %s", err)
	}
	v.Payload = output
	tv := &Vault{}
	if err := yaml.Unmarshal(output, tv); err != nil {
		return fmt.Errorf("error parsing YAML content of the vault: %s", err)
	}
	// Check regular expressions for their validity
	for _, c := range tv.Credentials {
		if !c.Default && c.Regex == "" {
			return fmt.Errorf("invalid vault entry, non-default and empty regex pattern")
		}
		if c.Default && c.Regex != "" {
			return fmt.Errorf("invalid vault entry, default and non-empty regex pattern")
		}
		if c.Default {
			continue
		}
		if _, err := regexp.Compile(c.Regex); err != nil {
			return fmt.Errorf("invalid vault entry, regex compilation for '%s', failed: %s", c.Regex, err)
		}
	}
	v.Credentials = tv.Credentials
	return nil
}

func unpadBytes(b []byte) ([]byte, error) {
	length := len(b)
	paddingLength := int(b[length-1])
	if paddingLength > length {
		return nil, fmt.Errorf("invalid padding")
	}
	return b[:(length - paddingLength)], nil
}

// LoadFromBytes loads vault data from an array of bytes.
func (v *Vault) LoadFromBytes(b []byte) error {
	return v.readVault(b)
}

// LoadFromFile loads vault data from a file.
func (v *Vault) LoadFromFile(fp string) error {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}
	return v.readVault(b)
}

// LoadPasswordFromFile loads unlock password for the vault from a file.
func (v *Vault) LoadPasswordFromFile(fp string) error {
	b, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}
	v.Password = []byte(strings.TrimSpace(strings.Split(string(b[:]), "\n")[0]))
	return nil
}

// SetPassword sets unlock password for the vault.
func (v *Vault) SetPassword(s string) error {
	if s == "" {
		return fmt.Errorf("empty password is unsupported")
	}
	v.Password = []byte(strings.TrimSpace(s))
	return nil
}

// GetCredentials returns a list of credential applicable to the provided
// host name.
func (v *Vault) GetCredentials(s string) ([]*VaultCredential, error) {
	cv := []*VaultCredential{}
	for _, c := range v.Credentials {
		if c.Default {
			continue
		}
		r, err := regexp.Compile(c.Regex)
		if err != nil {
			continue
		}
		if r.MatchString(s) == true {
			cv = append(cv, c)
		}
	}
	sort.SliceStable(cv, func(i, j int) bool {
		return cv[i].Priority < cv[j].Priority
	})
	dcv := []*VaultCredential{}
	for _, c := range v.Credentials {
		if !c.Default {
			continue
		}
		dcv = append(dcv, c)
	}
	sort.SliceStable(dcv, func(i, j int) bool {
		return dcv[i].Priority < dcv[j].Priority
	})
	for _, c := range dcv {
		cv = append(cv, c)
	}
	return cv, nil
}

func (c *VaultCredential) String() string {
	var s strings.Builder
	s.WriteString("username=" + c.Username)
	s.WriteString(", password=" + c.Password)
	s.WriteString(", enabled_password=" + c.EnabledPassword)
	s.WriteString(", priority=" + strconv.Itoa(c.Priority))
	s.WriteString(", default=" + strconv.FormatBool(c.Default))
	s.WriteString(", description=" + c.Description)
	return s.String()
}
