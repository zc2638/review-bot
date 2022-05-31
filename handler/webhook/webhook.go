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

package webhook

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/xanzy/go-gitlab"

	"github.com/zc2638/review-bot/handler/webhook/event"

	"github.com/pkg/errors"
	"github.com/pkgms/go/ctr"

	"github.com/zc2638/review-bot/global"
	"github.com/zc2638/review-bot/pkg/scm"
	"github.com/zc2638/review-bot/pkg/util"
)

func HandlerEvent(cfg *scm.Config, si scm.Interface) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Gitlab-Token")
		claims, err := util.JwtParse(token, global.JWTSecret)
		if err != nil {
			ctr.Unauthorized(w, errors.New("Signature Token Invalid"))
			return
		}
		if claims.Auth.CheckSign(cfg.Secret) {
			ctr.Unauthorized(w, errors.New("Signature Invalid"))
			return
		}
		defer r.Body.Close()
		data, err := ioutil.ReadAll(
			io.LimitReader(r.Body, 10000000),
		)
		if err != nil {
			ctr.InternalError(w, err)
			return
		}
		webhook, err := gitlab.ParseWebhook(gitlab.HookEventType(r), data)
		if err != nil {
			ctr.BadRequest(w, err)
			return
		}
		switch e := webhook.(type) {
		case *gitlab.MergeEvent:
			var scheme = "http"
			if r.URL != nil && r.URL.Scheme != "" {
				scheme = r.URL.Scheme
			}
			host := fmt.Sprintf("%s://%s", scheme, r.Host)
			mergeEvent, err := event.NewMerge(
				si, e.Project.PathWithNamespace, e.Project.DefaultBranch, e.ObjectAttributes.IID, host)
			if err != nil {
				ctr.InternalError(w, err)
				return
			}
			err = mergeEvent.Process(e)
		case *gitlab.MergeCommentEvent:
			commentEvent, err := event.NewComment(
				si, e.Project.PathWithNamespace, e.Project.DefaultBranch, e.MergeRequest.IID)
			if err != nil {
				ctr.InternalError(w, err)
				return
			}
			err = commentEvent.Process(e)
		}
		if err != nil {
			ctr.InternalError(w, err)
			return
		}
		ctr.Success(w)
	}
}
