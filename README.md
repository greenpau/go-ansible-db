[![Go Report Card](https://goreportcard.com/badge/github.com/greenpau/go-ansible-db)](https://goreportcard.com/report/github.com/greenpau/go-ansible-db) [![Build Status](https://travis-ci.org/greenpau/go-ansible-db.svg?branch=master)](https://travis-ci.org/greenpau/go-ansible-db) [![Hex.pm](https://img.shields.io/hexpm/l/plug.svg)](https://github.com/greenpau/go-ansible-db) [![](https://godoc.org/github.com/greenpau/go-ansible-db?status.svg)](http://godoc.org/github.com/greenpau/go-ansible-db/pkg/db)

# go-ansible-db

Ansible Inventory and Vault management client library written in Go.

## Why?

Ansible inventory and secrets management is being handled well by native
Ansible tools. The inventory format is well defined and the vault usage
is well understood. Ansible is written in Python and therefore integrates
nicely with Python code.

What happens when a user wants to read inventory and secrets for use in
Go applications?

This library allows:
* Reading Ansible ini-style inventory files
* Reading Ansible vault files
* Getting Ansible variables for a host or a group of hosts
* Getting Ansible secrets (credentials) for a host or a group of hosts

## Getting Started

To demonstrate the use of the library, please consider the following files:

* `assets/inventory/hosts`: Ansible inventory file
* `assets/inventory/vault.yml`: Ansible vault file
* `assets/inventory/vault.key`: The file with the password for the vault

The following code snippet would load the inventory and vault content.

```golang
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
```

After that, the code retrieves the inventory record for `ny-sw01` and makes
a subsequent call to retrieve the credentials for accessing `ny-sw01`.

```golang
h := "ny-sw01"
host, err := inv.GetHost(h)
if err != nil {
    t.Fatalf("error getting host %s from inventory: %s", h, err)
}
creds, err := vlt.GetCredentials(host.Name)
if err != nil {
    t.Fatalf("error getting credentials for host %s: %s", host.Name, err)
}
```
