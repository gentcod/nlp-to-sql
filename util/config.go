package util

import (
	"time"

	"github.com/gentcod/environ"
)

type Config struct {
	Port                string        `json:"PORT"`
	DBDriver            string        `json:"DB_DRIVER"`
	DBUrl               string        `json:"DB_URL"`
	DBName              string        `json:"DB_NAME"`
	Environment         string        `json:"ENVIRONMENT"`
	TokenSymmetricKey   string        `json:"TOKEN_SYMMETRIC_KEY"`
	TokenSecretKey      string        `json:"TOKEN_SECRET_KEY"`
	AccessTokenDuration time.Duration `json:"ACCESS_TOKEN_DURATION"`
	ApiKey              string        `json:"API_KEY"`
	OrgId               string        `json:"ORG_ID"`
	ProjectId           string        `json:"PROJECT_ID"`
	Model               string        `json:"MODEL"`
	Temp                string        `json:"TEMP"`
	CronSchedule        string        `json:"CRON_SCHEDULE"`
	CronBatchSize       string        `json:"CRON_BATCH_SIZE"`
	LogPath             string        `json:"LOG_PATH"`
}

func LoadConfig(path string) (config Config, err error) {
	err = environ.Init(path, &config)
	if err != nil {
		return
	}

	return
}
