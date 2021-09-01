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
package scm

type Config struct {
	Type   string `json:"type"`
	Host   string `json:"host"`
	Token  string `json:"token"`
	Secret string `json:"secret"`
}

type Interface interface {
	ListProjectMembers(pid string) ([]ProjectMember, error)
	ListLabels(pid string) ([]Label, error)
	CreateLabel(pid string, label *Label) error
	CreatePullRequestComment(pid string, prID int, comment string) error
	GetPullRequest(pid string, prID int) (*PullRequest, error)
	UpdatePullRequest(pid string, prID int, data *UpdatePullRequest) error
	UpdateBuildStatus(pid, sha string, state BuildState) error
	MergePullRequest(pid string, prID int, data *MergePullRequest) error
	GetReviewConfig(pid, ref string) (*ReviewConfig, error)
}

type BuildState = string

const (
	BuildStatePending  BuildState = "pending"
	BuildStateCreated  BuildState = "created"
	BuildStateRunning  BuildState = "running"
	BuildStateSuccess  BuildState = "success"
	BuildStateFailed   BuildState = "failed"
	BuildStateCanceled BuildState = "canceled"
	BuildStateSkipped  BuildState = "skipped"
	BuildStateManual   BuildState = "manual"
)
