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
}

// New creates the shared OpenAPI document. Every Huma API built from this Doc
// via NewAPI or NewInternalAPI contributes its operations to the same
// document, so one spec describes the whole migrated surface.
func New() *Doc {
	return &Doc{oapi: newConfig().OpenAPI}
}

// OpenAPI returns the shared document.
func (d *Doc) OpenAPI() *huma.OpenAPI { return d.oapi }

// JSON returns the marshaled document, computed once.
func (d *Doc) JSON() []byte {
	d.jsonOnce.Do(func() {
		d.json, _ = json.Marshal(d.oapi)
	})
	return d.json
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
		if _, err := w.Write(d.JSON()); err != nil {
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
func newConfig() huma.Config {
	cfg := huma.DefaultConfig(Title, Version)
	cfg.CreateHooks = nil
	cfg.Transformers = nil
	cfg.DocsPath = ""
	cfg.SchemasPath = ""
	return cfg
}

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
