package util

import (
	"time"

	"github.com/gentcod/environ"
)

type Config struct {
	Port                string
	DBDriver            string
	DBUrl               string
	DBName              string
	Environment         string
	TokenSymmetricKey   string
	TokenSecretKey      string
	AccessTokenDuration time.Duration
	ApiKey              string
	OrgId               string
	ProjectId           string
	Model               string
	Temp                string
	CronSchedule        string
	CronBatchSize       string
	LogPath             string
}

func LoadConfig(path string) (config Config, err error) {
	err = environ.Init(path, &config)
	if err != nil {
		return
	}

	return
}
