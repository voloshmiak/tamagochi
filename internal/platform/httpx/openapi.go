package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

// ValidateRequests rejects requests that do not match the OpenAPI spec
// (unknown fields, wrong types, values out of range) with 400 in the usual
// error format, before they reach a handler. basePath is the prefix the API
// is served under, e.g. "/api/v1". Authentication is not checked here; see
// Authenticate.
func ValidateRequests(spec *openapi3.T, basePath string) func(http.Handler) http.Handler {
	return nethttpmiddleware.OapiRequestValidatorWithOptions(spec, &nethttpmiddleware.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: openapi3filter.NoopAuthenticationFunc,
		},
		DoNotValidateServers: true,
		Prefix:               basePath,
		ErrorHandlerWithOpts: func(_ context.Context, err error, w http.ResponseWriter, _ *http.Request, opts nethttpmiddleware.ErrorHandlerOpts) {
			code := "invalid"
			if opts.StatusCode == http.StatusNotFound {
				code = "not_found"
			}
			JSON(w, opts.StatusCode, ErrorBody{Error: code, Message: validationMessage(err)})
		},
	})
}

// validationMessage keeps the first line of a kin-openapi error: the rest is
// a dump of the schema, which is noise for an API client.
func validationMessage(err error) string {
	msg, _, _ := strings.Cut(err.Error(), "\n")
	return msg
}

// PublicOperations returns the "METHOD /path" route patterns (with basePath
// prepended) of the operations that the spec marks as not requiring
// authentication, i.e. with an explicit empty `security: []`.
func PublicOperations(spec *openapi3.T, basePath string) map[string]bool {
	public := make(map[string]bool)
	globalOpen := len(spec.Security) == 0
	for path, item := range spec.Paths.Map() {
		for method, op := range item.Operations() {
			open := globalOpen
			if op.Security != nil {
				open = len(*op.Security) == 0
			}
			if open {
				public[method+" "+basePath+path] = true
			}
		}
	}
	return public
}
