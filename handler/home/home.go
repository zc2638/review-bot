/**
 * Created by zc on 2021/1/15.
 */
package home

import (
	"net/http"

	"github.com/pkgms/go/ctr"
	"github.com/zc2638/swag/endpoint"

	"github.com/zc2638/swag/swagger"
)

const tag = "home"

func Register(doc *swagger.API) {
	doc.AddTag(tag, "主要模块")
	doc.AddEndpoint(endpoint.New(
		http.MethodGet, "/",
		endpoint.Handler(index()),
		endpoint.ResponseSuccess(),
		endpoint.NoSecurity(),
	))
}

func index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctr.OK(w, "Hello Bot!")
	}
}
