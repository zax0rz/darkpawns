package admin

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/zax0rz/darkpawns/pkg/auth"
)

// VULN-043: the admin token lives in an HttpOnly cookie, not the SPA's
// localStorage. A same-origin script cannot read an HttpOnly cookie, so an
// XSS in the console can no longer exfiltrate a 24-hour admin credential; it
// would have to make its requests through the victim's own session instead,
// which the per-request CSRF header below also demands.

const (
	// adminTokenCookie carries the admin JWT. Same name the SPA used for its
	// localStorage key, so operator tooling keeps one name to remember.
	adminTokenCookie = "admin_token"
	// adminCSRFHeader must accompany every authenticated admin request.
	// Browsers cannot attach a custom header to a cross-site request without
	// a CORS preflight this router never approves, so a page on another
	// origin cannot drive an authenticated admin request even though the
	// cookie is attached automatically. SameSite=Strict already blocks the
	// simple case; this covers configured CORS origins and SameSite gaps.
	adminCSRFHeader = "X-Requested-With"
	// adminTokenMaxAge matches the JWT's 24-hour expiry (pkg/auth/jwt.go) so
	// the cookie dies with the token inside it.
	adminTokenMaxAge = 24 * 60 * 60
)

// adminCookieAttrs is the attribute suffix shared by the login and the
// clear variant. Path is /admin so the cookie never rides on game or site
// requests.
const adminCookieAttrs = "; Path=/admin; Max-Age="

// adminTokenCookieString renders the login Set-Cookie header value.
func adminTokenCookieString(token string) string {
	var b strings.Builder
	b.WriteString(adminTokenCookie)
	b.WriteByte('=')
	b.WriteString(token)
	b.WriteString(adminCookieAttrs)
	b.WriteString(strconv.Itoa(adminTokenMaxAge))
	b.WriteString("; Secure; HttpOnly; SameSite=Strict")
	return b.String()
}

// clearedAdminTokenCookieString renders the logout Set-Cookie header value:
// an empty value with Max-Age=0 expires the cookie in every browser.
func clearedAdminTokenCookieString() string {
	return adminTokenCookie + "=" + adminCookieAttrs + "0; Secure; HttpOnly; SameSite=Strict"
}

// setAdminTokenCookie issues the login cookie on an http.ResponseWriter.
func setAdminTokenCookie(w http.ResponseWriter, token string) {
	w.Header().Add("Set-Cookie", adminTokenCookieString(token))
}

// claimsFromAmbient resolves claims for an authenticated admin request: the
// HttpOnly cookie first (the console's path), then the Authorization Bearer
// header (kept for non-browser tooling such as curl and monitoring scripts).
// The Huma gates only see headers as strings, so this takes a header getter
// instead of an *http.Request.
func claimsFromAmbient(getHeader func(string) string) (*auth.Claims, error) {
	if token := adminCookieValue(getHeader("Cookie")); token != "" {
		if claims, err := auth.ValidateJWT(token); err == nil {
			return claims, nil
		}
		// An invalid cookie falls through to the Bearer check rather than
		// failing immediately, so a stale cookie plus a fresh Bearer token
		// still authenticates.
	}
	return claimsFromBearerHeader(getHeader("Authorization"))
}

// adminCookieValue reads one cookie out of a raw Cookie header. net/http's
// parser needs a request, so this adapts the header string to one.
func adminCookieValue(cookieHeader string) string {
	if cookieHeader == "" {
		return ""
	}
	req := &http.Request{Header: http.Header{}}
	req.Header.Set("Cookie", cookieHeader)
	for _, c := range req.Cookies() {
		if c.Name == adminTokenCookie {
			return c.Value
		}
	}
	return ""
}

// hasAdminCSRFHeader reports whether the request carries the custom header
// required of every authenticated admin request (see adminCSRFHeader).
func hasAdminCSRFHeader(getHeader func(string) string) bool {
	return getHeader(adminCSRFHeader) != ""
}
