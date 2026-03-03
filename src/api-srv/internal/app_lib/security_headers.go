package app_lib

import (
	"net/http"
	"strconv"
)

// SecurityHeadersMiddleware adds essential security headers to all responses
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Content Security Policy - restrict resource loading
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' 'unsafe-inline'; "+ // Allow inline styles for simplicity
				"img-src 'self' data: https:; "+ // Allow data URLs and HTTPS images
				"font-src 'self'; "+
				"connect-src 'self'; "+
				"media-src 'none'; "+
				"object-src 'none'; "+
				"frame-src 'none'; "+
				"base-uri 'self'; "+
				"form-action 'self';")

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
		//TODO determine why this would even be here
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
