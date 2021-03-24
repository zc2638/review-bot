# review-bot

![Main CI WorkFlow](https://github.com/zc2638/review-bot/workflows/Main%20CI%20WorkFlow/badge.svg)

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

### Generate Webhook Secret

`GET /secret?namespace=repo&name=test`

### Pull Request Template

- Download at url `GET /download?type=gitlab`
- Unzip and move the directory `gitlab` to `.gitlab` in your project
- Modify configuration file `review.yml`

### Setting

- the `review-bot` user must have your project permissions
- webhook must set sufficient permissions(e.g. `Comments`、`Confidential Comments`、`Pull request events`)

### Custom Config
the configuration file path is in `.gitlab/review.yml`
```yaml
# can use /lgtm
reviewers:
  - reviewer1
  - reviewer2

# can use /approve
approvers:
  - approver1
  - approver2
pullrequest:
  # The merge information is mainly based on the title of PR
  # otherwise it is mainly based on the content of <!-- title --><!-- end title --> in PR description template
  squash_with_title: true
```