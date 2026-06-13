package models

// StatusResponse represents the API status response
type StatusResponse struct {
	Status map[string]string `json:"status"`
}

// IncusContainer represents an Incus container
type IncusContainer struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	IPv4      string `json:"ipv4,omitempty"`
	IPv6      string `json:"ipv6,omitempty"`
	Type      string `json:"type"`
	Ephemeral bool   `json:"ephemeral"`
}

// IncusListResponse represents the response from Incus list command
type IncusListResponse struct {
	Containers []IncusContainer `json:"containers"`
	Status     string           `json:"status"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}
