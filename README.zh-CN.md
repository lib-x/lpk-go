# lzc-toolkit-go

[English](README.md) | [简体中文](README.zh-CN.md)

`lzc-toolkit-go` 是一个用于构建、读取、检查、签名、部署和发布 LazyCat LPK 的 Go 库。它参考 `@lazycatcloud/lzc-cli` `2.0.9` 的源码和服务端协议实现，但不会复制 lzc-cli 的命令行交互。

如果你第一次接触 LazyCat，可以先把它理解为一组可组合的底层能力：你的程序决定什么时候构建、用哪种镜像后端、凭据从哪里来，以及结果如何进入自己的 CI/CD。

## 这个库解决什么问题

常见使用场景包括：

- 在 Go 程序中把一个 LazyCat 项目构建成 `.lpk` 文件。
- 从 `io.Reader` 解析别人上传的 LPK，读取 Manifest 或安全解压。
- 用本机 Docker 构建 LPK 内嵌镜像。
- 通过 DebugBridge 在远端构建镜像，并把应用部署到懒猫设备预览。
- 请求 LazyCat 开发者平台把公共镜像复制到 `registry.lazycat.cloud`。
- 在 CI/CD 中登录、检查、上传并提交 LPK。
- 根据使用场景选择普通 lint 或 LazyCat 官方提交规则。

这是一个库，不是 CLI。它不会弹出登录框、询问是否覆盖文件、接管终端或自动安装 SSH 密钥。调用方负责配置、交互、日志和重试策略。

## 安装

项目要求 Go `1.25` 或更高版本：

```bash
go get github.com/lib-x/lzc-toolkit-go
```

基础包不会引入或执行 Docker、SSH、gRPC、App Store 客户端等可选能力。只有显式导入相应子包时，程序才需要对应依赖。

## 先认识几个概念

| 概念 | 含义 |
|------|------|
| LPK | LazyCat 应用安装包，文件扩展名通常是 `.lpk`。包内可以包含应用元数据、Manifest、静态内容、镜像层和签名。 |
| Manifest | 描述应用如何运行的 YAML，包括入口服务、镜像、域名、路由、环境变量和其他服务。项目文件通常叫 `lzc-manifest.yml`，进入 LPK 后叫 `manifest.yml`。 |
| `package.yml` | LPK v2 中的静态应用信息，包括包名、版本、名称、描述和多语言信息。 |
| `lzc-build.yml` | 项目构建配置。它告诉构建器去哪里找 Manifest、内容目录、图标、镜像和资源导出。它本身不会进入最终应用运行配置。 |
| LPK v1 / v2 | 两种兼容布局。v1 把包信息放在 Manifest 中；v2 使用独立的 `package.yml`，并支持 OCI 镜像布局和资源导出。新项目通常使用 v2。 |
| OCI 镜像布局 | LPK v2 保存容器镜像元数据和内嵌层的格式。`images.lock` 记录镜像及其层，`images/` 保存需要随 LPK 分发的 blob。 |
| `ImageBuilder` | `build` 包故意不绑定 Docker。带镜像项目必须显式提供本地 Docker 或远端 DebugBridge 镜像构建器。 |
| 普通 lint | 检查 LPK 和 Manifest 的通用有效性，适用于本地工具和非官方分发。 |
| 官方 lint | `lint.WithOfficial()` 开启的 LazyCat 开发者平台提交规则，比如官方 Registry、图标和多语言要求。 |
| 开发者平台 token | 用于复制镜像、创建应用、Testflight 和提交 LPK。它和设备端 ShellAPI 凭据、SSH 密钥不是同一种认证。 |
| ShellAPI | LazyCat 客户端在本机提供的发现接口，可以取得默认微服、登录 UID 和客户端 `dev.id`。 |
| DebugBridge | LazyCat Developer Tools 提供的远端协议，用于构建镜像、安装 LPK、控制应用、执行命令和读取日志。 |
| `project` | 建立在 DebugBridge 之上的应用生命周期服务，提供部署、启动、停止、等待、日志、命令和文件复制。 |

## 一次开发会经过哪些组件

不带镜像的项目可以直接构建 LPK。带镜像的项目需要先选择本地或远端镜像后端：

```text
本地项目
  -> build.Build / build.BuildFile
  -> LPK
     -> inspect / lint / signature        本地检查、解析和签名
     -> project + DebugBridge              部署到懒猫设备预览
     -> appstore.Publish                   提交到开发者平台

带镜像项目
  -> dockerlocal 或 buildpack
  -> OCI 镜像布局
  -> build.Build
  -> LPK
```

镜像复制是另一条独立链路：

```text
公共 Registry 镜像
  -> LazyCat 开发者平台 CopyImage API
  -> registry.lazycat.cloud/... 镜像地址
```

这条链路发生在开发者平台服务端，本机不需要安装 Docker。

## 按任务选择包

| 任务 | 建议导入的包 |
|------|--------------|
| 读写、解压 LPK | `lpk` |
| 解析和预处理 Manifest | `manifest` |
| 检查本地源码项目 | `project` |
| 查看 LPK 摘要 | `inspect` |
| 构建项目 | `build` |
| 检查 OCI 布局 | `oci` |
| 用本机 Docker 构建镜像 | `image/dockerlocal` |
| 用 DebugBridge 远端构建镜像 | `image/buildpack`、`remote/blobcache` |
| 账号登录和 token | `auth`、`auth/tokenfile` |
| 镜像复制和 LPK 提交 | `appstore` |
| 读取官方公开应用目录 | `appstore/official` |
| 查询喵喵私有商店 | `appstore/private` |
| 发现本机 LazyCat 客户端 | `remote/shellapi` |
| SSH 和 DebugBridge | `remote/ssh`、`remote/debugbridge` |
| 应用部署和生命周期 | `project` |
| 源码同步 | `project/rsync` |
| 构建、部署、同步、启动编排 | `workflow/project` |
| 签名和验签 | `signature` |

## 完整例子

以下例子按外部依赖从少到多排列。第一个例子只需要 Go，第二个需要本机 Docker，第三个需要懒猫设备和 SSH，第四、第五个需要开发者平台 token。

| 场景 | 本机 Docker | 开发者平台 token | 懒猫设备和 SSH |
|------|-------------|------------------|-----------------|
| 构建和解析无镜像 LPK | 不需要 | 不需要 | 不需要 |
| 本机构建内嵌镜像 | 需要 | 不需要 | 不需要 |
| 通过 DebugBridge 远端构建和预览 | 不需要，本地不执行 Docker | 不需要 | 需要 |
| 复制镜像到 LazyCat Registry | 不需要，平台服务端复制 | 需要 | 不需要 |
| 提交 LPK 到开发者平台 | 不需要 | 需要 | 不需要 |

### 例子一：本地构建、解析和检查 LPK

这个例子不需要 Docker、LazyCat 账号或设备。它会把一个静态页面项目构建成 `hello.lpk`，再读取包信息和有效 Manifest。

项目目录：

```text
hello-lpk/
├── lzc-build.yml
├── lzc-manifest.yml
├── package.yml
└── content/
    └── html/
        └── index.html
```

`hello-lpk/lzc-build.yml`：

```yaml
manifest: ./lzc-manifest.yml
contentdir: ./content
```

`hello-lpk/package.yml`：

```yaml
package: cloud.lazycat.apps.hello
version: 0.1.0
name: Hello LPK
locales:
  zh:
    name: 你好 LPK
  en:
    name: Hello LPK
```

`hello-lpk/lzc-manifest.yml`：

```yaml
application:
  subdomain: hello
  routes:
    - /=file:///lzcapp/pkg/content/html
```

`hello-lpk/content/html/index.html` 可以是任意静态页面：

```html
<!doctype html>
<html lang="zh-CN">
  <meta charset="utf-8">
  <title>Hello LPK</title>
  <h1>Hello LazyCat</h1>
</html>
```

构建并读取 LPK：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lib-x/lzc-toolkit-go/build"
    "github.com/lib-x/lzc-toolkit-go/inspect"
    "github.com/lib-x/lzc-toolkit-go/lpk"
)

func main() {
    ctx := context.Background()

    result, err := build.BuildFile(ctx, "hello.lpk", build.Request{
        Root:    "./hello-lpk",
        ForceV2: true,
        Strict:  true,
    })
    if err != nil {
        log.Fatal(err)
    }

    info, err := inspect.File(ctx, "hello.lpk")
    if err != nil {
        log.Fatal(err)
    }

    reader, err := lpk.OpenFile(ctx, "hello.lpk")
    if err != nil {
        log.Fatal(err)
    }
    defer reader.Close()

    effective, err := reader.EffectiveManifest(ctx)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("built %s %s as %s\n", result.Package, result.Version, result.Layout)
    fmt.Printf("archive: %s, size: %d bytes\n", info.Format, info.Size)
    fmt.Printf("subdomain: %s\n", effective.Manifest.Application.Subdomain)
}
```

运行后会生成 `hello.lpk`。`BuildFile` 使用原子替换，失败时不会留下半个输出文件。`ForceV2` 要求项目使用 v2 布局，因此包名和版本放在 `package.yml` 中。

如果输出已经是一个 `io.Writer`，可以改用 `build.Build`：

```go
var output bytes.Buffer
result, err := build.Build(ctx, &output, build.Request{
    Root:    "./hello-lpk",
    ForceV2: true,
})
```

`build.Build` 不会关闭 `output`。同样，`lpk.Open`、`inspect.Stream` 和发布 API 也不会关闭调用方传入的 Reader。

常见错误：

- `INVALID_CONFIG`：先检查 `lzc-build.yml` 中的路径、`package.yml` 的包名和版本。
- `INVALID_MANIFEST`：检查 YAML 结构和 `application.subdomain`。
- 找不到 Manifest：确认 `manifest` 指向的文件存在，默认文件名是 `lzc-manifest.yml`。
- 构建脚本没有运行：这是默认行为。只有设置 `RunBuildScript: true` 后，`buildscript` 才会执行。

### 例子二：使用本机 Docker 构建带镜像的 LPK

这个场景需要本机安装 Docker，并满足以下条件：

- `docker` 命令可以执行。
- Docker daemon 已启动。
- `docker buildx` 可用。

只导入 `build`、`lpk`、`oci` 或 `image` 不会调用 Docker。只有显式传入 `dockerlocal.New(...)` 时，SDK 才执行 Docker CLI。

项目增加 `app/Dockerfile`，并在 Manifest 中用 `embed:app` 引用镜像别名：

```text
docker-hello/
├── app/
│   └── Dockerfile
├── lzc-build.yml
├── lzc-manifest.yml
└── package.yml
```

`docker-hello/lzc-build.yml`：

```yaml
manifest: ./lzc-manifest.yml
images:
  app:
    builder: local
    context: ./app
    dockerfile: ./app/Dockerfile
```

`docker-hello/package.yml`：

```yaml
package: cloud.lazycat.apps.docker-hello
version: 0.1.0
name: Docker Hello
locales:
  zh:
    name: Docker 示例
  en:
    name: Docker Hello
```

`docker-hello/lzc-manifest.yml`：

```yaml
application:
  subdomain: docker-hello
  image: embed:app
```

`docker-hello/app/Dockerfile`：

```dockerfile
FROM alpine:3.20
CMD ["sh", "-c", "while true; do echo hello; sleep 30; done"]
```

Go 代码：

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/lib-x/lzc-toolkit-go/build"
    "github.com/lib-x/lzc-toolkit-go/image/dockerlocal"
)

func main() {
    ctx := context.Background()
    result, err := build.BuildFile(ctx, "docker-hello.lpk", build.Request{
        Root:         "./docker-hello",
        ForceV2:      true,
        ImageBuilder: dockerlocal.New(nil),
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("images: %d\n", result.ImageCount)
    fmt.Printf("resolved: %#v\n", result.ResolvedImages)
}
```

`dockerlocal.New(nil)` 使用默认 Docker CLI 后端，目标平台是 `linux/amd64`。构建其他平台时显式修改：

```go
builder := dockerlocal.New(nil, dockerlocal.WithPlatform("linux/arm64"))
```

构建器会运行与 lzc-cli 2.0.9 兼容的 `docker buildx build --load`、`docker image inspect` 和 `docker image save` 流程，然后把结果转换成 LPK v2 的 OCI 布局。Docker 29 通过 `Id` 返回 Manifest descriptor 时，构建器会从保存的归档中还原 config JSON 摘要；已经 gzip 压缩的 layer 会保留原始压缩字节。最终包中的 `images.lock` 描述镜像和层，`images/` 只保存需要内嵌的 blob。Manifest 中的 `embed:app` 会附加解析后的镜像摘要。

常见错误：

- `INCOMPATIBLE_BACKEND`：项目声明了镜像，但没有传 `ImageBuilder`，或者配置要求远端构建却传了本地构建器。
- `COMMAND_FAILED`：Docker 命令、Dockerfile 或基础镜像拉取失败。
- `INTEGRITY_MISMATCH`：生成的镜像布局、摘要或 blob 不一致。

### 例子三：部署到懒猫设备预览

这里的“预览”是指：源码保留在本地编辑，镜像和服务在懒猫或 build-remote 主机侧运行，然后从 LazyCat 客户端打开应用。

当前 DebugBridge 不会把你电脑上监听的 `localhost:3000` 直接反向暴露给懒猫。要预览本机进程，需要另行配置反向隧道或代理，这不是当前 SDK 和 lzc-cli 2.0.9 的 DebugBridge 能力。

前置条件：

- 本机安装并登录了 LazyCat 客户端，ShellAPI 配置文件可用。
- 已选择默认微服。
- 目标设备安装并启用了 Developer Tools。
- 你可以通过 SSH 访问 build-remote 主机。
- SSH 公钥授权已经配置好，SDK 不负责安装密钥。
- 远端 DebugBridge 版本满足相应功能要求。

以下环境变量由你的程序或 CI 配置，不是 SDK 固定变量：

```bash
export LZC_BUILD_REMOTE_USER=developer
export LZC_BUILD_REMOTE_ADDR=build-host.example:22
```

完整链路包括 ShellAPI 发现、SSH 传输、远端镜像构建、LPK 安装和启动：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "path/filepath"

    "github.com/lib-x/lzc-toolkit-go/build"
    "github.com/lib-x/lzc-toolkit-go/image/buildpack"
    "github.com/lib-x/lzc-toolkit-go/project"
    "github.com/lib-x/lzc-toolkit-go/remote/blobcache"
    "github.com/lib-x/lzc-toolkit-go/remote/debugbridge"
    "github.com/lib-x/lzc-toolkit-go/remote/shellapi"
    "github.com/lib-x/lzc-toolkit-go/remote/ssh"
)

func main() {
    if err := run(); err != nil {
        log.Fatal(err)
    }
}

func run() error {
    ctx := context.Background()
    projectDir := "./docker-hello"
    packagePath := filepath.Join(os.TempDir(), "docker-hello-dev.lpk")

    config, err := shellapi.LoadConfig(ctx, shellapi.ConfigOptions{})
    if err != nil {
        return err
    }
    shell, err := shellapi.New(ctx, shellapi.Options{Config: config})
    if err != nil {
        return err
    }
    defer shell.Close()

    box, err := shell.DefaultBox(ctx)
    if err != nil {
        return err
    }
    devID, err := shell.ClientID(ctx)
    if err != nil {
        return err
    }

    target, err := ssh.ParseTarget(
        os.Getenv("LZC_BUILD_REMOTE_USER"),
        os.Getenv("LZC_BUILD_REMOTE_ADDR"),
    )
    if err != nil {
        return err
    }
    runner := ssh.New(ssh.Options{Target: target})
    bridge := debugbridge.New(
        runner,
        runner.BridgeCommand,
        debugbridge.WithUID(box.LoginUser),
        debugbridge.WithHostCommand(runner.HostCommand),
    )

    remoteImages := buildpack.New(bridge, blobcache.New(projectDir))
    built, err := build.BuildFile(ctx, packagePath, build.Request{
        Root:         projectDir,
        ConfigFile:   "lzc-build.dev.yml",
        ForceV2:      true,
        ImageBuilder: remoteImages,
    })
    if err != nil {
        return err
    }

    packageFile, err := os.Open(packagePath)
    if err != nil {
        return err
    }
    defer packageFile.Close()

    projects, err := project.New(project.Options{Backend: bridge})
    if err != nil {
        return err
    }
    if _, err := projects.Deploy(ctx, project.DeployRequest{
        Package:   packageFile,
        PackageID: built.Package,
        DevID:     devID,
    }); err != nil {
        return err
    }

    running, err := projects.Start(ctx, project.StartRequest{
        AppID:        built.Package,
        LocalVersion: built.Version,
    })
    if err != nil {
        return err
    }
    fmt.Printf("running: %s, domain: %s\n", running.AppID, running.Domain)

    follow := false
    _, err = projects.Logs(ctx, project.LogRequest{
        AppID:   built.Package,
        Follow:  &follow,
        Stdout:  os.Stdout,
        Stderr:  os.Stderr,
    })
    return err
}
```

这个例子假设项目有 `lzc-build.dev.yml`，并且其中的镜像使用远端构建器。远端构建器会根据 Dockerfile 的 `COPY` 和 `ADD` 生成确定性 context，支持 `.dockerignore` 和 blob 缓存。

开发时还可以组合：

- `project/rsync.Sync`：把本地源码同步到 `/lzcapp/cache/project-mirror`。
- `project.Exec`：在应用容器中运行命令或重启开发进程。
- `project.CopyTo`：把本地目录通过 TAR 流复制进容器。
- `project.Logs`：读取一次日志或持续跟随日志。

SDK 不内置文件监听器。watch 模式应由调用方选择文件监听库，在文件变化后重新调用 `Sync`，再按项目需要执行重启命令。

### 例子四：把公共镜像复制到 LazyCat Registry

这个操作不需要本机安装 Docker。

需要准备的只有：

- 可以访问 LazyCat 开发者平台的网络。
- 有效的开发者平台 token。
- 开发者平台能够拉取的源镜像地址。

镜像复制由开发者平台服务端完成，不会执行本机 Docker，也不会调用懒猫设备上的 Docker、ShellAPI 或 SSH。

先把 token 放入环境变量：

```bash
export LZC_CLI_TOKEN='your-token'
```

不要把真实 token 写入源码或提交到 Git。

完整例子：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/lib-x/lzc-toolkit-go/appstore"
    "github.com/lib-x/lzc-toolkit-go/auth"
)

func main() {
    ctx := context.Background()
    client := appstore.New(appstore.Options{
        Token: auth.EnvironmentToken{},
    })

    result, err := client.CopyImage(ctx, appstore.CopyImageRequest{
        Image:    "docker.io/library/alpine:3.20",
        Platform: "amd64",
        Timeout:  20 * time.Minute,
        OnProgress: func(progress appstore.CopyProgress) {
            fmt.Printf("finished=%v layers=%d\n", progress.Finished, len(progress.Layers))
        },
    })
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(result.LazyCatImage)
}
```

返回的 `LazyCatImage` 是可写入应用配置的 `registry.lazycat.cloud/...` 地址。返回值还保留源镜像、目标平台和最终分层进度，CI/CD 不需要解析终端日志。

当前 lzc-cli 2.0.9 和本 SDK 的 `CopyImageRequest` 没有源 Registry 用户名、密码或 token 字段。如果源镜像需要认证，调用方不能通过这个接口额外传入私有 Registry 凭据，必须先确认开发者平台能够拉取该镜像。

### 例子五：登录并提交 LPK

旧版开发者平台会话认证有两种方式。

开发者平台还提供独立的 PAT API。PAT 与旧版 lzc-cli 会话 token 必须保持
不同语义：`appstore.New` 使用 `/api/v3/developer`、`X-User-Token` 和
`userToken` Cookie；`appstore.NewPAT` 使用 `/sdk/v3/developer` 和
`X-API-Token`：

```go
client, err := appstore.NewPAT(appstore.Options{
    Token: auth.StaticToken(os.Getenv("LZC_API_TOKEN")),
})
```

`NewPAT` 会禁用重定向，也不会把 PAT 转发到其他来源。SDK 不会复制或同步
PAT 与会话 token 的值。

可用 `WaitingReviewVersion` 查询指定应用当前是否有审核中版本。服务端返回
404 时，方法返回 `found == false`，不会把它当作错误：

```go
version, found, err := client.WaitingReviewVersion(ctx, "cloud.lazycat.example")
if err != nil {
    return err
}
if found {
    fmt.Printf("%s 正在审核中\n", version)
}
```

#### 方式一：账号密码换 token

`auth.Client.Login` 会向 `https://account.lazycat.cloud/api/login/signin` 提交账号和密码，并返回 `Session.Token`。密码不会保存；配置了 `TokenStore` 时只保存 token。

```go
package main

import (
    "context"
    "log"
    "os"
    "path/filepath"

    "github.com/lib-x/lzc-toolkit-go/auth"
    "github.com/lib-x/lzc-toolkit-go/auth/tokenfile"
)

func main() {
    ctx := context.Background()
    home, err := os.UserHomeDir()
    if err != nil {
        log.Fatal(err)
    }
    tokenPath := filepath.Join(home, ".config", "lazycat", "box-config.json")
    account := auth.NewClient(auth.ClientOptions{
        Store: tokenfile.Store{Path: tokenPath},
    })
    session, err := account.Login(ctx, auth.Credentials{
        Username: os.Getenv("LZC_USERNAME"),
        Password: os.Getenv("LZC_PASSWORD"),
    })
    if err != nil {
        log.Fatal(err)
    }
    if err := account.Validate(ctx, session.Token); err != nil {
        log.Fatal(err)
    }
}
```

不建议在 CI 中长期保存账号密码。更合适的方式是在可信环境登录一次，把得到的 token 保存到 CI Secret。

#### 方式二：直接提供 token

SDK 支持三种常用来源：

- `auth.StaticToken`：调用方直接传值。
- `auth.EnvironmentToken`：默认读取 `LZC_CLI_TOKEN`。
- `auth.StoreProvider`：从显式 `TokenStore` 读取。

如果本机已经通过 lzc-cli 登录，lzc-cli 2.0.9 的读取顺序是：

1. 如果存在 `LZC_CLI_TOKEN`，使用环境变量。
2. 否则读取 `~/.config/lazycat/box-config.json` 的 `token` 字段。

`lzc-cli config get token` 可以打印当前生效的 token，但不要在 CI 日志中执行。Go SDK 不会隐式读取用户目录，应显式配置 `tokenfile.Store{Path: tokenPath}`。

提交一个已经满足官方 lint 的 LPK：

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/lib-x/lzc-toolkit-go/appstore"
    "github.com/lib-x/lzc-toolkit-go/auth"
)

func main() {
    ctx := context.Background()
    packageFile, err := os.Open("application.lpk")
    if err != nil {
        log.Fatal(err)
    }
    defer packageFile.Close()

    client := appstore.New(appstore.Options{
        Token: auth.EnvironmentToken{},
    })
    published, err := client.Publish(ctx, appstore.PublishRequest{
        Package:  packageFile,
        FileName: "application.lpk",
        Changelogs: map[string]string{
            "zh": "修复启动流程。",
            "en": "Fix startup handling.",
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("submitted %s %s\n", published.Upload.Package, published.Upload.Version)
}
```

`Publish` 会执行以下步骤：

1. 在配置的大小限制内暂存输入流。
2. 解析并安全解压 LPK。
3. 执行 `lint.WithOfficial()` 检查。
4. 检查开发者平台中是否已经存在该应用。
5. 上传 LPK，并验证服务端返回的包名。
6. 提交带多语言更新日志的审核请求。

`Publish` 不会关闭 `packageFile`。如果允许自动创建不存在的应用，需要显式设置 `CreateIfMissing: true` 并提供 `Application`，库不会交互式询问。

准备提交官方平台时，通常还要满足：

- 镜像使用 `registry.lazycat.cloud` 或合规的内嵌镜像。
- 存在不超过 200 KiB 的 PNG `icon.png`。
- `package.yml` 提供 `locales`。
- 版本符合 SemVer。
- 包中不包含 devshell。

### 无需 Token 读取应用商店

官方目录客户端是完全匿名的，不接收也不构造账号 Token：

```go
client := official.New(official.Options{})
application, err := client.Application(ctx, "wx.clawbot.lazycat.app.mediasaber")
if err != nil {
    return err
}
downloadURL, err := client.ApplicationDownloadURL(application)
if err != nil {
    return err
}
fmt.Println(application.Version.Name, downloadURL)
```

导入 `github.com/lib-x/lzc-toolkit-go/appstore/official`。该包还提供
`Homepage`、`Categories`、`Kinds`、`More`、`DownloadRanking`、
`DeveloperRanking` 和 `VersionChangelog`。测试或使用镜像时，可以通过
`official.Options` 覆盖元数据和 LPK 下载地址。

喵喵私有商店使用独立子包，并可通过六位私有分组码开放应用：

```go
client, err := privatestore.New(privatestore.Options{
    BaseURL:    "https://store.example.com",
    GroupCodes: []string{"LATE23"},
})
if err != nil {
    return err
}
latest, err := client.LatestVersion(ctx, privatestore.LatestVersionRequest{
    PackageID: "community.lazycat.group-app",
})
if err != nil {
    return err
}
fmt.Println(latest.LatestVersion.Version, latest.LatestVersion.DownloadURL)
```

将 `github.com/lib-x/lzc-toolkit-go/appstore/private` 导入为
`privatestore`。分组码属于 bearer 凭据；客户端会统一转成大写、去重，
并默认通过 `X-Group-Codes` 请求头发送，也可显式选择查询参数或两者同时
发送。最新版本接口同样不需要账号 Token。应用不存在、未发布、没有已
批准版本或没有分组访问权限时，都会返回 `lpkgo.ErrNotFound`。

## API 与高级能力参考

### LPK Reader 和 Writer

写入任意 `io.Writer`：

```go
result, err := lpk.Write(ctx, dst, lpk.WriteRequest{
    Layout: lpk.LayoutV2,
    Files:  os.DirFS("package-root"),
    Strict: true,
})
```

从顺序流打开。实现会在受限制的临时存储中暂存输入：

```go
reader, err := lpk.Open(ctx, src)
```

已有随机访问能力时避免暂存整个输入：

```go
reader, err := lpk.OpenReaderAt(ctx, src, size)
```

`Reader` 可以列出条目、读取 Manifest 和 `package.yml`、打开单个文件、安全解压，以及合并得到有效 Manifest。归档解析支持可配置的大小、路径和条目数量限制。

`project.Restore` 可以进一步把发布包转换为可再次 `build.Build` 的项目目录；
`project.RestoreWithOptions(..., project.RestoreOptions{PreferUpstreamImages: true})`
会使用 Manifest 中明确写出的 `# upstream:` 镜像注释。懒猫 Registry 的 digest
本身不包含可逆的上游版本，未提供注释时不会猜测镜像地址或版本。

#### 从商店 LPK 解包

公共商店 API 返回的 `Version.PackagePath` 可以先通过
`official.Client.ApplicationDownloadURL` 解析为下载地址，再交给 `lpk.Open`
和 `Reader.Extract`：

```go
request, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
if err != nil {
    return err
}
response, err := http.DefaultClient.Do(request)
if err != nil {
    return err
}
defer response.Body.Close()
reader, err := lpk.Open(ctx, response.Body)
if err != nil {
    return err
}
defer reader.Close()
if err := reader.Extract(ctx, "restored-project"); err != nil {
    return err
}
```

解包目录包含 LPK 中实际发布的 `manifest.yml`、`package.yml`、静态文件、
OCI 镜像层和资源导出，可以继续用 `inspect` 或 `lint` 检查。它是发布包的还原，
不等同于恢复原始源码项目：未打进 LPK 的源码、Dockerfile、`lzc-build.yml`
和构建脚本无法恢复；内嵌镜像也不会自动转换成 Dockerfile。

### Lint

检查解压后的 LPK 根目录：

```go
warnings, err := lint.Package(ctx, os.DirFS("package-root"))
```

准备提交 LazyCat 官方平台时：

```go
warnings, err := lint.Package(
    ctx,
    os.DirFS("package-root"),
    lint.WithOfficial(),
)
```

官方规则默认关闭，因为 Registry、图标、多语言、SemVer 和 devshell 限制属于平台提交偏好，不代表一个 LPK 在所有场景下都无法安装。

### ShellAPI、SSH 和认证边界

| 操作 | 使用的凭据 |
|------|------------|
| 发现默认微服和 `dev.id` | LazyCat 客户端的 `shellapi_addr`、`shellapi_cred` |
| build-remote、DebugBridge、部署 | 目标主机 SSH 授权和 LazyCat UID |
| 镜像复制、应用创建、Testflight、LPK 提交 | 开发者平台 token |
| APK Shell 触发接口 | 与 lzc-cli 2.0.9 一样，不需要 App Store token |

ShellAPI 配置的默认位置：

- Linux：`~/.config/hportal-client/`
- macOS：`~/Library/Application Support/hportal-client/`
- Windows：`~/AppData/Roaming/hportal-client/`

### 检查本地项目

`project.Inspect` 可以读取现有项目，但不会运行 `buildscript`、构建镜像、
写入 LPK，也不会连接懒猫设备：

```go
inspection, err := project.Inspect(ctx, project.InspectRequest{
    Root:       ".",
    ConfigFile: "lzc-build.yml",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("%s %s %s services=%d images=%d templated=%t\n",
    inspection.Package.Package,
    inspection.Package.Version,
    inspection.Kind,
    len(inspection.Services),
    len(inspection.Images),
    inspection.Template.Present,
)
```

返回结果可以稳定编码为 JSON，初始契约为 `SchemaVersion == 1`。其中包含
规范化后的包信息、构建配置、应用、服务、镜像和模板元数据，但不会输出构建
环境值、部署参数值、脚本或原始模板表达式。

如果只处理 Manifest，可以使用 `manifest.Analyze`。它支持普通 YAML 和常见
LazyCat Go Template 控制块、标量表达式，能够生成 YAML 安全投影并精确恢复
原始 action，但永远不会执行或渲染模板。

### 项目生命周期

`project.Service` 提供：

- `Deploy`：安装调用方提供的 LPK，并同步 `dev.id`。
- `Start`、`Stop`、`Wait`：控制并等待明确的运行状态。
- `Uninstall`：可选择是否删除应用数据。
- `Exec`：通过调用方持有的 Reader 和 Writer 执行容器命令。
- `Logs`：读取或跟随应用日志。
- `CopyTo`：把 TAR 流直接传给 `lzc-docker cp -`。

后端版本低于 `1.0.4` 时，部署服务会等待 devshell 容器就绪后同步 `dev.id`；新版本使用 pending sync。

### rsync 和完整工作流

`project/rsync` 需要 rsync `3.2.0+`。它根据 lzc-cli 默认规则和 `.gitignore` 生成 `.lzcdevignore`，并且只通过子进程环境变量传递 rsync daemon 密码。

`workflow/project` 可以按顺序编排构建、部署、同步和启动。它使用临时 LPK 文件连接各阶段，返回强类型结果，并通过 Observer 发送不含敏感信息的事件。

### 签名和验签

```go
signed, err := signature.SignFile(ctx, "app.signed.lpk", "app.lpk", signature.SignRequest{
    PrivateKeyPEM: privatePEM,
    PublicKeyPEM:  publicPEM,
    KeyID:         "dev",
})
verified, err := signature.VerifyFile(ctx, "app.signed.lpk", signature.VerifyRequest{
    KeyID: "dev",
})
_, _ = signed, verified
```

### 其他开发者平台接口

`appstore` 还提供：

- `ListImages`：列出已经复制到 LazyCat Registry 的镜像记录。
- Testflight 相关接口：发布和管理内测版本。
- `TriggerAPK`：调用 lzc-cli 2.0.9 的匿名 Android APK Shell multipart 接口，默认超时 5 秒。

### 常见错误码

| 错误码 | 先检查什么 |
|--------|------------|
| `INVALID_ARGUMENT` | 是否传了 nil Reader、Writer、Context，或空路径、空包名。 |
| `INVALID_CONFIG` | YAML 路径、包名、版本、镜像别名和必填配置。 |
| `INVALID_MANIFEST` | Manifest YAML、`application.subdomain` 和字段类型。 |
| `INCOMPATIBLE_BACKEND` | 是否给带镜像项目配置了正确的本地或远端 `ImageBuilder`，远端版本是否支持该能力。 |
| `UNAUTHENTICATED` | `LZC_CLI_TOKEN`、token 文件或登录返回值是否存在且仍有效。 |
| `PERMISSION_DENIED` | 开发者账号是否有目标应用或平台操作权限。 |
| `REMOTE_UNAVAILABLE` | 开发者平台、ShellAPI、SSH 或 DebugBridge 是否可达。 |
| `COMMAND_FAILED` | Docker、SSH、rsync 或远端命令的退出状态。 |
| `INTEGRITY_MISMATCH` | LPK 条目、摘要、签名或 OCI blob 是否损坏。 |
| `DEADLINE_EXCEEDED` | 镜像复制、应用启动或状态等待是否超过调用方设置的超时。 |

## 兼容性

实现基线：

- 软件包：`@lazycatcloud/lzc-cli`
- 版本：`2.0.9`
- integrity：`sha512-L+DUKBD5HrFctnqZ4a8vofXY7f5+4ukpfw4rSnNbeE9s48lsLOr3vvbaWZCDSR6xkivRYTovQMWKqcli6s8mUQ==`
- shasum：`88a3847bbd1c0c2e709cbc7a96fae52f9f832a85`

SDK 以 lzc-cli 的文件格式和服务端协议语义为兼容目标，不要求生成字节完全相同的归档文件。SDK 自身的版本信息和参考 lzc-cli 版本可以从 `version` 包读取，便于后续兼容升级。

## 安全性

- 除只写入本地密钥文件的密钥生成外，所有阻塞操作都接收 `context.Context`。
- 流式 API 不会关闭调用方持有的 Reader 或 Writer。
- Token 文件使用调用方指定的路径，`auth/tokenfile` 以 `0600` 权限原子写入。
- 错误和工作流事件不会包含密码、token、私钥、认证请求头或携带凭据的原始响应。
- 归档解析和解压支持大小、路径和条目数量限制，并使用 Go 1.25 的 `os.Root` 限制文件系统修改范围。
- SDK 不会在错误中返回远端命令的原始敏感输出；需要调试时，通过调用方控制的 Writer 接收经过明确选择的日志流。
