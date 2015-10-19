package settings

// Proxy settings are changed via local webserver pages

import (
	"fmt"
	"net/http"
)

func ProxySettingsHandler(w http.ResponseWriter, r *http.Request) {
	// Don't cache response:
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate") // HTTP 1.1.
	w.Header().Set("Pragma", "no-cache")                                   // HTTP 1.0.
	w.Header().Set("Expires", "0")                                         // Proxies.
	fmt.Fprintf(w, `<h1>ProxyBlock Settings</h1><h3>Todo</h3>`)
}
