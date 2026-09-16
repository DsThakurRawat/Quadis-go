package auth

import (
	"context"
	"errors"
	"fmt"

	"quadis-backend-go/internal/config"
	"quadis-backend-go/internal/repository"
)

var (
	ErrIncorrectPIN = errors.New("incorrect admin PIN")
	ErrInvalidNewPIN = errors.New("new PIN must be at least 4 digits")
)

type AdminPINService struct {
	repo repository.Repository
	cfg  *config.Config
}

func NewAdminPINService(repo repository.Repository, cfg *config.Config) *AdminPINService {
	return &AdminPINService{repo: repo, cfg: cfg}
}

// VerifyPIN verifies an incoming PIN against the stored scrypt hash
// If no PIN is configured in the database, it bootstraps from cfg.AdminPIN
func (s *AdminPINService) VerifyPIN(ctx context.Context, pin string) (bool, error) {
	storedHash, err := s.repo.GetAdminPINHash(ctx)
	if err != nil {
		return false, err
	}

	if storedHash == "" {
		// Bootstrap from config.AdminPIN
		bootstrapPIN := s.cfg.AdminPIN
		if bootstrapPIN == "" {
			bootstrapPIN = "998877"
		}

		hash, err := HashPassword(bootstrapPIN)
		if err != nil {
			return false, fmt.Errorf("failed to hash bootstrap PIN: %w", err)
		}

		if err := s.repo.SetAdminPINHash(ctx, hash); err != nil {
			return false, fmt.Errorf("failed to store bootstrapped PIN: %w", err)
		}
		storedHash = hash
	}

	return VerifyPassword(pin, storedHash), nil
}

// UpdatePIN verifies oldPIN and sets newPIN
func (s *AdminPINService) UpdatePIN(ctx context.Context, oldPIN, newPIN string) error {
	if len(newPIN) < 4 {
		return ErrInvalidNewPIN
	}

	valid, err := s.VerifyPIN(ctx, oldPIN)
	if err != nil {
		return err
	}
	if !valid {
		return ErrIncorrectPIN
	}

	newHash, err := HashPassword(newPIN)
	if err != nil {
		return fmt.Errorf("failed to hash new PIN: %w", err)
	}

	return s.repo.SetAdminPINHash(ctx, newHash)
}
