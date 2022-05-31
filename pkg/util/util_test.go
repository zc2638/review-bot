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

package util

import (
	"reflect"
	"testing"
)

func TestInStringSlice(t *testing.T) {
	type args struct {
		ss  []string
		str string
	}
	tests := []struct {
		name       string
		args       args
		wantIndex  int
		wantExists bool
	}{
		{
			name: "notExists",
			args: args{
				ss:  []string{"a"},
				str: "b",
			},
			wantIndex:  -1,
			wantExists: false,
		},
		{
			name: "exists",
			args: args{
				ss:  []string{"a", "b", "c"},
				str: "b",
			},
			wantIndex:  1,
			wantExists: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIndex, gotExists := InStringSlice(tt.args.ss, tt.args.str)
			if gotIndex != tt.wantIndex {
				t.Errorf("InStringSlice() gotIndex = %v, want %v", gotIndex, tt.wantIndex)
			}
			if gotExists != tt.wantExists {
				t.Errorf("InStringSlice() gotExists = %v, want %v", gotExists, tt.wantExists)
			}
		})
	}
}

func TestRemoveStringSlice(t *testing.T) {
	type args struct {
		ss  []string
		str string
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "notExists",
			args: args{
				ss:  []string{"a", "b", "c"},
				str: "d",
			},
			want: []string{"a", "b", "c"},
		},
		{
			name: "exists",
			args: args{
				ss:  []string{"a", "b", "c"},
				str: "b",
			},
			want: []string{"a", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveStringSlice(tt.args.ss, tt.args.str); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveStringSlice() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRemoveStringSliceByIndex(t *testing.T) {
	type args struct {
		ss    []string
		index int
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{
			name: "zero",
			args: args{
				ss:    []string{"a", "b", "c"},
				index: 0,
			},
			want: []string{"b", "c"},
		},
		{
			name: "exists",
			args: args{
				ss:    []string{"a", "b", "c"},
				index: 1,
			},
			want: []string{"a", "c"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RemoveStringSliceByIndex(tt.args.ss, tt.args.index); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RemoveStringSliceByIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}
