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

package scm

import "time"

const ReviewConfigFileName = "review.yml"

type ReviewConfig struct {
	Reviewers    []string          `json:"reviewers" yaml:"reviewers"`
	Approvers    []string          `json:"approvers" yaml:"approvers"`
	CustomLabels []Label           `json:"custom_labels" yaml:"custom_labels"`
	PRConfig     PullRequestConfig `json:"pullrequest" yaml:"pullrequest"`
}

type PullRequestConfig struct {
	// 合并信息以PR的标题为主，否则以PR描述模板内的 <!-- title -->内容<!-- end title--> 内容为主
	SquashWithTitle bool `json:"squash_with_title" yaml:"squash_with_title"`
}

type Label struct {
	Order       string `json:"order"`
	Name        string `json:"name"`
	Short       string `json:"short"`
	Color       string `json:"color"`
	TextColor   string `json:"text_color"`
	Description string `json:"description"`
}

type ProjectMember struct {
	ID        int    `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
}

type PullRequest struct {
	ID                        int        `json:"id"`
	IID                       int        `json:"iid"`
	TargetBranch              string     `json:"target_branch"`
	SourceBranch              string     `json:"source_branch"`
	ProjectID                 int        `json:"project_id"`
	Title                     string     `json:"title"`
	State                     string     `json:"state"`
	CreatedAt                 *time.Time `json:"created_at"`
	UpdatedAt                 *time.Time `json:"updated_at"`
	SourceProjectID           int        `json:"source_project_id"`
	TargetProjectID           int        `json:"target_project_id"`
	Labels                    []string   `json:"labels"`
	Description               string     `json:"description"`
	WorkInProgress            bool       `json:"work_in_progress"`
	MergeWhenPipelineSucceeds bool       `json:"merge_when_pipeline_succeeds"`
	ShouldRemoveSourceBranch  bool       `json:"should_remove_source_branch"`
	ForceRemoveSourceBranch   bool       `json:"force_remove_source_branch"`
	Squash                    bool       `json:"squash"`
}

type UpdatePullRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	TargetBranch string   `json:"target_branch"`
	AssigneeID   int      `json:"assignee_id"`
	AssigneeIDs  []int    `json:"assignee_ids"`
	Labels       []string `json:"labels"`
	AddLabels    []string `json:"add_labels"`
	RemoveLabels []string `json:"remove_labels"`
}

type MergePullRequest struct {
	SquashCommitMessage       string `json:"squash_commit_message"`
	Squash                    bool   `json:"squash"`
	ShouldRemoveSourceBranch  bool   `json:"should_remove_source_branch"`
	MergeWhenPipelineSucceeds bool   `json:"merge_when_pipeline_succeeds"`
}
