package handler

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"hsm-server/domain/entity"
	"hsm-server/usecase"
)

// Handler holds dependencies for HTTP handlers.
type Handler struct {
	uc     *usecase.HSM
	logger *slog.Logger
}

func New(uc *usecase.HSM, logger *slog.Logger) *Handler {
	return &Handler{uc: uc, logger: logger}
}

// Register wires routes onto mux. Public routes are added directly,
// authenticated routes are wrapped with the given auth middleware.
func (h *Handler) Register(mux *http.ServeMux, authMW func(http.Handler) http.Handler) {
	mux.HandleFunc("GET /health", h.Health)
	mux.Handle("POST /v1/keys", authMW(http.HandlerFunc(h.GenerateKey)))
	mux.Handle("GET /v1/keys/{label}/pubkey", authMW(http.HandlerFunc(h.ExportPubKey)))
	mux.Handle("POST /v1/keys/{label}/sign", authMW(http.HandlerFunc(h.Sign)))
	mux.Handle("POST /v1/keys/{label}/verify", authMW(http.HandlerFunc(h.Verify)))
}

// ---------- handlers ----------

func (h *Handler) Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, entity.HealthResponse{Status: "ok"})
}

func (h *Handler) GenerateKey(w http.ResponseWriter, r *http.Request) {
	var req entity.GenerateKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	if req.Label == "" {
		writeErr(w, http.StatusBadRequest, "label required")
		return
	}

	id, err := h.uc.GenerateKey(r.Context(), req.Label, req.Type)
	if err != nil {
		status, msg := mapServiceErr(err)
		writeErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusCreated, entity.GenerateKeyResponse{
		Label: req.Label,
		ID:    hex.EncodeToString(id),
		Type:  req.Type,
	})
}

func (h *Handler) ExportPubKey(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")

	pemBytes, err := h.uc.ExportPublicKey(r.Context(), label)
	if err != nil {
		status, msg := mapServiceErr(err)
		writeErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, entity.PublicKeyResponse{PEM: string(pemBytes)})
}

func (h *Handler) Sign(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	var req entity.SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "data must be base64")
		return
	}

	sig, err := h.uc.Sign(r.Context(), label, data, req.Algorithm)
	if err != nil {
		status, msg := mapServiceErr(err)
		writeErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, entity.SignResponse{
		Signature: base64.StdEncoding.EncodeToString(sig),
	})
}

func (h *Handler) Verify(w http.ResponseWriter, r *http.Request) {
	label := r.PathValue("label")
	var req entity.VerifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid body")
		return
	}
	data, err := base64.StdEncoding.DecodeString(req.Data)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "data must be base64")
		return
	}
	sig, err := base64.StdEncoding.DecodeString(req.Signature)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "signature must be base64")
		return
	}

	valid, err := h.uc.Verify(r.Context(), label, data, sig, req.Algorithm)
	if err != nil {
		status, msg := mapServiceErr(err)
		writeErr(w, status, msg)
		return
	}

	writeJSON(w, http.StatusOK, entity.VerifyResponse{Valid: valid})
}

// ---------- helpers ----------

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, entity.ErrorResponse{Error: msg})
}

func mapServiceErr(err error) (int, string) {
	switch {
	case errors.Is(err, usecase.ErrKeyNotFound):
		return http.StatusNotFound, "key not found"
	case errors.Is(err, usecase.ErrKeyExists):
		return http.StatusConflict, "key already exists"
	case errors.Is(err, usecase.ErrUnsupportedType),
		errors.Is(err, usecase.ErrUnsupportedKeyAlgo):
		return http.StatusBadRequest, err.Error()
	default:
		return http.StatusInternalServerError, err.Error()
	}
}
