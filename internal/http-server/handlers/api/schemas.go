package api

type RespReport struct {
    Status string `json:"status"`
    Error string `json:"error,omitempty"`
}
