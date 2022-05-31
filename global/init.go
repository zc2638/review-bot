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
	"fmt"

	"github.com/pkgms/go/ctr"
	"github.com/sirupsen/logrus"

	"github.com/zc2638/review-bot/pkg/scm"
)

var config *Config

func InitCfg(cfg *Config) (err error) {
	config = cfg

	level, err := logrus.ParseLevel(cfg.Logger.Level)
	if err != nil {
		return fmt.Errorf("parse logger level failed: %v", err)
	}
	logrus.SetLevel(level)

	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors:            true,
		DisableLevelTruncation: true,
		PadLevelText:           true,
		FullTimestamp:          true,
		TimestampFormat:        "2006/01/02 15:04:05",
	})
	ctr.InitLog(logrus.StandardLogger())
	return initSCM(&cfg.SCM)
}

func Cfg() *Config {
	return config
}

var scmClient scm.Interface

func initSCM(cfg *scm.Config) error {
	var err error
	scmClient, err = scm.NewGitlabClient(cfg)
	return err
}

func SCM() scm.Interface {
	return scmClient
}
