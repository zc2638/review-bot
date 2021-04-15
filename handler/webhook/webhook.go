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
		token := r.Header.Get("X-Gitlab-Token")
		claims, err := util.JwtParse(token, global.JWTSecret)
		if err != nil {
			ctr.Unauthorized(w, errors.New("Signature Token Invalid"))
			return
		}
		if claims.Auth.CheckSign(global.Cfg().SCM.Secret) {
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
		ctr.Success(w)
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
		event.Project.DefaultBranch,
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

func getMembers(pid string, names []string) (map[string]scm.ProjectMember, error) {
	var projectMembers []scm.ProjectMember
	members := make(map[string]scm.ProjectMember)
	for _, name := range names {
		if member, ok := scm.UserCached().Get(name); ok {
			members[name] = member
			continue
		}
		if projectMembers == nil {
			projectMembers, _ = global.SCM().ListProjectMembers(pid)
			for _, v := range projectMembers {
				scm.UserCached().Add(v.Username, v)
			}
		}
		if member, ok := scm.UserCached().Get(name); ok {
			members[name] = member
		}
	}
	return members, nil
}

// 当pull request创建时，添加标签
func openEvent(event *gitlab.MergeEvent) error {
	// 获取仓库review配置
	config, err := global.SCM().GetReviewConfig(
		event.Project.PathWithNamespace,
		event.Project.DefaultBranch, // 获取默认分支的配置
	)
	if err != nil {
		return err
	}

	// 初始化所有需要的label
	_ = initLabels(event)

	// 添加review check流程
	if err := global.SCM().UpdateBuildStatus(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.LastCommit.ID,
		scm.BuildStateRunning,
	); err != nil {
		return err
	}

	var addLabels []string
	for _, v := range scm.Labels {
		if strings.Contains(event.ObjectAttributes.Description, v.Order) {
			addLabels = append(addLabels, v.Name)
			break
		}
	}
	if len(addLabels) == 0 {
		addLabels = []string{scm.Labels[scm.LabelKindMissing].Name}
	}

	if err := global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.IID,
		&scm.PullRequest{
			AddLabels: addLabels,
		}); err != nil {
		return err
	}

	// 获取reviewers的用户id
	members, err := getMembers(event.Project.PathWithNamespace, config.Reviewers)
	if err != nil {
		return err
	}
	if len(members) > 0 {
		count := 0
		reviewers := make([]string, 0, 2)
		for _, v := range members {
			if v.ID == event.ObjectAttributes.AuthorID { // 跳过请求提交者自己进行review
				continue
			}
			reviewers = append(reviewers, "@"+v.Username)
			count++
			if count > 1 { // 限制每次请求两位reviewer
				break
			}
		}
		if len(reviewers) > 0 {
			reviewData := make([]string, 0, 4)
			reviewData = append(reviewData, "等待")
			reviewData = append(reviewData, reviewers...)
			reviewData = append(reviewData, "处理review请求")
			return global.SCM().CreatePullRequestComment(
				event.Project.PathWithNamespace,
				event.ObjectAttributes.IID,
				strings.Join(reviewData, " "),
			)
		}
	}
	return nil
}

func updateEvent(event *gitlab.MergeEvent) error {
	// 获取仓库review配置
	config, err := global.SCM().GetReviewConfig(
		event.Project.PathWithNamespace,
		event.Project.DefaultBranch, // 获取默认分支的配置
	)
	if err != nil {
		return err
	}
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
		var title, kind string
		if config.PRConfig.SquashWithTitle {
			title = event.ObjectAttributes.Title
		} else {
			desc := event.ObjectAttributes.Description
			titleData := strings.Split(desc, "<!-- title -->")
			if len(titleData) > 1 {
				titleData = strings.Split(titleData[1], "<!-- end title -->")
				if len(titleData) > 0 {
					title = strings.TrimSuffix(titleData[0], "\n")
					title = strings.TrimSpace(title)
					title = strings.TrimLeft(title, ">")
					title = strings.TrimSpace(title)
				}
			}
		}
		for _, v := range scm.Labels {
			if v.Type != scm.LabelTypeKind {
				continue
			}
			for _, vv := range event.Labels {
				if vv.Name == v.Name {
					kind = v.Short
					break
				}
			}
		}

		opt := &scm.MergePullRequest{
			MergeWhenPipelineSucceeds: true,
		}
		if event.ObjectAttributes.MergeParams != nil {
			opt.ShouldRemoveSourceBranch = event.ObjectAttributes.MergeParams.ForceRemoveSourceBranch
		}
		if kind != "" && title != "" {
			opt.Squash = true
			opt.SquashCommitMessage = kind + ": " + title
		}
		// 完成review check流程
		if err := global.SCM().UpdateBuildStatus(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.LastCommit.ID,
			scm.BuildStateSuccess,
		); err != nil {
			return err
		}

		// 执行合并
		return global.SCM().MergePullRequest(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.IID,
			opt,
		)
	}

	// 尝试添加review check流程，如果存在报错则忽略
	_ = global.SCM().UpdateBuildStatus(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.LastCommit.ID,
		scm.BuildStateRunning,
	)

	var addLabels, removeLabels []string
	var exists bool
	for _, v := range scm.Labels {
		if v.Type != scm.LabelTypeKind {
			continue
		}
		if strings.Contains(event.ObjectAttributes.Description, v.Order) {
			for _, label := range event.Labels {
				if label.Name == v.Name {
					exists = true
					break
				}
			}
			if exists {
				break
			}
			addLabels = append(addLabels, v.Name)
		} else {
			removeLabels = append(removeLabels, v.Name)
		}
	}
	if len(addLabels) == 0 && !exists {
		missingName := scm.Labels[scm.LabelKindMissing].Name
		addLabels = []string{missingName}
		var currentRemoveLabels []string
		for _, v := range removeLabels {
			if v == missingName {
				continue
			}
			currentRemoveLabels = append(currentRemoveLabels, v)
		}
		removeLabels = currentRemoveLabels
	}
	if len(addLabels) == 0 && len(removeLabels) == 0 {
		return nil
	}
	return global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.IID,
		&scm.PullRequest{
			AddLabels:    addLabels,
			RemoveLabels: removeLabels,
		},
	)
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
