# BENZHI_README

## 项目说明
- 项目：benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec
- 项目用途：完整实现古树养护任务从建档、现状采集、可解释风险评估、版本化方案审核、现场实施到验收关闭的单流程 Web 工作台，具备修订冲突、请求幂等、本地原子快照、追加审计日志和有界 HTTP 自检。
- Go 工具链：`golang:1.23`
- 前端工具链：原生 HTML、CSS 和 JavaScript，由 Go 服务直接提供

## 项目描述
- 项目名称：古树养护闭环核验台
- 项目介绍：面向城市古树养护管理人员的单流程 Web 应用，将树木现状采集、风险评估、养护方案审核、现场实施与验收关闭串成可追溯闭环。
- 项目概述：面向城市古树养护管理人员的单流程 Web 应用，将树木现状采集、风险评估、养护方案审核、现场实施与验收关闭串成可追溯闭环。
- 核心工作流：负责人创建古树养护任务并记录现状，系统完成风险评估后形成养护方案，技术人员审核批准，现场人员提交实施证据，负责人验收并将任务关闭。
- 对外接口：由 Go HTTP 服务提供原生 HTML、CSS 和 JavaScript 的浏览器工作台，包含任务列表、现状采集、风险评估、方案审核、实施记录和验收视图；监听地址支持 -addr，默认使用 127.0.0.1:19081，且 PORT 为端口号时绑定 127.0.0.1:<PORT>。

## 标准构建、运行和测试命令
进入容器后执行：
cd '/app' && GOTOOLCHAIN=local go build ./...

cd /app && GOTOOLCHAIN=local go run ./cmd/server -addr=127.0.0.1:19081 -selfcheck

cd '/app' && GOTOOLCHAIN=local go test ./...

## Docker 构建和进入容器
chmod +x build_benzhi_docker.sh

./build_benzhi_docker.sh benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec-amd64 linux/amd64

./build_benzhi_docker.sh benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec-arm64 linux/arm64

docker run -it benzhi-project-66f1180d-b7bd-4c55-af98-4b319cdd75ec-amd64:latest

## 题目验证命令
1. 预期退出码 0：`go test ./...`
2. 预期退出码 0：`go run ./cmd/server -addr=127.0.0.1:19081 -selfcheck`
