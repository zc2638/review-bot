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

import "strings"

const DoNotMerge = "do-not-merge"

type Set int

const (
	_ Set = iota
	AdminSet
	AddSet
	RemoveSet
	CustomSet
)

func getLabelSet(s Set) map[string]Label {
	switch s {
	case AdminSet:
		return adminSet
	case AddSet:
		return addSet
	case RemoveSet:
		return removeSet
	case CustomSet:
		return customSet
	default:
		return customSet
	}
}

func (s Set) Labels() []Label {
	set := getLabelSet(s)
	labels := make([]Label, 0, len(set))
	for _, v := range set {
		labels = append(labels, v)
	}
	return labels
}

func (s Set) Label(name string) *Label {
	set := getLabelSet(s)
	for _, v := range set {
		if name == v.Name {
			return &v
		}
	}
	return nil
}

func (s Set) LabelByKey(key string) *Label {
	set := getLabelSet(s)
	for k, v := range set {
		if key == k {
			return &v
		}
	}
	return nil
}

func (s Set) FuzzyLabels(content string) []Label {
	set := getLabelSet(s)
	var labels []Label
	for _, v := range set {
		if strings.Contains(content, v.Order) {
			labels = append(labels, v)
		}
	}
	return labels
}

func (s Set) FuzzyLabelWithKey(key, content string) *Label {
	set := getLabelSet(s)
	for k, v := range set {
		if k == key && strings.Contains(content, v.Order) {
			return &v
		}
	}
	return nil
}

func (s Set) FuzzyLabelsWithPrefix(prefix, content string) []Label {
	set := getLabelSet(s)
	var labels []Label
	for _, v := range set {
		order := strings.TrimPrefix(v.Order, "/")
		order = "/" + prefix + "-" + order
		if strings.Contains(content, order) {
			labels = append(labels, v)
		}
	}
	return labels
}

var autoSet = map[string]Label{
	"KIND": {
		Order:       "/kind missing",
		Name:        DoNotMerge + "/kind-missing",
		Color:       "#FF0000",
		Description: "标识缺少分类，不要合并",
	},
}

var adminSet = map[string]Label{
	"LGTM": {
		Order:       "/lgtm",
		Name:        "lgtm",
		Color:       "#5CB85C",
		Description: "标识同意合并",
	},
	"APPROVE": {
		Order:       "/approve",
		Name:        "approved",
		Color:       "#5CB85C",
		Description: "标识审批通过",
	},
	"FORCE-MERGE": {
		Order:       "/force-merge",
		Name:        "force-merge",
		Color:       "#5CB85C",
		Description: "标识强制自动合并",
	},
}

var addSet = map[string]Label{
	"WIP": {
		Order:       "/wip",
		Name:        DoNotMerge + "/work-in-progress",
		Color:       "#FF0000",
		Description: "标识开发中，不要合并",
	},
	"HOLD": {
		Order:       "/hold",
		Name:        DoNotMerge + "/hold",
		Color:       "#FF0000",
		Description: "标识不要合并",
	},
}

var removeSet = map[string]Label{
	"WIP": {
		Order:       "/remove-wip",
		Name:        DoNotMerge + "/work-in-progress",
		Color:       "#FF0000",
		Description: "取消开发中状态",
	},
	"HOLD": {
		Order:       "/remove-hold",
		Name:        DoNotMerge + "/hold",
		Color:       "#FF0000",
		Description: "取消hold状态",
	},
	"LGTM": {
		Order:       "/remove-lgtm",
		Name:        "lgtm",
		Color:       "#5CB85C",
		Description: "取消同意合并",
	},
	"APPROVE": {
		Order:       "/remove-approve",
		Name:        "approved",
		Color:       "#5CB85C",
		Description: "取消审批通过",
	},
}

var customSet = map[string]Label{
	"MERGE": {
		Order:       "/kind merge",
		Name:        "kind/merge",
		Short:       "merge",
		Color:       "#00F5FF",
		Description: "分类：不压缩合并",
	},
	"FEATURE": {
		Order:       "/kind feature",
		Name:        "kind/feature",
		Short:       "feat",
		Color:       "#428BCA",
		Description: "分类：新功能",
	},
	"BUGFIX": {
		Order:       "/kind bug",
		Name:        "kind/bugfix",
		Short:       "fix",
		Color:       "#F0AD4E",
		Description: "分类：bug处理",
	},
	"STYLE": {
		Order:       "/kind style",
		Name:        "kind/style",
		Short:       "style",
		Color:       "#43CD80",
		Description: "分类：样式",
	},
	"DOCS": {
		Order:       "/kind docs",
		Name:        "kind/docs",
		Short:       "docs",
		Color:       "#CAFF70",
		Description: "分类：文档",
	},
	"REFACTOR": {
		Order:       "/kind refactor",
		Name:        "kind/refactor",
		Short:       "refactor",
		Color:       "#FF1493",
		Description: "分类：重构",
	},
	"PERF": {
		Order:       "/kind perf",
		Name:        "kind/perf",
		Short:       "perf",
		Color:       "#A020F0",
		Description: "分类：性能",
	},
	"TEST": {
		Order:       "/kind test",
		Name:        "kind/test",
		Short:       "test",
		Color:       "#8B0000",
		Description: "分类：测试",
	},
	"CI": {
		Order:       "/kind ci",
		Name:        "kind/ci",
		Short:       "ci",
		Color:       "#9AC0CD",
		Description: "分类：CICD",
	},
	"CLEANUP": {
		Order:       "/kind cleanup",
		Name:        "kind/cleanup",
		Short:       "cleanup",
		Color:       "#33a3dc",
		Description: "分类：整理",
	},
}
