// Copyright © 2021 zc2638 <zc2638@qq.com>.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package global

import (
	"github.com/99nil/gopkg/server"

	"github.com/zc2638/review-bot/pkg/scm"
)

const EnvPrefix = "BOT"

const JWTSecret = "bot"

type Config struct {
	Server server.Config `json:"server"`
	SCM    scm.Config    `json:"scm"`
	Logger LoggerConfig  `json:"logger"`
}

type LoggerConfig struct {
	Level string `json:"level"` // debug、info、warn、error、fatal、panic、trace
}

func Environ() *Config {
	cfg := &Config{}
	cfg.Server.Port = 2640
	cfg.SCM.Type = "gitlab"
	return cfg
}
