# review-bot

gitlab review-bot

## Quick start

### Config

change config file in `config/config.yaml`

```
scm:
  host: https://gitlab.com
  token: <your-private-token>
  secret: <your-webhook-secret>
```

| 配置项 | 环境变量 | 描述 |
| :----: | :----: | :----: |
| scm.host | BOT_SCM_HOST | 地址 |
| scm.token | BOT_SCM_TOKEN | 私有token |
| scm.secret | BOT_SCM_SECRET | webhook的访问密钥 |

### Run Local

```
go run github.com/zc2638/review-bot/cmd -c config/config.yaml
```

### Run Docker

```
docker build -t review-bot -f build/Dockerfile .
docker run -d -p 2640:2640 -e BOT_SCM_HOST=https://gitlab.com -e BOT_SCM_TOKEN=<your-private-token> -e BOT_SCM_SECRET=<your-webhook-secret> review-bot
```

## Note

### pull request template

- Download at url `/download?type=gitlab`
- Unzip and move the directory `gitlab` to `.gitlab` in your project
- Modify configuration file `review.yml`

### setting

- the `review-bot` user must have your project permissions
- webhook must set sufficient permissions(e.g. `Comments`、`Confidential Comments`、`Pull request events`)