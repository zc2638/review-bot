/**
 * Created by zc on 2021/1/15.
 */
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
