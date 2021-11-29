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
package handler

import (
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/cors"

	"github.com/zc2638/review-bot/handler/home"
	"github.com/zc2638/review-bot/handler/webhook"
	"github.com/zc2638/swag"
)

func New() http.Handler {
	mux := chi.NewRouter()
	mux.Use(
		middleware.Recoverer,
		middleware.Logger,
		cors.AllowAll().Handler,
	)

	apiDoc := swag.New(
		swag.Title("Review Bot API Doc"),
	)
	apiDoc.AddEndpointFunc(
		home.Register,
	)
	mux.Post("/webhook", webhook.HandlerEvent())

	apiDoc.RegisterMuxWithData(mux, false)
	return mux
}
