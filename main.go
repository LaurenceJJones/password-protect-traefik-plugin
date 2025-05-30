package password_protect_traefik_plugin

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
)

// Config holds the plugin configuration
type Config struct {
	Password      string `json:"password,omitempty"`
	LoginHTMLPath string `json:"loginHtmlPath,omitempty"`
}

// LoginPage holds data for the login page template
type LoginPage struct {
	Title   string
	Message string
}

const (
	cookieName           = "spp-session"
	defaultLoginHTMLPath = "/login.html"
)

// CreateConfig creates and initializes the plugin configuration
func CreateConfig() *Config {
	return &Config{}
}

// PasswordProtect contains the plugin configuration and implementation
type PasswordProtect struct {
	config *Config
	next   http.Handler
	tmpl   *template.Template
	valid  bool
}

// New creates a new PasswordProtect plugin
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	if config.LoginHTMLPath == "" {
		config.LoginHTMLPath = defaultLoginHTMLPath
	}

	if _, err := os.Stat(config.LoginHTMLPath); os.IsNotExist(err) {
		return &PasswordProtect{
			config: config,
			next:   next,
			valid:  false,
		}, nil
	}

	LoginHTMLBytes, err := os.ReadFile(config.LoginHTMLPath)
	if err != nil {
		return nil, fmt.Errorf("error reading login HTML file from %s: %v", config.LoginHTMLPath, err)
	}
	// Parse the template
	tmpl, err := template.New("login").Parse(string(LoginHTMLBytes))
	if err != nil {
		return nil, fmt.Errorf("error parsing login template: %v", err)
	}

	return &PasswordProtect{
		config: config,
		next:   next,
		tmpl:   tmpl,
		valid:  true,
	}, nil
}

// ServeHTTP implements http.Handler
func (p *PasswordProtect) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.valid {
		os.Stderr.WriteString("[PasswordProtect] Login HTML file does not exist at " + p.config.LoginHTMLPath + "\n")
		os.Stderr.WriteString("[PasswordProtect] Please check the path and try again as route is unprotected.\n")
		p.next.ServeHTTP(w, r)
		return
	}
	// Check if the request has a valid session cookie
	if cookie, err := r.Cookie(cookieName); err == nil {
		if p.isValidCookie(cookie.Value) {
			// Pass through to the next handler if cookie is valid
			p.next.ServeHTTP(w, r)
			return
		}
	}

	// Handle form submission with password
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err == nil {
			password := r.Form.Get("spp-password")
			if password == p.config.Password {
				// Password is correct, create a session and redirect back to the same page
				sessionID := uuid.New().String()
				signedCookie := p.signCookie(sessionID)

				cookie := &http.Cookie{
					Name:     cookieName,
					Value:    signedCookie,
					HttpOnly: true,
					Secure:   r.TLS != nil,
				}
				http.SetCookie(w, cookie)

				// Redirect to the original URL to avoid form resubmission
				http.Redirect(w, r, r.URL.String(), http.StatusSeeOther)
				return
			}

			// Incorrect password, show login page with error
			p.showLoginPage(w, "Invalid password. Please try again.")
			return
		}
	}

	// Show login page for GET requests or if POST handling failed
	p.showLoginPage(w, "")
}

// showLoginPage renders the login page
func (p *PasswordProtect) showLoginPage(w http.ResponseWriter, message string) {
	data := LoginPage{
		Title:   "Password Protected",
		Message: message,
	}

	w.Header().Set("Content-Type", "text/html")
	w.WriteHeader(http.StatusUnauthorized)
	p.tmpl.Execute(w, data)
}

// signCookie creates a signed cookie value
func (p *PasswordProtect) signCookie(sessionID string) string {
	h := hmac.New(sha256.New, []byte(p.config.Password))
	h.Write([]byte(sessionID))
	signature := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return sessionID + "." + signature
}

// isValidCookie checks if a cookie is valid
func (p *PasswordProtect) isValidCookie(cookieValue string) bool {
	parts := strings.Split(cookieValue, ".")
	if len(parts) != 2 {
		return false
	}

	sessionID := parts[0]
	receivedSignature := parts[1]

	// Generate expected signature
	h := hmac.New(sha256.New, []byte(p.config.Password))
	h.Write([]byte(sessionID))
	expectedSignature := base64.URLEncoding.EncodeToString(h.Sum(nil))

	return receivedSignature == expectedSignature
}
