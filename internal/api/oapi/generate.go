// Package oapi holds the code generated from api/openapi.yaml: request and
// response types, the ServerInterface the HTTP layer implements and the
// embedded spec used to validate requests. Do not edit api.gen.go by hand.
package oapi

//go:generate go tool oapi-codegen -config config.yaml ../../../api/openapi.yaml
