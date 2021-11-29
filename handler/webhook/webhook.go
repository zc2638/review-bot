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

	"github.com/sirupsen/logrus"

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
			err = processMergeEvent(event, r.Host)
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
func processMergeEvent(event *gitlab.MergeEvent, host string) error {
	var err error
	switch event.ObjectAttributes.Action {
	case "merge", "close", "reopen":
	case "open":
		err = openEvent(event, host)
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

	// 匹配admin标签
	if _, ok := util.InStringSlice(config.Reviewers, event.User.Username); ok {
		label := scm.AdminSet.FuzzyLabelWithKey("LGTM", note)
		if label != nil {
			addLabels = append(addLabels, label.Name)
		}
	}
	if _, ok := util.InStringSlice(config.Approvers, event.User.Username); ok {
		label := scm.AdminSet.FuzzyLabelWithKey("APPROVE", note)
		if label != nil {
			addLabels = append(addLabels, label.Name)
		}
		label = scm.AdminSet.FuzzyLabelWithKey("FORCE-MERGE", note)
		if label != nil {
			logrus.Infof("Run force merge by %s on PR(%v) in Repo(%s)",
				event.User.Username, event.MergeRequest.IID, event.Project.PathWithNamespace)
			return commentMerge(event, config)
		}
	}

	// 匹配common标签
	labels := scm.AddSet.FuzzyLabels(note)
	for _, v := range labels {
		addLabels = append(addLabels, v.Name)
	}
	labels = scm.RemoveSet.FuzzyLabels(note)
	for _, v := range labels {
		removeLabels = append(removeLabels, v.Name)
	}

	// 匹配custom标签
	labels = scm.CustomSet.FuzzyLabels(note)
	for _, v := range labels {
		addLabels = append(addLabels, v.Name)
	}
	// 匹配移除custom标签
	labels = scm.CustomSet.FuzzyLabelsWithPrefix("remove", note)
	for _, v := range labels {
		removeLabels = append(removeLabels, v.Name)
	}

	// 匹配配置内的custom标签
	// 匹配移除配置内的custom标签
	for _, v := range config.CustomLabels {
		// 创建不存在的标签，添加custom标签到缓存
		if !scm.RepoCached().IsExist(event.Project.PathWithNamespace, v.Name) {
			// label创建失败暂不处理
			if err := global.SCM().CreateLabel(event.Project.PathWithNamespace, &v); err != nil {
				logrus.Warningf("Create label failed: %s", err)
			} else {
				scm.RepoCached().Add(event.Project.PathWithNamespace, v.Name)
			}
		}

		removeOrder := strings.TrimPrefix(v.Order, "/")
		removeOrder = "/remove-" + removeOrder
		if strings.Contains(note, v.Order) {
			addLabels = append(addLabels, v.Name)
		}
		if strings.Contains(note, removeOrder) {
			removeLabels = append(removeLabels, v.Name)
		}
	}

	if len(addLabels) == 0 && len(removeLabels) == 0 {
		return nil
	}
	return global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.MergeRequest.IID,
		&scm.UpdatePullRequest{
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

func addAutoComment(namespace string, id int, host string) error {
	commandHelpURL := "http://" + host + "/command-help"

	content := `请求创建成功，恭喜您！
请注意，合并时将会压缩所有commits，合并后的commit为title内容。
可以在[【此处】](` + commandHelpURL + `)找到此机器人接受的命令的完整列表。

Reviewers(代码审查人员)可以通过评论` + "`/lgtm`" + `来表示审查通过。
Approvers(请求审批人员)可以通过评论` + "`/approve`" + `来表示审批通过。
Approvers(请求审批人员)可以通过评论` + "`/force-merge`" + `来进行强制合并。`
	return global.SCM().CreatePullRequestComment(namespace, id, content)
}

// 当pull request创建时，添加标签
func openEvent(event *gitlab.MergeEvent, host string) error {
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

	labels := scm.CustomSet.FuzzyLabels(event.ObjectAttributes.Description)
	for _, v := range labels {
		addLabels = append(addLabels, v.Name)
	}
	if len(addLabels) == 0 {
		label := scm.AutoSet.LabelByKey("KIND")
		if label != nil {
			addLabels = append(addLabels, label.Name)
		}
	}

	// 更新labels
	if err := global.SCM().UpdatePullRequest(
		event.Project.PathWithNamespace,
		event.ObjectAttributes.IID,
		&scm.UpdatePullRequest{
			AddLabels: addLabels,
		}); err != nil {
		return err
	}

	// 添加自动评论
	if err := addAutoComment(event.Project.PathWithNamespace, event.ObjectAttributes.IID, host); err != nil {
		// TODO 暂时忽略错误，需要处理，重复尝试5次
		logrus.Warningf("open pull request add auto comment failed: %s", err)
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
	var lgtmExists, approvedExists bool
	for _, v := range event.Labels {
		if strings.Contains(v.Name, scm.DoNotMerge) {
			approvedExists = false
			break
		}
		label := scm.AdminSet.LabelByKey("LGTM")
		if label != nil && v.Name == label.Name {
			lgtmExists = true
		}
		label = scm.AdminSet.LabelByKey("APPROVE")
		if label != nil && v.Name == label.Name {
			approvedExists = true
		}
	}

	// 尝试添加review check流程，如果存在报错则忽略
	if !lgtmExists || !approvedExists {
		_ = global.SCM().UpdateBuildStatus(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.LastCommit.ID,
			scm.BuildStateRunning,
		)
		return nil
	}

	// 当label满足lgtm和approved的时，执行分支合并
	var title, prefix string
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

	for _, v := range event.Labels {
		label := scm.CustomSet.Label(v.Name)
		if label != nil && strings.TrimSpace(label.Short) != "" {
			prefix = label.Short + ":"
			break
		}
	}

	opt := &scm.MergePullRequest{MergeWhenPipelineSucceeds: true}
	if event.ObjectAttributes.MergeParams != nil {
		opt.ShouldRemoveSourceBranch = event.ObjectAttributes.MergeParams.ForceRemoveSourceBranch
	}
	if title != "" {
		opt.Squash = true
		opt.SquashCommitMessage = prefix + title
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
	return global.SCM().MergePullRequest(event.Project.PathWithNamespace, event.ObjectAttributes.IID, opt)
}

// 初始化所有label
func initLabels(event *gitlab.MergeEvent) error {
	cache := scm.Cached()
	if exists := cache.IsExist(event.Project.PathWithNamespace); exists {
		return nil
	}

	labels, err := global.SCM().ListLabels(event.Project.PathWithNamespace)
	if err != nil {
		return err
	}

	var allLabels []scm.Label
	allLabels = append(allLabels, scm.AutoSet.Labels()...)
	allLabels = append(allLabels, scm.AdminSet.Labels()...)
	allLabels = append(allLabels, scm.AddSet.Labels()...)
	allLabels = append(allLabels, scm.CustomSet.Labels()...)
	for _, v := range allLabels {
		var exists bool
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

// 通过评论合并PR
func commentMerge(event *gitlab.MergeCommentEvent, config *scm.ReviewConfig) error {
	// 获取pr详细信息
	pr, err := global.SCM().GetPullRequest(
		event.Project.PathWithNamespace,
		event.MergeRequest.IID,
	)
	if err != nil {
		return err
	}

	var title, prefix string
	if config.PRConfig.SquashWithTitle {
		title = pr.Title
	} else {
		titleData := strings.Split(pr.Description, "<!-- title -->")
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

	for _, v := range pr.Labels {
		label := scm.CustomSet.Label(v)
		if label != nil && strings.TrimSpace(label.Short) != "" {
			prefix = label.Short + ":"
			break
		}
	}

	opt := &scm.MergePullRequest{
		MergeWhenPipelineSucceeds: true,
	}
	if event.MergeRequest.MergeParams != nil {
		opt.ShouldRemoveSourceBranch = pr.ForceRemoveSourceBranch
	}
	if title != "" {
		opt.Squash = true
		opt.SquashCommitMessage = prefix + title
	}
	// 完成review check流程
	if err := global.SCM().UpdateBuildStatus(
		event.Project.PathWithNamespace,
		event.MergeRequest.LastCommit.ID,
		scm.BuildStateSuccess,
	); err != nil {
		return err
	}

	// 执行合并
	return global.SCM().MergePullRequest(
		event.Project.PathWithNamespace,
		event.MergeRequest.IID,
		opt,
	)
}
