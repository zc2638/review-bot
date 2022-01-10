/*
Copyright © 2021 zc2638 <zc2638@qq.com>.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package global

import (
	"fmt"
	"os"
	"strings"

	"github.com/mitchellh/go-homedir"
	"github.com/mitchellh/mapstructure"
	"github.com/pkgms/go/server"
	"github.com/spf13/viper"

	"github.com/zc2638/review-bot/pkg/scm"
)

const EnvPrefix = "BOT"

const JWTSecret = "bot"

type Config struct {
	Server server.Config `json:"server"`
	SCM    scm.Config    `json:"scm"`
}

func Environ() *Config {
	cfg := &Config{}
	cfg.Server.Port = 2640
	cfg.SCM.Type = "gitlab"
	return cfg
}

func ParseConfig(cfgPath string) (*Config, error) {
	if cfgPath != "" {
		viper.SetConfigFile(cfgPath)
	} else {
		home, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		viper.AddConfigPath(home)
		viper.SetConfigName("config.yaml")
	}
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()
	cfg := Environ()
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(*os.PathError); ok {
			fmt.Println("Warning: not find config file, use default.")
			return cfg, nil
		}
		return nil, err
	}
	fmt.Println("Using config file:", viper.ConfigFileUsed())
	err := viper.Unmarshal(cfg, func(dc *mapstructure.DecoderConfig) {
		dc.TagName = "json"
	})
	return cfg, err
}
