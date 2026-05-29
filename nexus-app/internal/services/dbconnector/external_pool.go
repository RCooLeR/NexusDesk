package dbconnector

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	defaultConnectorPoolMaxOpenConns = 4
	defaultConnectorPoolMaxIdleConns = 2
	defaultConnectorPoolIdleLifetime = 2 * time.Minute
	defaultConnectorPoolMaxLifetime  = 15 * time.Minute
)

type connectorDBPool struct {
	db        *sql.DB
	key       string
	version   string
	profileID string
	name      string
	kind      string
	driver    string
	createdAt time.Time
	lastUsed  time.Time
}

type ConnectorPoolStatus struct {
	ProfileID          string
	Name               string
	Kind               string
	Driver             string
	CreatedAt          time.Time
	LastUsed           time.Time
	MaxOpenConnections int
	OpenConnections    int
	InUse              int
	Idle               int
}

func (s *Service) externalConnectorDB(profile ConnectorProfile, driverName string, dsn string) (*sql.DB, error) {
	if s == nil {
		return nil, fmt.Errorf("connector service is unavailable")
	}
	key := connectorPoolKey(profile)
	version := connectorPoolVersion(profile, driverName, dsn)
	now := time.Now().UTC()
	s.poolMu.Lock()
	defer s.poolMu.Unlock()
	if s.connectorPools == nil {
		s.connectorPools = map[string]*connectorDBPool{}
	}
	if existing := s.connectorPools[key]; existing != nil {
		if existing.version == version {
			existing.lastUsed = now
			return existing.db, nil
		}
		_ = existing.db.Close()
		delete(s.connectorPools, key)
	}
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(defaultConnectorPoolMaxOpenConns)
	db.SetMaxIdleConns(defaultConnectorPoolMaxIdleConns)
	db.SetConnMaxIdleTime(defaultConnectorPoolIdleLifetime)
	db.SetConnMaxLifetime(defaultConnectorPoolMaxLifetime)
	s.connectorPools[key] = &connectorDBPool{
		db:        db,
		key:       key,
		version:   version,
		profileID: strings.TrimSpace(profile.ID),
		name:      strings.TrimSpace(profile.Name),
		kind:      strings.ToLower(strings.TrimSpace(profile.Kind)),
		driver:    driverName,
		createdAt: now,
		lastUsed:  now,
	}
	return db, nil
}

func (s *Service) ConnectorPoolStatuses() []ConnectorPoolStatus {
	if s == nil {
		return nil
	}
	s.poolMu.Lock()
	defer s.poolMu.Unlock()
	statuses := make([]ConnectorPoolStatus, 0, len(s.connectorPools))
	for _, pool := range s.connectorPools {
		stats := pool.db.Stats()
		statuses = append(statuses, ConnectorPoolStatus{
			ProfileID:          pool.profileID,
			Name:               pool.name,
			Kind:               pool.kind,
			Driver:             pool.driver,
			CreatedAt:          pool.createdAt,
			LastUsed:           pool.lastUsed,
			MaxOpenConnections: stats.MaxOpenConnections,
			OpenConnections:    stats.OpenConnections,
			InUse:              stats.InUse,
			Idle:               stats.Idle,
		})
	}
	return statuses
}

func (s *Service) CloseConnectorPools() error {
	if s == nil {
		return nil
	}
	s.poolMu.Lock()
	defer s.poolMu.Unlock()
	var closeErr error
	for key, pool := range s.connectorPools {
		if err := pool.db.Close(); err != nil && closeErr == nil {
			closeErr = err
		}
		delete(s.connectorPools, key)
	}
	return closeErr
}

func connectorPoolKey(profile ConnectorProfile) string {
	id := strings.TrimSpace(profile.ID)
	if id != "" {
		return strings.ToLower(strings.TrimSpace(profile.Kind)) + ":" + id
	}
	parts := []string{
		strings.ToLower(strings.TrimSpace(profile.Kind)),
		strings.ToLower(strings.TrimSpace(profile.Host)),
		fmt.Sprintf("%d", profile.Port),
		strings.TrimSpace(profile.Database),
		strings.TrimSpace(profile.Username),
	}
	return strings.Join(parts, "|")
}

func connectorPoolVersion(profile ConnectorProfile, driverName string, dsn string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{
		driverName,
		strings.ToLower(strings.TrimSpace(profile.Kind)),
		strings.TrimSpace(profile.ID),
		dsn,
	}, "\x00")))
	return hex.EncodeToString(sum[:])
}
