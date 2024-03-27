<h1 align="center" style="border-bottom: none">
    <b>
        <a href="https://docs.openim.io">tools</a><br>
    </b>
</h1>
<h3 align="center" style="border-bottom: none">
      ⭐️  OpenIM tools.  ⭐️ <br>
<h3>


<p align=center>
<a href="https://goreportcard.com/report/github.com/openimsdk/tools"><img src="https://goreportcard.com/badge/github.com/openimsdk/tools" alt="A+"></a>
<a href="https://github.com/openimsdk/tools/issues?q=is%3Aissue+is%3Aopen+sort%3Aupdated-desc+label%3A%22good+first+issue%22"><img src="https://img.shields.io/github/issues/openimsdk/tools/good%20first%20issue?logo=%22github%22" alt="good first"></a>
<a href="https://github.com/openimsdk/tools"><img src="https://img.shields.io/github/stars/openimsdk/tools.svg?style=flat&logo=github&colorB=deeppink&label=stars"></a>
<a href="https://join.slack.com/t/openimsdk/shared_invite/zt-22720d66b-o_FvKxMTGXtcnnnHiMqe9Q"><img src="https://img.shields.io/badge/Slack-100%2B-blueviolet?logo=slack&amp;logoColor=white"></a>
<a href="https://github.com/openimsdk/tools/blob/main/LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-green"></a>
<a href="https://pkg.go.dev/github.com/openimsdk/tools"><img src="https://img.shields.io/badge/Language-Go-blue.svg"></a>
</p>

<p align="center">
    <a href="./README.md"><b>English</b></a> •
    <a href="./README_zh-CN.md"><b>中文</b></a>
</p>

</p>

----

# 项目工具集说明文档

本项目包含了一系列适用于 OpenIM 通用工具和库以及一些其他的项目提供工具支持，旨在支持高效方案开发。以下是各个模块的功能介绍：

## a2r

- `api2rpc.go`：API 到 RPC 的转换工具，用于将 HTTP API 请求转换为 RPC 调用。

## apiresp

- `format.go`, `gin.go`, `http.go`, `resp.go`：处理 API 响应的格式化、封装及发送，支持不同的Web框架。

## checker

- `check.go`：提供服务健康检查和依赖项验证功能。

## config

- `config.go`, `config_parser.go`, `config_source.go`, `manager.go`, `path.go`：配置管理模块，支持配置的解析、加载及动态更新。
- `validation`：提供配置验证的工具和库。

## db

- `mongo`, `pagination`, `redis`, `tx.go`：数据库操作相关的工具，包括 MongoDB、Redis 的支持和事务管理。

## discovery

- `discovery_register.go`：服务发现与注册功能。
- `zookeeper`：基于 Zookeeper 的服务发现实现。

## env

- `env.go`, `env_test.go`：环境变量管理工具，包括加载和解析环境变量。

## errs

- `code.go`, `coderr.go`, `error.go`, `predefine.go`, `relation.go`：错误码管理和自定义错误类型。

## field

- `file.go`, `path.go`：处理文件操作和路径生成的实用工具。
- 相关的测试文件。

## log

- `color.go`, `encoder.go`, `logger.go`, `sql_logger.go`, `zap.go`, `zk_logger.go`：日志管理模块，支持多种日志格式和输出。

> [!IMPORTANT]
> 关于 OpenIM log 可以阅读 [https://github.com/openimsdk/open-im-server/blob/main/docs/contrib/logging.md](https://github.com/openimsdk/open-im-server/blob/main/docs/contrib/logging.md)

## mcontext

- `ctx.go`：上下文管理工具，用于在中间件和服务之间传递请求相关的信息。

## mq

- `kafka`：基于 Kafka 的消息队列支持。

## mw

- `gin.go`, `intercept_chain.go`, `rpc_client_interceptor.go`, `rpc_server_interceptor.go`：中间件和拦截器，用于处理请求的预处理和后处理。
- `specialerror`：特殊错误处理模块。

## tokenverify

- `jwt_token.go`, `jwt_token_test.go`：JWT 令牌验证和测试。

## utils

utils 包含多个工具库，如 `encoding`, `encrypt`, `httputil`, `jsonutil`, `network`, `splitter`, `stringutil`, `timeutil`：提供各种常用功能，例如加密、编码、网络操作等。

#### encoding

- `base64.go` & `base64_test.go`：提供 Base64 编码和解码的实用函数，及其单元测试。

#### encrypt

- `encryption.go` & `encryption_test.go`：包含加密和解密的功能实现，支持常见的加密算法，以及相关的单元测试。

#### goassist

- `jsonutils.go` & `jsonutils_test.go`：提供 JSON 数据处理的实用函数，如解析和生成 JSON，以及相关的单元测试。

#### httputil

- `http_client.go` & `http_client_test.go`：封装 HTTP 客户端的操作，提供便捷的 HTTP 请求发送方法，及其单元测试。

#### jsonutil

- `interface.go`, `json.go` & `json_test.go`：专注于 JSON 数据处理，包括更高级的 JSON 操作和定制化的 JSON 解析方法，及其单元测试。

#### network

- `ip.go` & `ip_test.go`：提供网络相关的实用函数，如 IP 地址的解析和验证，以及相关的单元测试。

#### splitter

- `splitter.go` & `splitter_test.go`：提供字符串分割的工具，支持多种分割策略和复杂的分割场景，及其单元测试。

#### stringutil

- `strings.go` & `strings_test.go`：包含一系列字符串操作的实用函数，如字符串的修改、搜索、比较等，及其单元测试。

#### timeutil

- `time_format.go` & `time_format_test.go`：提供与时间相关的实用函数，包括时间格式的解析和格式化，以及相关的单元测试。

## version

- `base.go`, `doc.go`, `types.go`, `version.go`：版本管理工具，用于定义和管理项目版本信息。
