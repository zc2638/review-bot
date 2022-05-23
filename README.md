# review-bot

![Main CI WorkFlow](https://github.com/zc2638/review-bot/workflows/Main%20CI%20WorkFlow/badge.svg)

gitlab review-bot

## Preconditions
- Gitlab version 13+

## Quick start

### Config

change config file in `config/config.yaml`

```
scm:
  host: https://gitlab.com
  token: <your-private-token>
  secret: <your-webhook-secret>
```

| Configuration Item | Environment Variable |          Description           |
|:------------------:|:--------------------:|:------------------------------:|
|      scm.host      |     BOT_SCM_HOST     | source code management address |
|     scm.token      |    BOT_SCM_TOKEN     |         private token          |
|     scm.secret     |    BOT_SCM_SECRET    |         webhook secret         |

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

### Api Document
`GET /swagger-ui`

### Generate Webhook Secret

`GET /secret?namespace=repo&name=test`

### Pull Request Template

- Download at url `GET /download?type=gitlab`
- Unzip and move the directory `gitlab` to `.gitlab` in your project
- Modify configuration file `review.yml`

### Setting

- add webhook to associated project, URL is `http://<your-host-address>/webhook`
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
custom_labels:
    # Operation instructions in comments
  - order: /kind cleanup
    # Label name associated with the instruction
    name: kind/cleanup
    # Automatically add prefix for merged submission information
    short: cleanup
    # Label background color
    color: #33a3dc
    # Label description
    description: "kind: cleanup code"

  - order: /area scheduler
    name: area/scheduler
    color: #96582a
    description: "area: scheduler service code area"
```