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

const ReviewConfigFileName = "review.yml"

type ReviewConfig struct {
	Reviewers []string `json:"reviewers"`
	Approvers []string `json:"approvers"`
}

type Label struct {
	Order       string    `json:"order"`
	Type        LabelType `json:"type"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	TextColor   string    `json:"text_color"`
	Description string    `json:"description"`
}

type PullRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	TargetBranch string   `json:"target_branch"`
	AssigneeID   int      `json:"assignee_id"`
	AssigneeIDs  []int    `json:"assignee_i_ds"`
	Labels       []string `json:"labels"`
	AddLabels    []string `json:"add_labels"`
	RemoveLabels []string `json:"remove_labels"`
}
