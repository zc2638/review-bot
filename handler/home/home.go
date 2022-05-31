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

package home

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/zc2638/review-bot/pkg/scm"

	"github.com/pkg/errors"

	"github.com/zc2638/review-bot/global"

	"github.com/zc2638/review-bot/pkg/util"

	"github.com/pkgms/go/ctr"
)

func index() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctr.OK(w, "Hello Bot!")
	}
}

func secret() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		namespace := r.URL.Query().Get("namespace")
		name := r.URL.Query().Get("name")
		if strings.TrimSpace(namespace) == "" ||
			strings.TrimSpace(name) == "" {
			ctr.BadRequest(w, errors.New("namespace or name required"))
			return
		}
		slug := path.Join(namespace, name)
		authInfo := &util.JwtAuthInfo{
			Slug:      slug,
			CreatedAt: time.Now(),
		}
		authInfo.Signature = authInfo.BuildSign(global.Cfg().SCM.Secret)
		token, err := util.JwtCreate(util.JwtClaims{
			Auth: authInfo,
		}, global.JWTSecret)
		if err != nil {
			ctr.InternalError(w, err)
			return
		}
		ctr.OK(w, token)
	}
}

func download(staticPath string) http.HandlerFunc {
	if staticPath == "" {
		staticPath = "public"
	}
	return func(w http.ResponseWriter, r *http.Request) {
		dir := r.URL.Query().Get("type")
		if dir == "" {
			dir = "gitlab"
		}
		fileName := dir + ".zip"
		zipFilePath := filepath.Join(staticPath, fileName)
		_, err := os.Stat(zipFilePath)
		if err != nil && !os.IsNotExist(err) {
			ctr.InternalError(w, err)
			return
		}
		var data []byte
		if os.IsNotExist(err) {
			data, err = buildZIP(staticPath, dir)
			if err != nil {
				ctr.InternalError(w, err)
				return
			}
			if err := ioutil.WriteFile(zipFilePath, data, os.ModePerm); err != nil {
				ctr.InternalError(w, err)
				return
			}
		} else {
			data, err = ioutil.ReadFile(zipFilePath)
			if err != nil {
				ctr.InternalError(w, err)
				return
			}
		}

		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("content-disposition", "attachment; filename=\""+fileName+"\"")
		_, _ = w.Write(data)
	}
}

func buildZIP(static, dir string) ([]byte, error) {
	src := filepath.Join(static, dir)
	var buf bytes.Buffer
	archive := zip.NewWriter(&buf)
	archive.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.BestCompression)
	})
	if err := filepath.Walk(src, func(path string, info os.FileInfo, _ error) error {
		// 如果是源路径，跳过
		if path == src {
			return nil
		}
		// 获取：文件头信息
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = strings.TrimLeft(strings.TrimPrefix(path, src+`\`), static)
		if info.IsDir() {
			header.Name += `/`
		}
		// 创建：压缩包头部信息
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			file, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			_, _ = writer.Write(file)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	if err := archive.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func commandHelp() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var list string
		for _, v := range scm.AdminSet.Labels() {
			list += `<tr align="center">
                <td>` + v.Order + `</td>
                <td><span class="label-item" style="background: ` + v.Color + `;">` + v.Name + `</span></td>
                <td>` + v.Description + `</td>
            </tr>` + "\n"
		}
		for _, v := range scm.AddSet.Labels() {
			list += `<tr align="center">
                <td>` + v.Order + `</td>
                <td><span class="label-item" style="background: ` + v.Color + `;">` + v.Name + `</span></td>
                <td>` + v.Description + `</td>
            </tr>` + "\n"
		}
		for _, v := range scm.RemoveSet.Labels() {
			removeOrder := strings.TrimPrefix(v.Order, "/")
			removeOrder = "/remove-" + removeOrder

			list += `<tr align="center">
                <td>` + v.Order + `</td>
                <td></td>
                <td>` + v.Description + `</td>
            </tr>` + "\n"
		}
		for _, v := range scm.CustomSet.Labels() {
			order := strings.TrimPrefix(v.Order, "/")
			order = "/【remove-】" + order

			list += `<tr align="center">
                <td>` + order + `</td>
                <td><span class="label-item" style="background: ` + v.Color + `;">` + v.Name + `</span></td>
                <td>` + v.Description + `</td>
            </tr>` + "\n"
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		data := generateTemplate(list)
		ctr.Str(w, data)
	}
}

func generateTemplate(list string) string {
	return `<!doctype html>
<html xmlns=http://www.w3.org/1999/xhtml>
<meta charset=utf-8>
<title>Review Bot Command Help</title>
<head>
<style type="text/css">
.list td {
	text-align: left;
}
.label-item{
    padding: 2px 10px;
    border-radius: 4px;
}
</style>
</head>
<body>
<div class="content">
    <div class="list">
        <table border="1" align="center" cellspacing="0" cellpadding="6">
            <caption>Command Help</caption>

            <thead>
            <tr align="center">
                <th>Order</th>
                <th>Label Name</th>
                <th>Description</th>
            </tr>
            </thead>

            <tbody>
            ` + list + `
            </tbody>
        </table>
    </div>
</div>

<script>
    let host = "http://" + window.location.host;

    function createRequest(host, method, data, callback) {
        let xhr = window.XMLHttpRequest ? new window.XMLHttpRequest() :
            new window.ActiveXObject('Microsoft.XMLHTTP');
        xhr.open(method || "GET", host, false);
        xhr.onreadystatechange = function () {
            //判断请求状态是否是已经完成
            if (xhr.readyState === 4) {
                //判断服务器是否返回成功200,304
                if (xhr.status >= 200 && xhr.status <= 300 || xhr.status === 304) {
                    //接收xhr的数据
                    callback(xhr.responseText);
                }
            }
        };
        xhr.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
        // xhr.setRequestHeader("Canhe-Control", "no-cache");//阻止浏览器读取缓存
        xhr.send(data);
    }
</script>
</body>
</html>`
}
