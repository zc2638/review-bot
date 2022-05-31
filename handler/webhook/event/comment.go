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
	"github.com/sirupsen/logrus"
	"github.com/xanzy/go-gitlab"

	"github.com/zc2638/review-bot/pkg/scm"
	"github.com/zc2638/review-bot/pkg/util"
)

func NewComment(si scm.Interface, pid string, ref string, prID int) (*Comment, error) {
	pr, err := si.GetPullRequest(pid, prID)
	if err != nil {
		return nil, err
	}
	cfg, err := si.GetReviewConfig(pid, ref)
	if err != nil {
		return nil, err
	}
	return &Comment{
		si:   si,
		cfg:  cfg,
		pr:   pr,
		pid:  pid,
		prID: prID,
	}, nil
}

type Comment struct {
	si  scm.Interface
	cfg *scm.ReviewConfig
	pr  *scm.PullRequest

	pid  string
	prID int
}

func (e *Comment) newMerge() *Merge {
	return &Merge{
		si:   e.si,
		cfg:  e.cfg,
		pr:   e.pr,
		pid:  e.pid,
		prID: e.prID,
	}
}

func (e *Comment) Process(event *gitlab.MergeCommentEvent) error {
	// 获取评论内容
	note := event.ObjectAttributes.Note

	var addLabels, removeLabels []string

	// 匹配admin标签
	if _, ok := util.InStringSlice(e.cfg.Approvers, event.User.Username); ok {
		label := scm.AdminSet.FuzzyLabelWithKey("FORCE-MERGE", note)
		if label != nil {
			logrus.Infof("Run force merge by %s on PR(%v) in Repo(%s)", event.User.Username, e.prID, e.pid)
			return e.newMerge().merge(event.MergeRequest.LastCommit.ID)
		}

		label = scm.AdminSet.FuzzyLabelWithKey("APPROVE", note)
		if label != nil {
			addLabels = append(addLabels, label.Name)
		}
	}
	if _, ok := util.InStringSlice(e.cfg.Reviewers, event.User.Username); ok {
		label := scm.AdminSet.FuzzyLabelWithKey("LGTM", note)
		if label != nil {
			addLabels = append(addLabels, label.Name)
		}
	}

	adds, removes := dealCommonLabel(e.cfg, event.Project.PathWithNamespace, note)
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
			if err := e.si.MergePullRequestApprove(
				event.Project.PathWithNamespace,
				event.MergeRequest.IID,
				false,
			); err != nil {
				logrus.Debugf("Remove approve failed: %v", err)
			}
			break
		}
	}

	opt := &scm.UpdatePullRequest{
		Labels:       filterLabels(e.pr.Labels, addLabels, removeLabels),
		AddLabels:    addLabels,
		RemoveLabels: removeLabels,
		AssigneeID:   event.MergeRequest.AssigneeID,
		AssigneeIDs:  event.MergeRequest.AssigneeIDs,
	}
	return e.si.UpdatePullRequest(event.Project.PathWithNamespace, event.MergeRequest.IID, opt)
}
