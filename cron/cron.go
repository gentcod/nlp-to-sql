package cron

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	database "github.com/gentcod/nlp-to-sql/internal/database"
	"github.com/robfig/cron/v3"
)

const (
	cronschedule = "@monthly"
	testschedule = "@every 30s"
	logfile      = "/cron.txt"
)

type DBCron struct {
	store  database.Store
	c      *cron.Cron
	config CronConfig
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
	}
}

func setupLogging(logPath string) *os.File {
	logFile := fmt.Sprintf("%s%s", logPath, logfile)
	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatal(err)
	}
	log.SetOutput(file)
	return file
}

func (dbcron *DBCron) runCleanup(batchSize int) {
	logFile := setupLogging(dbcron.config.LogPath)
	defer logFile.Close()

	var err error
	totalDeleted, err := dbcron.store.DeleteExpDeletedUserRecords(context.Background(), batchSize)
	if err != nil {
		err = fmt.Errorf("error during cleanup: %v", err)
		log.Printf("Eror running cleanup job -> %v. Total records deleted: %d", err, totalDeleted)
	} else {
		log.Printf("Cleanup job completed successfully. Total records deleted: %d", totalDeleted)
	}
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

	log.Print("Cleanup job scheduled successfully")

	dbcron.c.Start()
	return nil
}
