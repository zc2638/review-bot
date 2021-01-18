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

type LabelKind int

const (
	LabelNone LabelKind = iota
	LabelLGTM
	LabelApprove
	LabelWIP
	LabelWIPCancel
	LabelHold
	LabelHoldCancel
	LabelKindMissing
	LabelKindFeature
	LabelKindBugfix
	LabelKindStyle
	LabelKindDocs
	LabelKindRefactor
	LabelKindPerf
	LabelKindTest
	LabelKindCI
)

type LabelType int

const (
	LabelTypeKind LabelType = iota
	LabelTypeAdmin
	LabelTypeNormal
	LabelTypeRemove
)

const DoNotMerge = "do-not-merge"

var Labels = map[LabelKind]Label{
	LabelLGTM: {
		Order:       "/lgtm",
		Type:        LabelTypeAdmin,
		Name:        "lgtm",
		Color:       "#5CB85C",
		Description: "标识同意合并",
	},
	LabelApprove: {
		Order:       "/approve",
		Type:        LabelTypeAdmin,
		Name:        "approved",
		Color:       "#5CB85C",
		Description: "标识审批通过",
	},
	LabelWIP: {
		Order:       "/wip",
		Type:        LabelTypeNormal,
		Name:        DoNotMerge + "/work-in-progress",
		Color:       "#FF0000",
		Description: "标识开发中，不要合并",
	},
	LabelHold: {
		Order:       "/hold",
		Type:        LabelTypeNormal,
		Name:        DoNotMerge + "/hold",
		Color:       "#FF0000",
		Description: "标识不要合并",
	},
	LabelWIPCancel: {
		Order:       "/wip cancel",
		Type:        LabelTypeRemove,
		Name:        DoNotMerge + "/work-in-progress",
		Color:       "#FF0000",
		Description: "标识取消开发中状态",
	},
	LabelHoldCancel: {
		Order:       "/hold cancel",
		Type:        LabelTypeRemove,
		Name:        DoNotMerge + "/hold",
		Color:       "#FF0000",
		Description: "标识取消hold状态",
	},
	LabelKindMissing: {
		Order:       "/kind missing",
		Type:        LabelTypeKind,
		Name:        DoNotMerge + "/kind-missing",
		Color:       "#FF0000",
		Description: "标识缺少分类，不要合并",
	},
	LabelKindFeature: {
		Order:       "/kind feature",
		Type:        LabelTypeKind,
		Name:        "kind/feature",
		Color:       "#428BCA",
		Description: "分类：新功能",
	},
	LabelKindBugfix: {
		Order:       "/kind bugfix",
		Type:        LabelTypeKind,
		Name:        "kind/bugfix",
		Color:       "#FF0000",
		Description: "分类：bug处理",
	},
	LabelKindStyle: {
		Order:       "/kind style",
		Type:        LabelTypeKind,
		Name:        "kind/style",
		Color:       "#43CD80",
		Description: "分类：样式",
	},
	LabelKindDocs: {
		Order:       "/kind docs",
		Type:        LabelTypeKind,
		Name:        "kind/docs",
		Color:       "#CAFF70",
		Description: "分类：文档",
	},
	LabelKindRefactor: {
		Order:       "/kind refactor",
		Type:        LabelTypeKind,
		Name:        "kind/refactor",
		Color:       "#FF1493",
		Description: "分类：重构",
	},
	LabelKindPerf: {
		Order:       "/kind perf",
		Type:        LabelTypeKind,
		Name:        "kind/perf",
		Color:       "#A020F0",
		Description: "分类：性能",
	},
	LabelKindTest: {
		Order:       "/kind test",
		Type:        LabelTypeKind,
		Name:        "kind/test",
		Color:       "#8B0000",
		Description: "分类：测试",
	},
	LabelKindCI: {
		Order:       "/kind ci",
		Type:        LabelTypeKind,
		Name:        "kind/ci",
		Color:       "#9AC0CD",
		Description: "分类：CICD",
	},
}
