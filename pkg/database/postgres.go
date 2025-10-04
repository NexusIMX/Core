package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dollarkillerx/im-system/pkg/config"
	"github.com/dollarkillerx/im-system/pkg/logger"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// NewPostgresDB creates a new PostgreSQL connection
func NewPostgresDB(cfg *config.DatabaseConfig) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host,
		cfg.Port,
		cfg.User,
		cfg.Password,
		cfg.DBName,
		cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)

	// Verify connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Log.Info("Connected to PostgreSQL",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
	)

	// Start partition management goroutine
	go managePartitions(db)

	return db, nil
}

// managePartitions creates daily partitions and drops old ones
func managePartitions(db *sql.DB) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		// Create partitions for today and tomorrow
		createPartitionsForDate(db, time.Now())
		createPartitionsForDate(db, time.Now().Add(24*time.Hour))

		// Drop old partitions
		dropOldPartitions(db)
	}
}

func createPartitionsForDate(db *sql.DB, date time.Time) {
	// Create message partition
	_, err := db.Exec("SELECT create_messages_partition($1)", date)
	if err != nil {
		logger.Log.Error("Failed to create messages partition",
			zap.Time("date", date),
			zap.Error(err),
		)
	}

	// Create file partition
	_, err = db.Exec("SELECT create_files_partition($1)", date)
	if err != nil {
		logger.Log.Error("Failed to create files partition",
			zap.Time("date", date),
			zap.Error(err),
		)
	}
}

func dropOldPartitions(db *sql.DB) {
	_, err := db.Exec("SELECT drop_old_partitions()")
	if err != nil {
		logger.Log.Error("Failed to drop old partitions", zap.Error(err))
	}
}
