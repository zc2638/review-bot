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

var cache = Cache{}

func Cached() Cache {
	return cache
}

type Cache map[string]struct{}

func (c Cache) Add(key string) {
	c[key] = struct{}{}
}

func (c Cache) Remove(key string) {
	delete(c, key)
}

func (c Cache) IsExist(key string) bool {
	_, ok := c[key]
	return ok
}

var userCache = UserCache{}

func UserCached() UserCache {
	return userCache
}

type UserCache map[string]ProjectMember

func (c UserCache) Add(name string, member ProjectMember) {
	c[name] = member
}

func (c UserCache) Remove(name string) {
	delete(c, name)
}

func (c UserCache) Get(name string) (ProjectMember, bool) {
	member, ok := c[name]
	return member, ok
}
