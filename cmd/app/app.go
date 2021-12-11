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
package app

import (
	"context"
	"fmt"
	"os"

	"github.com/pkgms/go/server"
	"github.com/zc2638/review-bot/handler"

	"github.com/spf13/cobra"
	"github.com/zc2638/review-bot/global"
)

var cfgFile string

func NewServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:          "bot",
		Short:        "review bot",
		Long:         `Review Bot.`,
		SilenceUsage: true,
		RunE:         run,
	}
	cfgFilePath := os.Getenv(global.EnvPrefix + "_CONFIG")
	if cfgFilePath == "" {
		cfgFilePath = "config/config.yaml"
	}
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", cfgFilePath, "config file (default is $HOME/config.yaml)")
	return cmd
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()
	cfg, err := global.ParseConfig(cfgFile)
	if err != nil {
		return err
	}
	if err := global.InitCfg(cfg); err != nil {
		return err
	}
	s := server.New(&cfg.Server)
	s.Handler = handler.New()
	fmt.Println("Listen on", s.Addr)
	return s.Run(ctx)
}
