// Package apidoc owns the single OpenAPI document for the Dark Pawns HTTP
// API and the Huma v2 APIs that contribute operations to it.
//
// Tranche 1 of the Huma migration mounts one API on the root mux (serving the
// generated spec at /openapi.json) and lets the admin router register its
// migrated operations into the same document through NewInternalAPI. Later
// tranches migrate the remaining admin routes off the plain mux; the document
// grows, but the ownership stays here.
package apidoc

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// Title and Version identify the generated OpenAPI document. They replace the
// hand-maintained web/api/openapi.json, whose title ("Dark Pawns Agent API")
// described the WebSocket agent surface; the generated document covers the
// typed REST operations only.
const (
	Title   = "Dark Pawns API"
	Version = "1.0.0"
)

// Doc is the shared OpenAPI document plus the cached JSON encoding served at
// /openapi.json (by Huma) and /api/openapi.json (by Handler).
type Doc struct {
	oapi *huma.OpenAPI

	jsonOnce sync.Once
	json     []byte
	jsonErr  error
}

// New creates the shared OpenAPI document. Every Huma API built from this Doc
// via NewAPI or NewInternalAPI contributes its operations to the same
// document, so one spec describes the whole migrated surface.
func New() *Doc {
	return &Doc{oapi: newConfig().OpenAPI}
}

// OpenAPI returns the shared document.
func (d *Doc) OpenAPI() *huma.OpenAPI { return d.oapi }

// JSON returns the marshaled document, computed once. The marshal error, if
// any, is recorded and returned alongside the bytes so callers can answer 500
// instead of serving an empty body as success.
func (d *Doc) JSON() ([]byte, error) {
	d.jsonOnce.Do(func() {
		d.json, d.jsonErr = json.Marshal(d.oapi)
	})
	return d.json, d.jsonErr
}

// Handler serves the generated document for the /api/openapi.json
// compatibility alias. The canonical /openapi.json is served by Huma itself.
// GET and HEAD are answered (the http.ServeFile this replaces answered both);
// anything else is a 405.
func (d *Doc) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/openapi+json")
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		body, err := d.JSON()
		if err != nil {
			http.Error(w, "failed to encode OpenAPI document", http.StatusInternalServerError)
			return
		}
		if _, err := w.Write(body); err != nil {
			http.Error(w, "write failed", http.StatusInternalServerError)
		}
	}
}

// NewAPI returns a Huma API on mux that serves the shared document at
// /openapi.json (+ .yaml and 3.0 downgrade variants, Huma's default spec
// routes). Use it for the root mux. The mux is tracked by the caller if its
// pattern list matters (the root drift gate).
func (d *Doc) NewAPI(mux humago.Mux) huma.API {
	cfg := newConfig()
	cfg.OpenAPI = d.oapi
	return humago.New(mux, cfg)
}

// NewInternalAPI returns a Huma API on mux that contributes operations to the
// shared document without serving spec, docs, or schema routes of its own.
// Use it for routers mounted behind middleware on another mux (the admin
// console's migrated operations).
func (d *Doc) NewInternalAPI(mux humago.Mux) huma.API {
	cfg := newConfig()
	cfg.OpenAPI = d.oapi
	cfg.OpenAPIPath = ""
	return humago.New(mux, cfg)
}

// newConfig builds the base Huma configuration. The schema-link transformer
// is disabled: it injects a `$schema` property into every response body and a
// Link header, which would change player-visible bytes the port already emits
// (R1 — the migrated handlers must keep returning the bodies they returned on
// the plain mux).
//
// The JSON marshal is also pinned to the standard library's default encoder
// settings: Huma's DefaultJSONFormat disables HTML escaping, but the
// hand-rolled handlers this migration replaces wrote through
// json.NewEncoder's defaults, which escape `<`, `>` and `&` as <, > and
// &. Keeping the default escaping means every migrated success body is
// byte-identical to the plain-mux handler it replaces, for every possible
// string, not just the ones the fixtures happen to contain. Both encoders
// append the trailing newline json.Encoder has always emitted.
func newConfig() huma.Config {
	cfg := huma.DefaultConfig(Title, Version)
	cfg.CreateHooks = nil
	cfg.Transformers = nil
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	cfg.Formats["application/json"] = huma.Format{
		Marshal:   func(w io.Writer, v any) error { return json.NewEncoder(w).Encode(v) },
		Unmarshal: json.Unmarshal,
	}
	cfg.Formats["text/plain"] = huma.Format{
		Marshal: func(w io.Writer, v any) error {
			pe, ok := v.(*PlainError)
			if !ok {
				return fmt.Errorf("text/plain format cannot marshal %T", v)
			}
			_, err := io.WriteString(w, pe.Body+"\n")
			return err
		},
		Unmarshal: func(_ []byte, _ any) error { return fmt.Errorf("text/plain requests are not supported") },
	}
	return cfg
}

// PlainError is an operation error that answers with the exact wire shape
// http.Error produced for the pre-migration handlers: the given status, a
// text/plain content type, the X-Content-Type-Options: nosniff header, and
// the body written verbatim followed by a newline. Huma's default error
// writer would instead emit an application/problem+json body (title/status/
// detail); returning a PlainError from an operation handler keeps the
// error-path bytes identical to the plain mux the operation migrated off.
type PlainError struct {
	Status int
	Body   string
}

// NewPlainError returns a PlainError for status with body written verbatim.
func NewPlainError(status int, body string) *PlainError {
	return &PlainError{Status: status, Body: body}
}

// Error implements the error interface; it returns the response body.
func (e *PlainError) Error() string { return e.Body }

// GetStatus implements huma.StatusError.
func (e *PlainError) GetStatus() int { return e.Status }

// GetHeaders implements huma.HeadersError: http.Error sets nosniff on every
// error it writes, and the bytes a caller sees include that header.
func (e *PlainError) GetHeaders() http.Header {
	return http.Header{"X-Content-Type-Options": {"nosniff"}}
}

// ContentType implements huma.ContentTypeFilter: answer text/plain for any
// negotiated content type, exactly as http.Error did regardless of Accept.
func (e *PlainError) ContentType(string) string { return "text/plain; charset=utf-8" }

// HealthOutput is the body of the /health liveness probe. A []byte Body is
// written verbatim, preserving the exact "OK\n" response the endpoint has
// always returned (load balancers and deploy scripts match on it).
type HealthOutput struct {
	Body []byte
}

// RegisterHealth registers GET /health on api. The endpoint is deliberately
// unauthenticated and byte-for-byte identical to the hand-rolled handler it
// replaces: 200 with body "OK\n".
func RegisterHealth(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "get-health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Liveness probe",
		Description: "Returns 200 OK while the server process is serving. Unauthenticated.",
	}, func(ctx context.Context, in *struct{}) (*HealthOutput, error) {
		return &HealthOutput{Body: []byte("OK\n")}, nil
	})
}
