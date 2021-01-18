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
package home

import (
	"net/http"

	"github.com/zc2638/swag/endpoint"
	"github.com/zc2638/swag/swagger"
)

const tag = "home"

func Register(doc *swagger.API) {
	doc.AddTag(tag, "主要模块")
	doc.AddEndpoint(
		endpoint.New(
			http.MethodGet, "/",
			endpoint.Handler(index()),
			endpoint.ResponseSuccess(),
			endpoint.NoSecurity(),
		),
		endpoint.New(
			http.MethodGet, "/secret",
			endpoint.Handler(secret()),
			endpoint.Summary("生成webhook密钥"),
			endpoint.Query("namespace", "string", "仓库中间名称", true),
			endpoint.Query("name", "string", "仓库名称", true),
			endpoint.ResponseSuccess(),
			endpoint.NoSecurity(),
		),
		endpoint.New(
			http.MethodGet, "/download",
			endpoint.Handler(download("")),
			endpoint.Summary("模板文件下载"),
			endpoint.QueryDefault("type", "string", "版本系统类型", "gitlab", true),
			endpoint.ResponseSuccess(),
			endpoint.NoSecurity(),
		),
	)
}
