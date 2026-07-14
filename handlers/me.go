package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/Alokxk/Throttle/httpx"
	"github.com/Alokxk/Throttle/middleware"
)

// Me returns the authenticated client's own record. Lets a caller resolve
// its client_id from an API key alone, instead of having to remember and
// pass it separately — the dashboard is the first consumer of this.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		httpx.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed", "METHOD_NOT_ALLOWED")
		return
	}

	client := middleware.GetClientFromContext(r)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}
