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
package webhook

import (
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/zc2638/review-bot/pkg/util"

	"github.com/zc2638/review-bot/pkg/scm"

	"github.com/pkg/errors"
	"github.com/pkgms/go/ctr"
	"github.com/xanzy/go-gitlab"
	"github.com/zc2638/review-bot/global"
)

func HandlerEvent() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if global.Cfg().SCM.Secret != r.Header.Get("X-Gitlab-Token") {
			ctr.BadRequest(w, errors.New("Signature Invalid"))
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
		switch event := webhook.(type) {
		case *gitlab.MergeEvent:
			err = processMergeEvent(event)
		case *gitlab.MergeCommentEvent:
			err = processMergeCommentEvent(event)
		}
		if err != nil {
			ctr.InternalError(w, err)
			return
		}
		ctr.OK(w, webhook)
	}
}

// 处理merge事件
func processMergeEvent(event *gitlab.MergeEvent) error {
	var err error
	switch event.ObjectAttributes.Action {
	case "merge", "close", "reopen":
	case "open":
		err = openEvent(event)
	case "update":
		err = updateEvent(event)
	}
	return err
}

// 处理merge评论事件
func processMergeCommentEvent(event *gitlab.MergeCommentEvent) error {
	// 获取评论内容
	note := event.ObjectAttributes.Note

	// 获取仓库review配置
	config, err := global.SCM().GetReviewConfig(
		event.Project.PathWithNamespace,
		"master",
	)
	if err != nil {
		return err
	}

	var addLabels, removeLabels []string
	for k, v := range scm.Labels {
		switch v.Type {
		case scm.LabelTypeAdmin:
			// 匹配review权限
			if _, ok := util.InStringSlice(config.Reviewers, event.User.Username); ok &&
				k == scm.LabelLGTM &&
				strings.Contains(note, v.Order) {
				addLabels = append(addLabels, v.Name)
			}
			// 匹配approve权限
			if _, ok := util.InStringSlice(config.Approvers, event.User.Username); ok &&
				k == scm.LabelApprove &&
				strings.Contains(note, v.Order) {
				addLabels = append(addLabels, v.Name)
			}
		case scm.LabelTypeNormal: // 匹配通用标签
			if strings.Contains(note, v.Order) {
				addLabels = append(addLabels, v.Name)
			}
		case scm.LabelTypeRemove: // 匹配需要移除的标签
			if strings.Contains(note, v.Order) {
				removeLabels = append(removeLabels, v.Name)
			}
		}
	}

	// 匹配kind标签，仅限一个
	for _, v := range scm.Labels {
		if v.Type == scm.LabelTypeKind && strings.Contains(note, v.Order) {
			addLabels = append(addLabels, v.Name)
			removeLabels = append(removeLabels, scm.Labels[scm.LabelKindMissing].Name)
			break
		}
	}
	if len(addLabels) == 0 && len(removeLabels) == 0 {
		return nil
	}
	return global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.MergeRequest.IID,
		&scm.PullRequest{
			AddLabels:    addLabels,
			RemoveLabels: removeLabels,
		})
}

// 当pull request创建时，添加标签
func openEvent(event *gitlab.MergeEvent) error {
	_ = initLabels(event)
	var addLabels []string
	for _, v := range scm.Labels {
		if strings.Contains(event.ObjectAttributes.Description,
			"/"+strings.ReplaceAll(v.Name, "/", " ")) {
			addLabels = append(addLabels, v.Name)
			break
		}
	}
	if len(addLabels) == 0 {
		addLabels = []string{scm.Labels[scm.LabelKindMissing].Name}
	}
	return global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.IID,
		&scm.PullRequest{
			AddLabels: addLabels,
		})
}

func updateEvent(event *gitlab.MergeEvent) error {
	// 当label存在do-not-merge时，禁止合并
	// 当label满足lgtm和approved的时，执行分支合并
	var lgtmExists, approvedExists bool
	for _, v := range event.Labels {
		if strings.Contains(v.Name, scm.DoNotMerge) {
			approvedExists = false
			break
		}
		switch v.Name {
		case scm.Labels[scm.LabelLGTM].Name:
			lgtmExists = true
		case scm.Labels[scm.LabelApprove].Name:
			approvedExists = true
		}
	}
	if lgtmExists && approvedExists {
		return global.SCM().MergePullRequest(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.IID,
		)
	}
	return nil
}

// 初始化所有label
func initLabels(event *gitlab.MergeEvent) error {
	cache := scm.Cached()
	if exists := cache.IsExist(event.Project.PathWithNamespace); exists {
		return nil
	}
	labels, err := global.SCM().ListLabels(
		event.Project.PathWithNamespace,
	)
	if err != nil {
		return err
	}
	for _, v := range scm.Labels {
		exists := false
		for _, label := range labels {
			if v.Name == label.Name {
				exists = true
				break
			}
		}
		if !exists {
			// label创建失败暂不处理
			_ = global.SCM().CreateLabel(event.Project.PathWithNamespace, &v)
		}
	}
	cache.Add(event.Project.PathWithNamespace)
	return nil
}
