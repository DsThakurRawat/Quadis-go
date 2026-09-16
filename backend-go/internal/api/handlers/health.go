package handlers

import (
	"net/http"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/repository"
)

type HealthHandler struct {
	repo repository.Repository
	cfg  *config.Config
}

func NewHealthHandler(repo repository.Repository, cfg *config.Config) *HealthHandler {
	return &HealthHandler{repo: repo, cfg: cfg}
}

// Health checks the storage status and returns 200 (healthy) or 503 (degraded)
func (h *HealthHandler) Health(w http.ResponseWriter, r *http.Request) {
	storage, reachable, err := h.repo.Ping(r.Context())

	inMemoryOnProd := storage == "in-memory" && h.cfg.IsProduction()
	healthy := reachable && !inMemoryOnProd

	res := map[string]interface{}{
		"success":  healthy,
		"service":  "Quadis Hotels API Server",
		"version":  "1.0.0",
		"status":   "degraded",
		"storage":  storage,
		"database": "unreachable",
	}

	if reachable {
		res["database"] = "ok"
	}
	if healthy {
		res["status"] = "healthy"
	}

	if err != nil {
		res["error"] = err.Error()
	}
	if inMemoryOnProd {
		res["error"] = "Running in memory on production — DATABASE_URL is unset. Bookings will not survive a restart."
	}

	status := http.StatusOK
	if !healthy {
		status = http.StatusServiceUnavailable
	}

	JSON(w, status, res)
}

// Healthz serves a simple plain text "ok\n" for lightweight load balancer checks.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok\n"))
}
