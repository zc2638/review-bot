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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"github.com/pkgms/go/ctr"
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"

	"github.com/zc2638/review-bot/global"
	"github.com/zc2638/review-bot/pkg/scm"
	"github.com/zc2638/review-bot/pkg/util"
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
	case "approved":
		err = approveEvent(event, true)
	case "unapproved":
		err = approveEvent(event, false)
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

	adds, removes := dealCommonLabel(config, event.Project.PathWithNamespace, note)
	addLabels = append(addLabels, adds...)
	removeLabels = append(removeLabels, removes...)

	if len(addLabels) == 0 && len(removeLabels) == 0 {
		return nil
	}

	approveLabelName := scm.RemoveSet.LabelByKey("APPROVE").Name
	for _, v := range removeLabels {
		if v == approveLabelName {
			// TODO Don't handle the error for now, continue to execute down.
			// PREMIUM version supports this feature.
			_ = global.SCM().MergePullRequestApprove(
				event.Project.PathWithNamespace,
				event.MergeRequest.IID,
				false,
			)
			break
		}
	}

	opt := &scm.UpdatePullRequest{
		AddLabels:    addLabels,
		RemoveLabels: removeLabels,
		AssigneeID:   event.MergeRequest.AssigneeID,
		AssigneeIDs:  event.MergeRequest.AssigneeIDs,
	}
	return global.SCM().UpdatePullRequest(event.Project.PathWithNamespace, event.MergeRequest.IID, opt)
}

func dealCommonLabel(config *scm.ReviewConfig, repo string, content string) (adds []string, removes []string) {
	// 匹配common标签
	labels := scm.AddSet.FuzzyLabels(content)
	for _, v := range labels {
		adds = append(adds, v.Name)
	}
	labels = scm.RemoveSet.FuzzyLabels(content)
	for _, v := range labels {
		removes = append(removes, v.Name)
	}

	// 匹配custom标签
	labels = scm.CustomSet.FuzzyLabels(content)
	for _, v := range labels {
		adds = append(adds, v.Name)
	}
	// 匹配移除custom标签
	labels = scm.CustomSet.FuzzyLabelsWithPrefix("remove", content)
	for _, v := range labels {
		removes = append(removes, v.Name)
	}

	if len(config.CustomLabels) == 0 {
		return
	}

	// 匹配配置内的custom标签
	// 匹配移除配置内的custom标签
	var currentLabels []scm.Label
	for _, v := range config.CustomLabels {
		removeOrder := strings.TrimPrefix(v.Order, "/")
		removeOrder = "/remove-" + removeOrder
		if strings.Contains(content, removeOrder) {
			removes = append(removes, v.Name)
		}
		if strings.Contains(content, v.Order) {
			adds = append(adds, v.Name)
		}

		if !scm.RepoCached().IsExist(repo, v.Name) {
			if currentLabels == nil {
				var err error
				currentLabels, err = global.SCM().ListLabels(repo)
				if err != nil {
					logrus.Warningf("Sync custom labels failed: %s", err)
					return
				}
			}

			exists := false
			for _, vv := range currentLabels {
				if vv.Name == v.Name {
					exists = true
					break
				}
			}
			if !exists {
				// label创建失败暂不处理
				if err := global.SCM().CreateLabel(repo, &v); err != nil {
					logrus.Warningf("Create label failed: %s", err)
					continue
				}
			}
			scm.RepoCached().Add(repo, v.Name)
		}
	}
	return
}

func getMembers(pid string, names []string) map[string]scm.ProjectMember {
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
	return members
}

func addAutoComment(config *scm.ReviewConfig, event *gitlab.MergeEvent, host string) error {
	repo := event.Project.PathWithNamespace
	id := event.ObjectAttributes.IID
	authorID := event.ObjectAttributes.AuthorID

	count := 0
	reviewers := make([]string, 0, 2)

	// 获取reviewers的用户id
	members := getMembers(repo, config.Reviewers)
	for _, v := range members {
		if v.ID == authorID { // 跳过请求提交者自己进行review
			continue
		}
		reviewers = append(reviewers, "@"+v.Username)
		count++
		if count > 1 { // 限制每次请求两位reviewer
			break
		}
	}

	var reviewContent string
	if len(reviewers) > 0 {
		reviewData := make([]string, 0, 4)
		reviewData = append(reviewData, "等待")
		reviewData = append(reviewData, reviewers...)
		reviewData = append(reviewData, "处理 review 请求")
		reviewContent = strings.Join(reviewData, " ")
	}

	commandHelpURL := "http://" + host + "/command-help"
	commitMsg := "合并后的 commit 信息为 title 内容"
	if !config.PRConfig.SquashWithTitle {
		commitMsg = "合并后的 commit 信息为 描述中`<!-- title --><!-- end title -->`之间的内容"
	}

	content := `恭喜您，请求创建成功！  
` + reviewContent + `  

请注意，合并时将会压缩所有 commits ，` + commitMsg + `。  
可以在[【此处】](` + commandHelpURL + `)找到 bot 接受的完整指令列表。  

Reviewers(代码审查人员)可以通过评论` + "`/lgtm`" + `来表示审查通过。  
Approvers(请求审批人员)可以通过评论` + "`/approve`" + `来表示审批通过。  
Approvers(请求审批人员)可以通过评论` + "`/force-merge`" + `来进行强制合并。  
`
	return global.SCM().CreatePullRequestComment(repo, id, content)
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

	var eg errgroup.Group
	eg.Go(func() error {
		// TODO 需要检查pipeline是否存在，所以暂不处理错误
		// 添加review check流程
		err := global.SCM().UpdateBuildStatus(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.LastCommit.ID,
			scm.BuildStateRunning,
		)
		return err
	})

	eg.Go(func() error {
		// 更新labels
		adds, removes := dealCommonLabel(config, event.Project.PathWithNamespace, event.ObjectAttributes.Description)
		if len(adds) == 0 {
			return nil
		}

		opt := &scm.UpdatePullRequest{
			AddLabels:    adds,
			RemoveLabels: removes,
		}
		completeAssignees(event, opt)
		return global.SCM().UpdatePullRequest(event.Project.PathWithNamespace, event.ObjectAttributes.IID, opt)
	})

	eg.Go(func() error {
		// TODO 暂时忽略错误，需要处理，重复尝试5次
		// 添加自动评论
		err = addAutoComment(config, event, host)
		if err != nil {
			logrus.Warningf("open pull request add auto comment failed: %s", err)
		}
		return err
	})
	return eg.Wait()
}

func updateEvent(event *gitlab.MergeEvent) error {
	// TODO 更新commit自动移除LGTM

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

func approveEvent(event *gitlab.MergeEvent, approved bool) error {
	// 获取仓库review配置
	config, err := global.SCM().GetReviewConfig(
		event.Project.PathWithNamespace,
		event.Project.DefaultBranch, // 获取默认分支的配置
	)
	if err != nil {
		return err
	}
	if _, ok := util.InStringSlice(config.Approvers, event.User.Username); !ok {
		return fmt.Errorf("this user(%s) does not have the approve permission, the operation is prohibited", event.User.Username)
	}

	label := scm.AdminSet.LabelByKey("APPROVE").Name
	opt := &scm.UpdatePullRequest{}
	if approved {
		opt.AddLabels = append(opt.AddLabels, label)
	} else {
		opt.RemoveLabels = append(opt.RemoveLabels, label)
	}

	completeAssignees(event, opt)
	return global.SCM().UpdatePullRequest(event.Project.PathWithNamespace, event.ObjectAttributes.IID, opt)
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
		logrus.Errorln(err)
	}

	// 执行合并
	return global.SCM().MergePullRequest(
		event.Project.PathWithNamespace,
		event.MergeRequest.IID,
		opt,
	)
}

func completeAssignees(event *gitlab.MergeEvent, opt *scm.UpdatePullRequest) {
	getAssigneeIDs := func(event *gitlab.MergeEvent) []int {
		var ids []int
		for _, v := range event.Assignees {
			ids = append(ids, v.ID)
		}
		return ids
	}
	if event.Assignees != nil {
		opt.AssigneeIDs = getAssigneeIDs(event)
	}
	if event.Assignee != nil {
		opt.AssigneeID = event.Assignee.ID
	}
}
