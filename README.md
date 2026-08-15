# Personal Assistant

一个前后端分离的个人助手管理后台起点：前端使用 Vue 3，后端采用 go-admin 风格的 Gin + GORM + JWT 分层结构，并在 `internal/goutils` 内置了基于 [usust/goUtils](https://github.com/usust/goUtils) 调整的配置和日志工具代码。

## 技术栈

- 前端：Vue 3、TypeScript、Vite、Vue Router、Pinia、Element Plus、Axios
- 后端：Go、Gin、GORM、SQLite、JWT、bcrypt、内置配置及日志工具（Viper + Zap）
- 默认账号：`admin`
- 默认密码：`123456`

> 默认账号仅用于本地开发。部署前请修改管理员密码和配置中的 `jwt_secret`。

## 目录

```text
.
├── frontend/                    # Vue 3 前端
│   └── src/
│       ├── api/                 # HTTP 请求封装
│       ├── layouts/             # 后台布局
│       ├── router/              # 路由及登录守卫
│       ├── stores/              # Pinia 状态
│       └── views/               # 登录、工作台、设置页
└── backend/                     # Gin 后端
    └── internal/
        ├── config/              # YAML 配置定义及启动校验
        ├── database/            # GORM 初始化及数据迁移
        ├── goutils/             # 本地配置加载、Bootstrap 和 Zap 日志代码
        ├── handler/             # API 控制器
        ├── middleware/          # JWT、CORS 中间件
        ├── model/               # 数据模型
        ├── router/              # 路由注册
        └── service/             # 业务服务
```

## 运行命令

需要 Node.js 20+、npm 和 Go 1.26.2+。

以下命令均在项目根目录 `PersonalAssistant` 下执行。

### 1. 首次安装

```bash
# 安装前端依赖
cd frontend
npm install
cp .env.example .env.local

# 下载后端依赖并创建 YAML 配置
cd ../backend
go mod download
cp config.example.yaml config.yaml

# 返回项目根目录
cd ..
```

### 2. 启动后端

打开第一个终端：

```bash
cd backend
go run .
```

后端默认读取当前目录下的 `config.yaml`，监听 `http://localhost:16101`。首次启动会在 `backend/data/` 创建 SQLite 数据库，并在 `backend/log/` 生成按等级拆分和轮转的 Zap 日志。

部署时可用 `EXPORT_CONFIG_FILE` 指定其他配置文件，无需复制为默认文件名：

```bash
cd backend
EXPORT_CONFIG_FILE=/path/to/production.yaml go run .
```

配置在启动阶段完成校验；端口、运行模式、数据库路径、JWT 配置或跨域来源无效时，服务会直接拒绝启动。

### 3. 启动前端

打开第二个终端：

```bash
cd frontend
npm run dev
```

访问 `http://localhost:16100`。开发服务器会把 `/api` 请求代理到 `http://localhost:16101` 的后端。

### 4. 部署 Web 服务

生产环境不需要常驻 Vite。前端 workflow 会把构建结果发布到
`/srv/www/personal-assistant/current`，宿主机上的独立 Web 服务负责在 `16100`
端口提供静态页面。Docker 中监听 `8000` 的入口 Nginx 是另一个服务，可以按需
反向代理到宿主机的 `16100`。

首次部署或 Web 服务配置发生变化时，在服务器执行：

```bash
sudo cp /srv/www/personal-assistant/deploy/personal-assistant-web.service \
  /etc/systemd/system/personal-assistant-web.service
sudo systemctl daemon-reload
sudo systemctl enable --now personal-assistant-web.service
```

随后只需访问：

```text
http://192.168.31.6:16100
```

可以用下面的地址检查宿主机 Web 服务：

```bash
curl -I http://192.168.31.6:16100/
```

该服务只负责前端文件，不依赖后端 `16101`。如果入口 Nginx 需要通过 `8000`
提供同一个页面，应将它的上游设置为宿主机 `16100`。

### 5. 通过 CI 部署后端 Docker 容器

后端 workflow 会在代码推送到 `main` 且 `backend/**` 发生变化时，将源码同步到
`192.168.31.5`，在目标 Docker 服务器原生构建镜像，并将服务发布到 `16101`。
部署包含容器健康检查；新版本启动失败时会恢复上一个容器。

首次使用前，需要在运行 self-hosted GitHub Runner 的机器上准备 SSH 私钥
`/Users/lyu/.ssh/personal_assistant_backend`，并配置 `dok@192.168.31.5` 免密登录和
`known_hosts`。CI 会通过 `-i` 显式使用该密钥。Docker 服务器上的 `dok` 用户必须有
Docker 权限。

在 GitHub 仓库中只需配置：

- Secret `BACKEND_CONFIG_YAML`：正式后端配置，格式参考 `backend/config.example.yaml`，
  且必须包含 `port: "16101"`，生产环境务必替换 `jwt_secret`。

Docker 服务器上的部署用户需要有 Docker 权限，并能写入
`/srv/docker/personal-assistant`。部署成功后可检查：

```bash
curl http://192.168.31.5:16101/api/health
```

正式前端构建会直接使用 `http://192.168.31.5:16101/api`，因此还需要确保服务器
防火墙允许 Web 用户访问 `16101`。

### 使用 Makefile 快速启动

完成首次安装后，也可以在项目根目录运行：

```bash
# 启动后端
make dev-backend

# 启动前端，需要另开一个终端
make dev-frontend
```

### 停止服务

在运行服务的终端按 `Ctrl + C`。

## API

| 方法 | 地址 | 鉴权 | 说明 |
| --- | --- | --- | --- |
| GET | `/api/health` | 否 | 服务健康检查 |
| GET | `/api/captcha` | 否 | 获取一次性登录验证码，验证码两分钟内有效 |
| POST | `/api/login` | 否 | 用户登录并获取 JWT |
| GET | `/api/task-lists` | 是 | 查询当前用户的任务清单 |
| POST | `/api/task-lists` | 是 | 创建任务清单 |
| PATCH | `/api/task-lists/:taskListId` | 是 | 更新任务清单 |
| DELETE | `/api/task-lists/:taskListId` | 是 | 删除任务清单及其中任务 |
| GET | `/api/tasks` | 是 | 查询主任务、子任务和服务端汇总进度 |
| POST | `/api/tasks` | 是 | 创建主任务或带工作量设置的子任务 |
| PATCH | `/api/tasks/:taskId` | 是 | 更新无子任务主任务的完成状态 |
| PATCH | `/api/tasks/:taskId/progress` | 是 | 增加、编辑、完成或重置子任务进度 |
| DELETE | `/api/tasks/:taskId` | 是 | 删除任务；主任务可显式级联删除子任务 |

任务清单的创建接口支持 `name`、`remark`、`color` 和 `icon` 字段，其中 `remark` 为最长 2000 个字符的可选备注。更新接口支持部分更新这些字段，传入空字符串可清空备注。

子任务使用“进度总量、当前完成量、默认增量”记录执行情况。主任务的进度由后端按工作量加权计算：

```text
主任务进度 = 所有子任务完成量之和 / 所有子任务总量之和 × 100%
```

例如两个子任务分别为 `60/100` 和 `10/20`，主任务进度为 `70/120 = 58.33%`。进度数值最多支持两位小数；增量更新由后端原子处理，主任务状态随汇总结果自动切换为待办、进行中或已完成。

登录前先请求验证码：

```bash
curl http://localhost:16101/api/captcha
```

响应中的 `data.captchaId` 和 `data.image` 分别用于关联验证码及展示验证码图片。识别图片内容后，将验证码 ID 和输入内容随登录请求一并提交：

```bash
curl -X POST http://localhost:16101/api/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"123456","captchaId":"验证码ID","captchaCode":"图片中的验证码"}'
```

验证码只能校验一次，无论答案是否正确，登录尝试后都需要重新获取。

## 测试与构建命令

```bash
# 在项目根目录运行全部测试
make test

# 构建前端和后端
make build
```

也可以分别运行：

```bash
# 前端类型检查和生产构建
cd frontend
npm run type-check
npm run build

# 后端测试、静态检查和编译
cd ../backend
go test ./...
go vet ./...
go build ./...
```

## 后续扩展

新增业务时，可以按 `model → service → handler → router` 添加后端模块，再在前端的 `api → store → view` 中接入。若要完整引入 go-admin 的 Casbin RBAC、代码生成、菜单权限和 Swagger，可继续在当前分层上扩展。
