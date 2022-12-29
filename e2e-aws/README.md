# End to end testing over AWS

This test suite is expected to be run locally, calling a running provisioner deployed directly over AWS infrastructure.

## Requirements

- A provisioner running [docs](../README.md)

## Running

```
go test ./... -v -timeout 3h
```
