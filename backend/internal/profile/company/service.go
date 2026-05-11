package company

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"gorm.io/gorm"
)

// Service defines operations for company profile management.
type Service interface {
	// GetCompanyByID retrieves a company record by its ID.
	// Returns ErrCompanyNotFound if no record exists.
	GetCompanyByID(id string) (*Record, error)

	// GetCompanyByOUHandle retrieves a company record by its IdP organisational unit handle.
	// Returns ErrCompanyNotFound if no record exists.
	GetCompanyByOUHandle(ouHandle string) (*Record, error)

	// GetCompanyByOUId retrieves a company record by its IdP organisational unit ID.
	// Returns ErrCompanyNotFound if no record exists.
	GetCompanyByOUId(ouId string) (*Record, error)

	// UpdateCompany performs an append-only merge of data into the company's Data field.
	// New keys are added and existing keys are replaced only when explicitly provided.
	// Keys absent from data are never removed.
	// Returns ErrCompanyNotFound if the company does not exist.
	UpdateCompany(id string, data map[string]any) error

	// Health checks if the service can access the database.
	Health() error
}

type service struct {
	db *gorm.DB
}

// NewService creates a new company service instance.
func NewService(db *gorm.DB) Service {
	return &service{db: db}
}

func (s *service) GetCompanyByID(id string) (*Record, error) {
	if id == "" {
		return nil, ErrInvalidCompanyID
	}

	var record Record
	result := s.db.Where("id = ?", id).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("company record not found", "id", id)
			return nil, ErrCompanyNotFound
		}
		slog.Error("failed to fetch company record", "id", id, "error", result.Error)
		return nil, fmt.Errorf("database query failed: %w", result.Error)
	}

	return &record, nil
}

func (s *service) GetCompanyByOUHandle(ouHandle string) (*Record, error) {
	var record Record
	result := s.db.Where("ou_handle = ?", ouHandle).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("company record not found", "ou_handle", ouHandle)
			return nil, ErrCompanyNotFound
		}
		slog.Error("failed to fetch company record", "ou_handle", ouHandle, "error", result.Error)
		return nil, fmt.Errorf("database query failed: %w", result.Error)
	}

	return &record, nil
}

func (s *service) GetCompanyByOUId(ouId string) (*Record, error) {
	var record Record
	result := s.db.Where("ou_id = ?", ouId).First(&record)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			slog.Debug("company record not found", "ou_id", ouId)
			return nil, ErrCompanyNotFound
		}
		slog.Error("failed to fetch company record", "ou_id", ouId, "error", result.Error)
		return nil, fmt.Errorf("database query failed: %w", result.Error)
	}

	return &record, nil
}

func (s *service) UpdateCompany(id string, data map[string]any) error {
	if id == "" {
		return ErrInvalidCompanyID
	}

	if len(data) == 0 {
		return nil
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal company data: %w", err)
	}

	// Use PostgreSQL's JSONB concatenation operator (||) for an atomic merge.
	// This avoids race conditions inherent in a read-modify-write cycle.
	result := s.db.Model(&Record{}).
		Where("id = ?", id).
		Update("data", gorm.Expr("data || ?", string(jsonBytes)))

	if result.Error != nil {
		slog.Error("failed to update company data", "id", id, "error", result.Error)
		return fmt.Errorf("failed to update company data: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrCompanyNotFound
	}

	return nil
}

func (s *service) Health() error {
	if err := s.db.Exec("SELECT 1").Error; err != nil {
		slog.Error("company service health check failed", "error", err)
		return fmt.Errorf("company service health check failed: %w", err)
	}
	return nil
}
