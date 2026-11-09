package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	database "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/robfig/cron/v3"
)

const (
	cronschedule = "@monthly"
	testschedule = "@every 30s"
	logfile      = "cron.txt"
)

// JobState represents the explicit execution state of the background cleanup cron
type JobState string

const (
	JobStateUnscheduled JobState = "UNSCHEDULED"
	JobStateIdle        JobState = "IDLE"
	JobStateRunning     JobState = "RUNNING"
	JobStateFailed      JobState = "FAILED"
)

type DBCron struct {
	store  database.Store
	c      *cron.Cron
	config CronConfig
	state  JobState
	mu     sync.RWMutex
}

type CronConfig struct {
	BatchSize string
	LogPath   string
}

func NewDBCron(store database.Store, config CronConfig) *DBCron {
	c := cron.New(cron.WithSeconds())

	return &DBCron{
		store:  store,
		config: config,
		c:      c,
		state:  JobStateUnscheduled,
	}
}

// State returns the current explicit state of the DBCron scheduler
func (dbcron *DBCron) State() JobState {
	dbcron.mu.RLock()
	defer dbcron.mu.RUnlock()
	return dbcron.state
}

// newDedicatedLogger opens the cron log file and returns a scoped logger so closing the file never corrupts global log output.
func newDedicatedLogger(logPath string) (*log.Logger, *os.File) {
	if logPath == "" {
		logPath = "./logs"
	}
	_ = os.MkdirAll(logPath, 0755)
	logFile := filepath.Join(logPath, logfile)
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return log.Default(), nil
	}
	return log.New(file, "[CRON] ", log.LstdFlags), file
}

func (dbcron *DBCron) runCleanup(batchSize int) {
	dbcron.mu.Lock()
	if dbcron.state == JobStateRunning {
		dbcron.mu.Unlock()
		return
	}
	dbcron.state = JobStateRunning
	dbcron.mu.Unlock()

	logger, logFile := newDedicatedLogger(dbcron.config.LogPath)
	if logFile != nil {
		defer logFile.Close()
	}

	totalDeleted, err := dbcron.store.DeleteExpDeletedUserRecords(context.Background(), batchSize)
	if err != nil {
		dbcron.mu.Lock()
		dbcron.state = JobStateFailed
		dbcron.mu.Unlock()
		logger.Printf("Error running cleanup job -> error during cleanup: %v. Total records deleted: %d", err, totalDeleted)
		return
	}

	dbcron.mu.Lock()
	dbcron.state = JobStateIdle
	dbcron.mu.Unlock()
	logger.Printf("Cleanup job completed successfully. Total records deleted: %d", totalDeleted)
}

func (dbcron *DBCron) InitCron() error {
	batchSize, err := strconv.Atoi(dbcron.config.BatchSize)
	if err != nil {
		return fmt.Errorf("error during cleanup: %v", err)
	}

	_, err = dbcron.c.AddFunc(cronschedule, func() {
		dbcron.runCleanup(batchSize)
	})
	if err != nil {
		return fmt.Errorf("error initializing and scheduling cleanup job: %v", err)
	}

	dbcron.mu.Lock()
	dbcron.state = JobStateIdle
	dbcron.mu.Unlock()

	log.Print("Cleanup job scheduled successfully")

	dbcron.c.Start()
	return nil
}
