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
	"strings"

	"github.com/99nil/go/sets"

	"github.com/sirupsen/logrus"

	"github.com/zc2638/review-bot/global"
	"github.com/zc2638/review-bot/pkg/scm"
)

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

func filterLabels(exists []string, adds []string, removes []string) []string {
	s := sets.NewString(exists...)
	s.Add(adds...)
	s.Remove(removes...)
	return s.List()
}
