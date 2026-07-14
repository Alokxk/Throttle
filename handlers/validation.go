package handlers

var validAlgorithms = map[string]bool{
	"fixed_window":   true,
	"sliding_window": true,
	"token_bucket":   true,
}

// validateRateLimitParams checks algorithm/limit/window form a valid
// configuration. Shared by rule creation and inline /check requests, which
// both accept the same three fields under the same constraints.
func validateRateLimitParams(algorithm string, limit, window int) (code, message string, ok bool) {
	if !validAlgorithms[algorithm] {
		return "INVALID_ALGORITHM", "Algorithm must be fixed_window, sliding_window, or token_bucket", false
	}
	if limit <= 0 {
		return "INVALID_LIMIT", "Limit must be greater than 0", false
	}
	if algorithm != "token_bucket" && window <= 0 {
		return "INVALID_WINDOW", "Window must be greater than 0 for " + algorithm, false
	}
	return "", "", true
}
