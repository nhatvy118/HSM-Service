package main

import (
	"log/slog"
	"net/http"
	"os"

	"hsm-server/handler"
	"hsm-server/usecase"

	"github.com/ThalesGroup/crypto11"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := loadConfig()
	if err != nil {
		logger.Error("config", "err", err)
		os.Exit(1)
	}

	hsmCtx, err := crypto11.Configure(&crypto11.Config{
		Path:       cfg.ModulePath,
		TokenLabel: cfg.TokenLabel,
		Pin:        cfg.UserPIN,
	})
	if err != nil {
		logger.Error("hsm configure", "err", err)
		os.Exit(1)
	}
	defer hsmCtx.Close()

	uc := usecase.New(hsmCtx, logger)
	h := handler.New(uc, logger)

	mux := http.NewServeMux()
	h.Register(mux, authMiddleware(cfg.APIKey))

	addr := ":" + cfg.Port
	logger.Info("hsm-server starting", "addr", addr, "token", cfg.TokenLabel)
	if err := http.ListenAndServe(addr, requestLogMiddleware(logger)(mux)); err != nil {
		logger.Error("server died", "err", err)
		os.Exit(1)
	}
}
