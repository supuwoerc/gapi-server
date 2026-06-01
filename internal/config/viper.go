package config

import (
	"os"

	"github.com/spf13/viper"
)

func NewViper() *viper.Viper {
	v := viper.New()
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.SetConfigName("default")
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
	env := DetermineEnvironment()
	v.SetConfigName(env)
	if err := v.MergeInConfig(); err != nil {
		panic(err)
	}
	return v
}

func DetermineEnvironment() string {
	env := os.Getenv("APP_ENV")
	switch env {
	case "prod", "test":
		return env
	default:
		return "dev"
	}
}
