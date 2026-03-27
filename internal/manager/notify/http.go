package notify

import (
	"net/http"
	"time"
)

// httpClient is a shared HTTP client with a 10-second timeout for all notification channels.
var httpClient = &http.Client{Timeout: 10 * time.Second}
