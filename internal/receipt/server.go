package receipt

import (
	"encoding/base64"
	"log/slog"
	"net/http"
	"strings"
)

// Server handles HTTP requests for receipts
type Server struct {
	service   *Service
	basicAuth BasicAuth
	mux       *http.ServeMux
}

// BasicAuth holds basic authentication credentials
type BasicAuth struct {
	Username string
	Password string
}

// NewServer creates a new Server with default mux
func NewServer(service *Service, basicAuth BasicAuth) *Server {
	return NewServerWithMux(service, basicAuth, http.NewServeMux())
}

// NewServerWithMux creates a new Server with a custom mux for testing
func NewServerWithMux(service *Service, basicAuth BasicAuth, mux *http.ServeMux) *Server {
	s := &Server{
		service:   service,
		basicAuth: basicAuth,
		mux:       mux,
	}
	s.registerRoutes()
	return s
}

// authenticate checks basic auth credentials
func (s *Server) authenticate(r *http.Request) bool {
	if s.basicAuth.Username == "" && s.basicAuth.Password == "" {
		return true // No auth required if not configured
	}

	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "Basic ") {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
	if err != nil {
		return false
	}

	credentials := strings.SplitN(string(decoded), ":", 2)
	if len(credentials) != 2 {
		return false
	}

	return credentials[0] == s.basicAuth.Username && credentials[1] == s.basicAuth.Password
}

// corsMiddleware adds CORS headers to responses
func (s *Server) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		s.setCORSHeaders(w)

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next(w, r)
	}
}

// requireAuth middleware
func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !s.authenticate(r) {
			// Ensure CORS headers are set before error response
			s.setCORSHeaders(w)
			w.Header().Set("WWW-Authenticate", `Basic realm="HSA Tracker"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

// setCORSHeaders sets CORS headers on a response
func (s *Server) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "3600")
}

// handleControllers serves controller JavaScript files with correct MIME type
func (s *Server) handleControllers(w http.ResponseWriter, r *http.Request) {
	// Set CORS headers for JavaScript modules
	s.setCORSHeaders(w)

	fs := http.FS(getControllersFS())
	fileServer := http.FileServer(fs)

	// Set correct MIME type for JavaScript modules
	if strings.HasSuffix(r.URL.Path, ".js") {
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	}
	// Strip the /static/controllers/ prefix to get just the filename
	r.URL.Path = strings.TrimPrefix(r.URL.Path, "/static/controllers/")
	if r.URL.Path == "" {
		r.URL.Path = "/"
	}
	fileServer.ServeHTTP(w, r)
}

// registerRoutes registers all API routes on the server's mux
// Routes must be registered from most specific to least specific to avoid conflicts
func (s *Server) registerRoutes() {
	// Static files (CSS, JS, controllers) - register prefix routes first
	s.mux.HandleFunc("GET /static/controllers/", s.requireAuth(s.handleControllers))
	s.mux.HandleFunc("GET /static/app.css", s.requireAuth(s.handleStaticCSS))
	s.mux.HandleFunc("GET /static/app.js", s.requireAuth(s.handleStaticJS))

	// API endpoints - receipts (most specific paths first)
	s.mux.HandleFunc("GET /api/receipts/{id}/file", s.requireAuth(s.handleGetReceiptFile))
	s.mux.HandleFunc("GET /api/receipts/{id}", s.requireAuth(s.handleGetReceipt))
	s.mux.HandleFunc("DELETE /api/receipts/{id}", s.requireAuth(s.handleDeleteReceipt))
	s.mux.HandleFunc("GET /api/receipts", s.requireAuth(s.handleListReceipts))
	s.mux.HandleFunc("POST /api/receipts", s.requireAuth(s.handleUploadReceipt))

	// API endpoints - reimbursements
	s.mux.HandleFunc("GET /api/reimbursements/{id}", s.requireAuth(s.handleGetReimbursement))
	s.mux.HandleFunc("GET /api/reimbursements", s.requireAuth(s.handleListReimbursements))
	s.mux.HandleFunc("POST /api/reimbursements", s.requireAuth(s.handleCreateReimbursement))

	// Static HTML interface (register last as it's the catch-all)
	s.mux.HandleFunc("GET /index.html", s.requireAuth(s.handleIndex))
	s.mux.HandleFunc("GET /", s.requireAuth(s.handleIndex))
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	slog.Info("Starting server", "address", addr)
	// Wrap the mux with CORS middleware to handle all requests including OPTIONS
	return http.ListenAndServe(addr, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
			s.mux.ServeHTTP(w, r)
		})(w, r)
	}))
}

// ServeHTTP implements http.Handler for testing
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}
