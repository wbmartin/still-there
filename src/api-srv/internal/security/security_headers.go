package security

import (
	"net/http"
	"strconv"
)

// HeadersMiddleware adds essential security headers to all responses
func HeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy. The API answers JSON only and serves no
		// page, script or asset (nginx serves the web bundle), so nothing
		// may load, frame, or submit from a response. A file route shown
		// inline replaces this and X-Frame-Options in its handler
		// (doc/105, Serving user files).
		w.Header().Set("Content-Security-Policy",
			"default-src 'none'; "+
				"frame-ancestors 'none'; "+
				"base-uri 'none'; "+
				"form-action 'none'")

		// HTTP Strict Transport Security - force HTTPS
		w.Header().Set("Strict-Transport-Security",
			"max-age="+strconv.Itoa(365*24*3600)+"; "+
				"includeSubDomains; "+
				"preload")

		// X-Frame-Options - prevent clickjacking
		w.Header().Set("X-Frame-Options", "DENY")

		// X-Content-Type-Options - prevent MIME type sniffing
		w.Header().Set("X-Content-Type-Options", "nosniff")

		// Referrer-Policy - control referrer information
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

		// Permissions-Policy - restrict browser features
		w.Header().Set("Permissions-Policy",
			"camera=(), "+
				"microphone=(), "+
				"geolocation=(), "+
				"gyroscope=(), "+
				"magnetometer=(), "+
				"payment=(), "+
				"usb=()")

		// X-Permitted-Cross-Domain-Policies - restrict Flash/cross-domain requests
		w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")

		// Remove server header for security
		w.Header().Del("Server")

		// Cache control for API responses (no caching by default)
		if r.Method == "GET" && r.URL.Path != "/health" {
			w.Header().
				Set("Cache-Control", "no-cache, no-store, must-revalidate")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
		}

		// Call the next handler
		next.ServeHTTP(w, r)
	})
}

// CORSMiddleware handles CORS for API clients (if needed)
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			allowed := false
			for _, allowedOrigin := range allowedOrigins {
				if origin == allowedOrigin {
					allowed = true
					break
				}
			}

			if allowed {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set(
					"Access-Control-Allow-Methods",
					"GET, POST, PUT, DELETE, OPTIONS",
				)
				w.Header().Set(
					"Access-Control-Allow-Headers",
					"Content-Type, X-CSRF-Token, Authorization",
				)
				w.Header().Set("Access-Control-Max-Age", "86400") // 24 hours
				w.Header().
					Set("Vary", "Origin")
				// For proper caching
			}

			// Handle preflight requests
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
