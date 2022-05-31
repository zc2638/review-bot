// Copyright © 2022 zc2638 <zc2638@qq.com>.
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

package event

import (
	"fmt"
	"strings"

	"github.com/zc2638/review-bot/pkg/util"

	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"
	"golang.org/x/sync/errgroup"

	"github.com/zc2638/review-bot/global"

	"github.com/zc2638/review-bot/pkg/scm"
)

func NewMerge(si scm.Interface, pid string, ref string, prID int, host string) (*Merge, error) {
	pr, err := si.GetPullRequest(pid, prID)
	if err != nil {
		return nil, err
	}
	cfg, err := si.GetReviewConfig(pid, ref)
	if err != nil {
		return nil, err
	}
	return &Merge{
		si:   si,
		cfg:  cfg,
		pr:   pr,
		pid:  pid,
		prID: prID,
		host: host,
	}, nil
}

type Merge struct {
	si  scm.Interface
	cfg *scm.ReviewConfig
	pr  *scm.PullRequest

	pid  string
	prID int
	host string
}

func (e *Merge) Process(event *gitlab.MergeEvent) error {
	// 处理merge事件
	var err error
	switch event.ObjectAttributes.Action {
	case "merge", "close", "reopen":
	case "open":
		err = e.open(event)
	case "update":
		err = e.update(event)
	case "approved":
		err = e.approve(event, true)
	case "unapproved":
		err = e.approve(event, false)
	}
	return err
}

func (e *Merge) approve(event *gitlab.MergeEvent, approved bool) error {
	if _, ok := util.InStringSlice(e.cfg.Approvers, event.User.Username); !ok {
		return fmt.Errorf("this user(%s) does not have the approve permission, the operation is prohibited", event.User.Username)
	}

	label := scm.AdminSet.LabelByKey("APPROVE").Name
	opt := &scm.UpdatePullRequest{}
	if approved {
		opt.AddLabels = append(opt.AddLabels, label)
	} else {
		opt.RemoveLabels = append(opt.RemoveLabels, label)
	}
	opt.Labels = filterLabels(e.pr.Labels, opt.AddLabels, opt.RemoveLabels)

	e.completeAssignees(event, opt)
	return e.si.UpdatePullRequest(e.pid, e.prID, opt)
}

func (e *Merge) open(event *gitlab.MergeEvent) error {
	// 初始化所有需要的label
	_ = e.initLabels()

	var eg errgroup.Group
	eg.Go(func() error {
		// TODO 需要检查pipeline是否存在，所以暂不处理错误
		// 添加review check流程
		return e.si.UpdateBuildStatus(
			event.Project.PathWithNamespace,
			event.ObjectAttributes.LastCommit.ID,
			scm.BuildStateRunning,
		)
	})

	eg.Go(func() error {
		// 更新labels
		adds, removes := dealCommonLabel(e.cfg, e.pid, event.ObjectAttributes.Description)
		if len(adds) == 0 {
			return nil
		}

		opt := &scm.UpdatePullRequest{
			Labels:       filterLabels(e.pr.Labels, adds, removes),
			AddLabels:    adds,
			RemoveLabels: removes,
		}
		e.completeAssignees(event, opt)
		return e.si.UpdatePullRequest(e.pid, e.prID, opt)
	})

	eg.Go(func() error {
		// TODO 暂时忽略错误，需要处理，比如重复尝试5次
		// 添加自动评论
		err := e.addAutoComment(event)
		if err != nil {
			logrus.Warningf("open pull request add auto comment failed: %s", err)
		}
		return err
	})
	return eg.Wait()
}

func (e *Merge) update(event *gitlab.MergeEvent) error {
	// TODO 更新commit自动移除LGTM

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
	return e.merge(event.ObjectAttributes.LastCommit.ID)
}

func (e *Merge) merge(lastCommitID string) error {
	var title, prefix string
	if e.cfg.PRConfig.SquashWithTitle {
		title = e.pr.Title
	} else {
		titleData := strings.Split(e.pr.Description, "<!-- title -->")
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

	for _, v := range e.pr.Labels {
		label := scm.CustomSet.Label(v)
		if label != nil && strings.TrimSpace(label.Short) != "" {
			prefix = label.Short + ":"
			break
		}
	}

	opt := &scm.MergePullRequest{
		MergeWhenPipelineSucceeds: true,
		ShouldRemoveSourceBranch:  true,
	}
	if title != "" {
		opt.Squash = true
		opt.SquashCommitMessage = prefix + title
	}
	// 完成review check流程
	if err := e.si.UpdateBuildStatus(e.pid, lastCommitID, scm.BuildStateSuccess); err != nil {
		logrus.Errorln(err)
	}
	// 执行合并
	return e.si.MergePullRequest(e.pid, e.prID, opt)
}

func (e *Merge) initLabels() error {
	cache := scm.Cached()
	if exists := cache.IsExist(e.pid); exists {
		return nil
	}

	labels, err := e.si.ListLabels(e.pid)
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
			_ = e.si.CreateLabel(e.pid, &v)
		}
	}

	cache.Add(e.pid)
	return nil
}

func (e *Merge) completeAssignees(event *gitlab.MergeEvent, opt *scm.UpdatePullRequest) {
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

func (e *Merge) addAutoComment(event *gitlab.MergeEvent) error {
	repo := event.Project.PathWithNamespace
	id := event.ObjectAttributes.IID
	authorID := event.ObjectAttributes.AuthorID

	count := 0
	reviewers := make([]string, 0, 2)

	// 获取reviewers的用户id
	members := e.getMembers(e.cfg.Reviewers)
	for _, v := range members {
		if v.ID == authorID {
			// 跳过 请求提交者 进行review
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

	commandHelpURL := e.host + "/command-help"
	commitMsg := "合并后的 commit 信息为 title 内容"
	if !e.cfg.PRConfig.SquashWithTitle {
		commitMsg = "合并后的 commit 信息为 描述中`<!-- title --><!-- end title -->`之间的内容"
	}

	content := `您好 ` + event.User.Username + `，请求创建成功！  
` + reviewContent + `  

请注意，合并时将会压缩所有 commits ，` + commitMsg + `。  
可以在[【此处】](` + commandHelpURL + `)找到 bot 接受的完整指令列表。  

Reviewers(代码审查人员)可以通过评论` + "`/lgtm`" + `来表示审查通过。  
Approvers(请求审批人员)可以通过评论` + "`/approve`" + `来表示审批通过。  
Approvers(请求审批人员)可以通过评论` + "`/force-merge`" + `来进行强制合并。  
`
	return global.SCM().CreatePullRequestComment(repo, id, content)
}

func (e *Merge) getMembers(names []string) map[string]scm.ProjectMember {
	var projectMembers []scm.ProjectMember
	members := make(map[string]scm.ProjectMember)
	for _, name := range names {
		if member, ok := scm.UserCached().Get(name); ok {
			members[name] = member
			continue
		}
		if projectMembers == nil {
			// TODO 无需处理错误，错误时会返回nil
			projectMembers, _ = e.si.ListProjectMembers(e.pid)
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
