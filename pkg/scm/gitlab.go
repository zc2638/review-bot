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

import (
	"path"

	"github.com/xanzy/go-gitlab"
	"gopkg.in/yaml.v2"
)

type gitlabClient struct {
	config *Config
	client *gitlab.Client
}

func NewGitlabClient(cfg *Config) (Interface, error) {
	client, err := gitlab.NewClient(
		cfg.Token,
		gitlab.WithBaseURL(cfg.Host),
	)
	if err != nil {
		return nil, err
	}
	return &gitlabClient{
		config: cfg,
		client: client,
	}, nil
}

func (s *gitlabClient) ListLabels(pid string) ([]Label, error) {
	var result []Label
	var page int
	for {
		page++
		opt := &gitlab.ListLabelsOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		labels, _, err := s.client.Labels.ListLabels(pid, opt)
		if err != nil {
			return nil, err
		}
		for _, v := range labels {
			result = append(result, Label{
				Name:        v.Name,
				Color:       v.Color,
				TextColor:   v.TextColor,
				Description: v.Description,
			})
		}
		if len(labels) < 100 {
			break
		}
	}
	return result, nil
}

func (s *gitlabClient) CreateLabel(pid string, label *Label) error {
	opt := &gitlab.CreateLabelOptions{
		Name:        &label.Name,
		Color:       &label.Color,
		Description: &label.Description,
	}
	_, _, err := s.client.Labels.CreateLabel(pid, opt)
	return err
}

func (s *gitlabClient) UpdatePullRequest(pid string, prID int, data *PullRequest) error {
	opt := &gitlab.UpdateMergeRequestOptions{
		AssigneeIDs:  data.AssigneeIDs,
		Labels:       data.Labels,
		AddLabels:    data.AddLabels,
		RemoveLabels: data.RemoveLabels,
	}
	if data.Title != "" {
		opt.Title = &data.Title
	}
	if data.Description != "" {
		opt.Description = &data.Description
	}
	if data.TargetBranch != "" {
		opt.TargetBranch = &data.TargetBranch
	}
	if data.AssigneeID > 0 {
		opt.AssigneeID = &data.AssigneeID
	}
	_, _, err := s.client.MergeRequests.UpdateMergeRequest(pid, prID, opt)
	return err
}

func (s *gitlabClient) MergePullRequest(pid string, prID int) error {
	opt := &gitlab.AcceptMergeRequestOptions{}
	_, _, err := s.client.MergeRequests.AcceptMergeRequest(pid, prID, opt)
	return err
}

func (s *gitlabClient) GetReviewConfig(pid, ref string) (*ReviewConfig, error) {
	opt := &gitlab.GetRawFileOptions{
		Ref: &ref,
	}
	data, _, err := s.client.RepositoryFiles.GetRawFile(pid, path.Join(".gitlab", ReviewConfigFileName), opt)
	if err != nil {
		return nil, err
	}
	var config ReviewConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, err
}

func (s *gitlabClient) ListProjectMembers(pid string) ([]ProjectMember, error) {
	var result []ProjectMember
	var page int
	for {
		page++
		opt := &gitlab.ListProjectMembersOptions{
			ListOptions: gitlab.ListOptions{
				Page:    page,
				PerPage: 100,
			},
		}
		members, _, err := s.client.ProjectMembers.ListAllProjectMembers(pid, opt)
		if err != nil {
			return nil, err
		}
		for _, member := range members {
			result = append(result, ProjectMember{
				ID:        member.ID,
				Username:  member.Username,
				Email:     member.Email,
				Name:      member.Name,
				AvatarURL: member.AvatarURL,
			})
		}
		if len(members) < 100 {
			break
		}
	}
	return result, nil
}
