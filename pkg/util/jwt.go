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
	"crypto/md5"
	"errors"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

type JwtClaims struct {
	Auth *JwtAuthInfo `json:",inline"`
	jwt.RegisteredClaims
}

type JwtAuthInfo struct {
	Slug      string    `json:"slug"` // namespace/name
	CreatedAt time.Time `json:"created_at"`
	Signature string    `json:"signature"`
}

func (j *JwtAuthInfo) BuildSign(secret string) string {
	data := j.Slug + strconv.FormatInt(j.CreatedAt.Unix(), 10)
	hash := md5.New()
	_, _ = hash.Write([]byte(data))
	sign := hash.Sum([]byte(secret))
	return string(sign)
}

func (j *JwtAuthInfo) CheckSign(secret string) bool {
	sign := j.BuildSign(secret)
	return j.Signature == sign
}

func JwtCreate(claims JwtClaims, secret string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func JwtParse(tokenStr string, secret string) (*JwtClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JwtClaims)
	if !ok || !token.Valid {
		return nil, errors.New("token is not valid")
	}
	return claims, nil
}
