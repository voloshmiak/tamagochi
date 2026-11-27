package httpx

import (
	"context"
	"net/http"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	nethttpmiddleware "github.com/oapi-codegen/nethttp-middleware"
)

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

func validationMessage(err error) string {
	msg, _, _ := strings.Cut(err.Error(), "\n")
	return msg
}

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
