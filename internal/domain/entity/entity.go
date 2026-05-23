// Package entity holds request/response data transfer objects exchanged
// between the HTTP handlers and clients.
package entity

// ---------- Key management ----------

type GenerateKeyRequest struct {
	Label string `json:"label"`
	Type  string `json:"type"` // ECDSA-P256, ECDSA-P384, RSA-2048, RSA-4096
}

type GenerateKeyResponse struct {
	Label string `json:"label"`
	ID    string `json:"id"` // hex
	Type  string `json:"type"`
}

type PublicKeyResponse struct {
	PEM string `json:"pem"`
}

// ---------- Sign / Verify ----------

type SignRequest struct {
	Data      string `json:"data"`      // base64
	Algorithm string `json:"algorithm"` // ECDSA-SHA256
}

type SignResponse struct {
	Signature string `json:"signature"` // base64 DER
}

type VerifyRequest struct {
	Data      string `json:"data"`
	Signature string `json:"signature"`
	Algorithm string `json:"algorithm"`
}

type VerifyResponse struct {
	Valid bool `json:"valid"`
}

// ---------- Common ----------

type HealthResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}
