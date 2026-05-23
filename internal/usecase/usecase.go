package usecase

import (
	"context"
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"log/slog"

	"github.com/ThalesGroup/crypto11"
)

const AlgECDSASHA256 = "ECDSA-SHA256"

var (
	ErrKeyNotFound        = errors.New("key not found")
	ErrKeyExists          = errors.New("key already exists")
	ErrUnsupportedType    = errors.New("unsupported key type")
	ErrUnsupportedKeyAlgo = errors.New("key algorithm mismatch")
)

// HSM is the HSM usecase: wraps crypto11 operations.
// Safe for concurrent use.
type HSM struct {
	hsmCtx *crypto11.Context
	logger *slog.Logger
}

func New(hsmCtx *crypto11.Context, logger *slog.Logger) *HSM {
	return &HSM{hsmCtx: hsmCtx, logger: logger}
}

// GenerateKey creates a new key pair in the HSM. Returns the assigned id.
func (u *HSM) GenerateKey(ctx context.Context, label, keyType string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if existing, _ := u.hsmCtx.FindKeyPair(nil, []byte(label)); existing != nil {
		return nil, ErrKeyExists
	}

	id := make([]byte, 4)
	_, _ = rand.Read(id)
	lbl := []byte(label)

	var err error
	switch keyType {
	case "ECDSA-P256":
		_, err = u.hsmCtx.GenerateECDSAKeyPairWithLabel(id, lbl, elliptic.P256())
	case "ECDSA-P384":
		_, err = u.hsmCtx.GenerateECDSAKeyPairWithLabel(id, lbl, elliptic.P384())
	case "RSA-2048":
		_, err = u.hsmCtx.GenerateRSAKeyPairWithLabel(id, lbl, 2048)
	case "RSA-4096":
		_, err = u.hsmCtx.GenerateRSAKeyPairWithLabel(id, lbl, 4096)
	default:
		return nil, ErrUnsupportedType
	}
	if err != nil {
		return nil, fmt.Errorf("generate: %w", err)
	}
	u.logger.InfoContext(ctx, "key generated", "label", label, "type", keyType)
	return id, nil
}

// ExportPublicKey returns the PEM-encoded public key for the given label.
func (u *HSM) ExportPublicKey(ctx context.Context, label string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	pair, err := u.hsmCtx.FindKeyPair(nil, []byte(label))
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrKeyNotFound
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pair.Public())
	if err != nil {
		return nil, err
	}
	return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER}), nil
}

// Sign hashes data with SHA-256, then signs with the named ECDSA key.
// Returns the DER-encoded ECDSA signature.
func (u *HSM) Sign(ctx context.Context, label string, data []byte, algorithm string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if algorithm != AlgECDSASHA256 {
		return nil, fmt.Errorf("%w: only %s supported", ErrUnsupportedType, AlgECDSASHA256)
	}

	pair, err := u.hsmCtx.FindKeyPair(nil, []byte(label))
	if err != nil {
		return nil, err
	}
	if pair == nil {
		return nil, ErrKeyNotFound
	}

	hash := sha256.Sum256(data)
	sig, err := pair.Sign(rand.Reader, hash[:], crypto.SHA256)
	if err != nil {
		return nil, fmt.Errorf("sign: %w", err)
	}
	u.logger.InfoContext(ctx, "sign ok", "label", label, "bytes", len(data))
	return sig, nil
}

// Verify validates an ECDSA-SHA256 signature using the public key from the HSM.
func (u *HSM) Verify(ctx context.Context, label string, data, sig []byte, algorithm string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if algorithm != AlgECDSASHA256 {
		return false, fmt.Errorf("%w: only %s supported", ErrUnsupportedType, AlgECDSASHA256)
	}

	pair, err := u.hsmCtx.FindKeyPair(nil, []byte(label))
	if err != nil {
		return false, err
	}
	if pair == nil {
		return false, ErrKeyNotFound
	}
	pub, ok := pair.Public().(*ecdsa.PublicKey)
	if !ok {
		return false, ErrUnsupportedKeyAlgo
	}

	hash := sha256.Sum256(data)
	valid := ecdsa.VerifyASN1(pub, hash[:], sig)
	u.logger.InfoContext(ctx, "verify done", "label", label, "valid", valid)
	return valid, nil
}
