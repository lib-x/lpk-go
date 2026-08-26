# lzc-toolkit-go

[English](README.md) | [简体中文](README.zh-CN.md)

`lzc-toolkit-go` is a Go library for building, reading, inspecting, signing, deploying, and publishing LazyCat LPK packages. It follows the source and service contracts of `@lazycatcloud/lzc-cli` `2.0.9`, but it does not copy the CLI's prompts or terminal behavior.

If you are new to LazyCat, think of this repository as a set of composable building blocks. Your program decides when to build, which image backend to use, where credentials come from, and how results enter your CI/CD workflow.

## What this library does

Common use cases include:

- Build a LazyCat project into a `.lpk` file from Go.
- Parse an uploaded LPK from an `io.Reader`, read its Manifest, or extract it safely.
- Build embedded LPK images with the local Docker daemon.
- Build images remotely through DebugBridge and deploy the app to a LazyCat device for preview.
- Ask the LazyCat developer platform to copy a public image into `registry.lazycat.cloud`.
- Log in, validate, upload, and submit an LPK from CI/CD.
- Choose between general validation and LazyCat official submission checks.

This is a library, not a CLI. It does not open login prompts, ask whether to overwrite a file, take over the terminal, or install SSH keys. The caller owns configuration, interaction, logging, and retry policy.

## Installation

The module requires Go `1.25` or newer:

```bash
go get github.com/lib-x/lzc-toolkit-go
```

The base packages do not import or execute Docker, SSH, gRPC, or App Store integrations. Those dependencies are needed only when you import the corresponding optional package.

## Core concepts

| Concept | Meaning |
|---------|---------|
| LPK | A LazyCat application package, normally stored as a `.lpk` file. It can contain application metadata, a Manifest, static content, image layers, and signatures. |
| Manifest | YAML that describes how the app runs, including the primary service, images, domains, routes, environment variables, and auxiliary services. The project file is usually `lzc-manifest.yml`; inside an LPK it is `manifest.yml`. |
| `package.yml` | Static application metadata in LPK v2, including package ID, version, name, description, and locales. |
| `lzc-build.yml` | Project build configuration. It points to the Manifest, content directory, icon, images, and exported resources. It is not the runtime Manifest. |
| LPK v1 / v2 | Two compatible layouts. v1 keeps package metadata in the Manifest. v2 uses a separate `package.yml` and supports OCI images and resource exports. New projects normally use v2. |
| OCI image layout | The LPK v2 representation of container images. `images.lock` describes images and layers; `images/` stores blobs that must travel inside the LPK. |
| `ImageBuilder` | The `build` package is independent of Docker. Projects with images must explicitly select a local Docker or remote DebugBridge builder. |
| General lint | Checks package and Manifest validity without applying LazyCat developer-platform preferences. |
| Official lint | `lint.WithOfficial()` enables submission rules such as LazyCat Registry, icon, locale, and version requirements. |
| Developer-platform token | Authenticates image copy, application creation, Testflight, and LPK submission. It is separate from ShellAPI credentials and SSH keys. |
| ShellAPI | A local discovery API provided by the LazyCat client. It returns the default box, login UID, and client `dev.id`. |
| DebugBridge | The protocol exposed by LazyCat Developer Tools for remote image builds, LPK installation, lifecycle control, commands, and logs. |
| `project` | A lifecycle service built on DebugBridge, providing deploy, start, stop, wait, logs, exec, and file copy operations. |

## How the pieces fit

A project without images can be built directly. A project with images first selects a local or remote image backend:

```text
local project
  -> build.Build / build.BuildFile
  -> LPK
     -> inspect / lint / signature        local processing
     -> project + DebugBridge              preview on a LazyCat device
     -> appstore.Publish                   submit to developer platform

project with images
  -> dockerlocal or buildpack
  -> OCI image layout
  -> build.Build
  -> LPK
```

Image copy is a separate path:

```text
public Registry image
  -> LazyCat developer-platform CopyImage API
  -> registry.lazycat.cloud/... image reference
```

That copy runs on the developer platform. It does not require Docker on the caller's machine.

## Choose a package

| Task | Packages to import |
|------|--------------------|
| Read, write, or extract an LPK | `lpk` |
| Parse and preprocess a Manifest | `manifest` |
| Inspect a local source project | `project` |
| Inspect an LPK summary | `inspect` |
| Build a project | `build` |
| Validate an OCI layout | `oci` |
| Build images with local Docker | `image/dockerlocal` |
| Build images through DebugBridge | `image/buildpack`, `remote/blobcache` |
| Log in and obtain a token | `auth`, `auth/tokenfile` |
| Copy images and publish LPKs | `appstore` |
| Read the official public catalog | `appstore/official` |
| Query a Miaomiao private store | `appstore/private` |
| Discover the local LazyCat client | `remote/shellapi` |
| Use SSH and DebugBridge | `remote/ssh`, `remote/debugbridge` |
| Deploy and manage an app | `project` |
| Synchronize source files | `project/rsync` |
| Orchestrate build, deploy, sync, and start | `workflow/project` |
| Sign and verify packages | `signature` |

## Tutorials

The examples are ordered by external dependencies. The first needs only Go. The second needs local Docker. The third needs a LazyCat device and SSH. The fourth and fifth need a developer-platform token.

| Scenario | Local Docker | Developer-platform token | LazyCat device and SSH |
|----------|--------------|--------------------------|------------------------|
| Build and parse an LPK without images | Not required | Not required | Not required |
| Build embedded images locally | Required | Not required | Not required |
| Build remotely and preview through DebugBridge | Not required; Docker runs remotely | Not required | Required |
| Copy an image into LazyCat Registry | Not required; the platform copies it | Required | Not required |
| Submit an LPK to the developer platform | Not required | Required | Not required |

### Example 1: Build, parse, and inspect an LPK locally

This example does not need Docker, a LazyCat account, or a device. It builds a static page as `hello.lpk`, then reads the package summary and effective Manifest.

Project tree:

```text
hello-lpk/
├── lzc-build.yml
├── lzc-manifest.yml
├── package.yml
└── content/
    └── html/
        └── index.html
```

`hello-lpk/lzc-build.yml`:

```yaml
manifest: ./lzc-manifest.yml
contentdir: ./content
```

`hello-lpk/package.yml`:

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

`hello-lpk/lzc-manifest.yml`:

```yaml
application:
  subdomain: hello
  routes:
    - /=file:///lzcapp/pkg/content/html
```

`hello-lpk/content/html/index.html` can be any static page:

```html
<!doctype html>
<html lang="en">
  <meta charset="utf-8">
  <title>Hello LPK</title>
  <h1>Hello LazyCat</h1>
</html>
```

Build and read the LPK:

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

The program creates `hello.lpk`. `BuildFile` uses atomic replacement, so a failed build does not leave a partial output file. `ForceV2` selects the layout where package ID and version live in `package.yml`.

When the destination is already an `io.Writer`, use `build.Build`:

```go
var output bytes.Buffer
result, err := build.Build(ctx, &output, build.Request{
    Root:    "./hello-lpk",
    ForceV2: true,
})
```

`build.Build` does not close `output`. Likewise, `lpk.Open`, `inspect.Stream`, and publishing APIs do not close caller-owned readers.

Common failures:

- `INVALID_CONFIG`: check paths in `lzc-build.yml`, the package ID, and the version.
- `INVALID_MANIFEST`: check YAML shapes and `application.subdomain`.
- Manifest not found: confirm the configured path exists. The default is `lzc-manifest.yml`.
- Build script did not run: scripts are disabled by default and run only when `RunBuildScript` is true.

### Example 2: Build an image-bearing LPK with local Docker

This scenario requires Docker on the caller's machine:

- The `docker` executable must be available.
- The Docker daemon must be running.
- `docker buildx` must be available.

Importing `build`, `lpk`, `oci`, or `image` alone does not call Docker. The SDK executes Docker only after you explicitly pass `dockerlocal.New(...)`.

Add `app/Dockerfile` and reference the image alias as `embed:app`:

```text
docker-hello/
├── app/
│   └── Dockerfile
├── lzc-build.yml
├── lzc-manifest.yml
└── package.yml
```

`docker-hello/lzc-build.yml`:

```yaml
manifest: ./lzc-manifest.yml
images:
  app:
    builder: local
    context: ./app
    dockerfile: ./app/Dockerfile
```

`docker-hello/package.yml`:

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

`docker-hello/lzc-manifest.yml`:

```yaml
application:
  subdomain: docker-hello
  image: embed:app
```

`docker-hello/app/Dockerfile`:

```dockerfile
FROM alpine:3.20
CMD ["sh", "-c", "while true; do echo hello; sleep 30; done"]
```

Go program:

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

`dockerlocal.New(nil)` uses the Docker CLI backend and targets `linux/amd64`. Select another platform explicitly:

```go
builder := dockerlocal.New(nil, dockerlocal.WithPlatform("linux/arm64"))
```

The adapter runs the lzc-cli 2.0.9 compatible `docker buildx build --load`, `docker image inspect`, and `docker image save` flow, then converts the result into the LPK v2 OCI layout. Docker 29 responses that expose a manifest descriptor through `Id` are resolved back to the saved config JSON digest, and already-gzipped saved layers retain their original compressed bytes. `images.lock` describes the image and its layers; `images/` contains only blobs embedded in the package. The builder appends the resolved image digest to `embed:app`.

Common failures:

- `INCOMPATIBLE_BACKEND`: the project has images but no `ImageBuilder`, or a remote image configuration was sent to the local builder.
- `COMMAND_FAILED`: the Docker CLI, Dockerfile, or base image pull failed.
- `INTEGRITY_MISMATCH`: the generated OCI layout, digest, or blob is inconsistent.

### Example 3: Deploy to a LazyCat device for preview

In this workflow, source editing stays local while images and services run on the LazyCat or build-remote side. You open the resulting app through the normal LazyCat client entry.

DebugBridge does not expose a process listening on your computer's `localhost:3000` to the device. Previewing a local process requires a separate reverse tunnel or proxy; that is not part of the current SDK or lzc-cli 2.0.9 DebugBridge contract.

Prerequisites:

- The LazyCat client is installed and logged in, with ShellAPI files available.
- A default box is selected.
- Developer Tools is installed and enabled on the target.
- The build-remote host is reachable through SSH.
- SSH public-key authorization is already configured. The SDK does not install keys.
- The remote DebugBridge version supports the requested capabilities.

The following variables belong to this example program. They are not SDK-defined environment variables:

```bash
export LZC_BUILD_REMOTE_USER=developer
export LZC_BUILD_REMOTE_ADDR=build-host.example:22
```

The complete path includes ShellAPI discovery, SSH transport, remote image building, LPK installation, and app startup:

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

This example expects an `lzc-build.dev.yml` whose image entries use the remote builder. The remote builder creates deterministic contexts from Dockerfile `COPY` and `ADD` sources, honors `.dockerignore`, and uses the per-project blob cache.

An iterative development loop can also combine:

- `project/rsync.Sync` to synchronize local source into `/lzcapp/cache/project-mirror`.
- `project.Exec` to run commands or restart the development process.
- `project.CopyTo` to copy a local directory through a TAR stream.
- `project.Logs` to read once or follow logs.

The SDK does not include a filesystem watcher. The caller selects a watcher, invokes `Sync` after changes, and applies the project's restart policy.

### Example 4: Copy a public image into LazyCat Registry

This operation does not require Docker on the caller's machine.

You need only:

- Network access to the LazyCat developer platform.
- A valid developer-platform token.
- A source image that the developer platform can pull.

The developer platform performs the copy. The operation does not use local Docker, a box Docker daemon, ShellAPI, or SSH.

Expose the token through the environment:

```bash
export LZC_CLI_TOKEN='your-token'
```

Do not put a real token in source code or commit it to Git.

Complete example:

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

`LazyCatImage` is the resulting `registry.lazycat.cloud/...` reference. The result also preserves the source image, selected platform, and final layer state, so CI/CD callers do not need to parse terminal output.

The lzc-cli 2.0.9 and Go SDK `CopyImageRequest` contracts have no source Registry username, password, or token field. Callers cannot add private Registry credentials through this API and must ensure that the developer platform can pull the source image.

### Example 5: Log in and submit an LPK

Legacy developer-platform session authentication supports two flows.

The developer platform also exposes a separate PAT API. Keep its credential
distinct from the legacy lzc-cli session token: `appstore.New` uses
`/api/v3/developer`, `X-User-Token`, and the `userToken` cookie, while
`appstore.NewPAT` uses `/sdk/v3/developer` and `X-API-Token`:

```go
client, err := appstore.NewPAT(appstore.Options{
    Token: auth.StaticToken(os.Getenv("LZC_API_TOKEN")),
})
```

`NewPAT` disables redirects and does not forward the PAT to another origin.
The SDK does not copy or synchronize PAT and session-token values.

Use `WaitingReviewVersion` to check whether an exact package currently has a
version under review. A 404 response is represented as `found == false`, not
as an error:

```go
version, found, err := client.WaitingReviewVersion(ctx, "cloud.lazycat.example")
if err != nil {
    return err
}
if found {
    fmt.Printf("%s is awaiting review\n", version)
}
```

#### Flow 1: Exchange username and password for a token

`auth.Client.Login` submits credentials to `https://account.lazycat.cloud/api/login/signin` and returns `Session.Token`. The password is not stored. When a `TokenStore` is configured, only the token is saved.

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

Keeping an account password in CI is not recommended. Log in once from a trusted environment, save the token in the CI secret store, and inject it at runtime.

#### Flow 2: Provide an existing token

The SDK supports:

- `auth.StaticToken` for a caller-provided value.
- `auth.EnvironmentToken`, which reads `LZC_CLI_TOKEN` by default.
- `auth.StoreProvider` for an explicit `TokenStore`.

When lzc-cli is already logged in, lzc-cli 2.0.9 resolves the token in this order:

1. Use `LZC_CLI_TOKEN` when it is set.
2. Otherwise read the `token` field from `~/.config/lazycat/box-config.json`.

`lzc-cli config get token` prints the effective token, but should not be run in CI logs. The Go SDK does not read the user-level file implicitly; configure `tokenfile.Store{Path: tokenPath}` explicitly.

Submit an LPK that already passes official lint:

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

`Publish` performs these steps:

1. Spool the stream under the configured size limit.
2. Parse and safely extract the LPK.
3. Run `lint.WithOfficial()` checks.
4. Check whether the application exists on the developer platform.
5. Upload the LPK and validate the returned package ID.
6. Submit the localized changelog for review.

`Publish` does not close `packageFile`. To permit automatic application creation, set `CreateIfMissing: true` and provide `Application`; the library never asks interactively.

For a first official information submission, query the authenticated developer state instead of guessing from the public catalog. `ApplicationState` distinguishes a missing application, approved information, and an existing pending review:

```go
state, err := client.ApplicationState(ctx, "cloud.lazycat.example")
if err != nil {
    return err
}
if state.ReviewPending {
    return errors.New("an official review is already pending")
}
```

Screenshot processing lives in the separate `appstore/media` package. It reads only a project-relative PNG or JPEG regular file, rejects paths outside the project root and symbolic links, enforces a 15 MiB and 320-3840 pixel boundary, center-crops to 16:9, and returns a bounded ASCII-named PNG:

```go
asset, err := media.NormalizeFile(ctx, projectRoot, ".github/screenshots/pc-1.png")
if err != nil {
    return err
}
uploadedPath, err := client.UploadApplicationImage(
    ctx,
    bytes.NewReader(asset.Data),
    asset.FileName,
)
```

Pass the returned developer-platform paths through `PublishRequest.ApplicationInfos`. Desktop information requires 2-8 screenshots and mobile information requires 3-8. Do not submit information again when `InformationReady` is true, and stop before uploading when `ReviewPending` is true.

An official submission usually also requires:

- Images from `registry.lazycat.cloud` or valid embedded images.
- A PNG `icon.png` no larger than 200 KiB.
- `locales` in `package.yml`.
- A SemVer version.
- No devshell in the submitted package.

### Read application stores without a token

The official catalog client is anonymous. It does not accept or construct an
account token:

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

Import `github.com/lib-x/lzc-toolkit-go/appstore/official`. The package also
provides `Homepage`, `Categories`, `Kinds`, `More`, `DownloadRanking`,
`DeveloperRanking`, and `VersionChangelog`. Metadata and LPK base URLs can be
overridden in `official.Options` for tests and mirrors.

Miaomiao private community stores use a separate package and may expose apps
through six-character private group codes:

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

Import `github.com/lib-x/lzc-toolkit-go/appstore/private` as `privatestore`.
Group codes are bearer credentials. They are normalized, deduplicated, and sent
through `X-Group-Codes` by default; query and combined placement are explicit
options. The latest-version endpoint also requires no account token. Missing,
unpublished, versionless, and inaccessible packages all return
`lpkgo.ErrNotFound`.

## Advanced reference

### LPK Reader and Writer

Write to any `io.Writer`:

```go
result, err := lpk.Write(ctx, dst, lpk.WriteRequest{
    Layout: lpk.LayoutV2,
    Files:  os.DirFS("package-root"),
    Strict: true,
})
```

Open a sequential stream. The implementation spools it into bounded temporary storage:

```go
reader, err := lpk.Open(ctx, src)
```

Avoid whole-stream spooling when random access is available:

```go
reader, err := lpk.OpenReaderAt(ctx, src, size)
```

`Reader` can list entries, read the Manifest and `package.yml`, open a single entry, extract safely, and return the effective merged Manifest. Archive parsing supports configurable size, path, and entry-count limits.

`project.Restore` converts a published package into a project directory that can
be passed to `build.Build` again. `project.RestoreWithOptions(...,
project.RestoreOptions{PreferUpstreamImages: true})` uses only explicit
`# upstream:` image comments from the Manifest. A LazyCat Registry digest does
not encode a reversible upstream tag or version, so the restore operation never
guesses an image reference when that comment is absent.

#### Extracting a store LPK

The `Version.PackagePath` returned by the public store API can be resolved with
`official.Client.ApplicationDownloadURL`, then downloaded and passed to
`lpk.Open` and `Reader.Extract`:

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

The extracted directory contains the published `manifest.yml`, `package.yml`,
static files, OCI image layers, and resource exports, and can be inspected with
`inspect` or `lint`. This restores the published package, not the original
source project: source files, Dockerfiles, `lzc-build.yml`, and build scripts
that were not included in the LPK cannot be reconstructed, and embedded images
are not converted back into Dockerfiles.

### Lint

Check an extracted LPK root:

```go
warnings, err := lint.Package(ctx, os.DirFS("package-root"))
```

Before submitting to the LazyCat developer platform:

```go
warnings, err := lint.Package(
    ctx,
    os.DirFS("package-root"),
    lint.WithOfficial(),
)
```

Official rules are disabled by default because Registry, icon, locale, SemVer, and devshell restrictions are platform submission preferences, not universal LPK validity rules.

### ShellAPI, SSH, and authentication boundaries

| Operation | Credential |
|-----------|------------|
| Discover the default box and `dev.id` | LazyCat client `shellapi_addr` and `shellapi_cred` |
| build-remote, DebugBridge, deployment | Target-host SSH authorization and LazyCat UID |
| Image copy, application creation, Testflight, LPK submission | Developer-platform token |
| APK Shell trigger | No App Store token, matching lzc-cli 2.0.9 |

Default ShellAPI directories:

- Linux: `~/.config/hportal-client/`
- macOS: `~/Library/Application Support/hportal-client/`
- Windows: `~/AppData/Roaming/hportal-client/`

### Inspect a local project

`project.Inspect` reads an existing project without running `buildscript`,
building images, writing an LPK, or contacting a LazyCat device:

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

The result is deterministic JSON-ready data with `SchemaVersion == 1`. It
contains normalized package, build, application, service, image, and template
metadata, but never build environment values, deployment values, scripts, or
raw template actions.

For Manifest-only tooling, `manifest.Analyze` accepts plain YAML and common
LazyCat Go Template controls and scalar expressions. It creates a YAML-safe
projection and can restore the exact original actions, but it never executes
or renders the template.

### Project lifecycle

`project.Service` provides:

- `Deploy` to install a caller-provided LPK and synchronize `dev.id`.
- `Start`, `Stop`, and `Wait` for explicit state transitions.
- `Uninstall` with optional application-data deletion.
- `Exec` with caller-owned input and output streams.
- `Logs` for one-shot or follow mode.
- `CopyTo` using a TAR stream sent directly to `lzc-docker cp -`.

Backends before `1.0.4` wait for the devshell container before synchronizing `dev.id`; newer backends use pending sync.

### rsync and complete workflow

`project/rsync` requires rsync `3.2.0+`. It creates `.lzcdevignore` from lzc-cli defaults and `.gitignore`, and passes the rsync daemon password only through the child process environment.

`workflow/project` composes build, deployment, optional synchronization, and startup. It joins stages with a temporary LPK, returns typed results, and emits credential-safe events through an Observer.

### Sign and verify

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

### Other developer-platform APIs

`appstore` also provides:

- `ListImages` to list images copied into LazyCat Registry.
- Testflight APIs for internal-test releases.
- `TriggerAPK` for the lzc-cli 2.0.9 anonymous Android APK Shell multipart endpoint, with a five-second default timeout.

### Common error codes

| Code | First thing to check |
|------|----------------------|
| `INVALID_ARGUMENT` | Nil Reader, Writer, or Context; empty path or package ID. |
| `INVALID_CONFIG` | YAML paths, package ID, version, image aliases, and required configuration. |
| `INVALID_MANIFEST` | Manifest YAML, `application.subdomain`, and field shapes. |
| `INCOMPATIBLE_BACKEND` | Correct local or remote `ImageBuilder`; remote backend version. |
| `UNAUTHENTICATED` | `LZC_CLI_TOKEN`, token file, or login result exists and remains valid. |
| `PERMISSION_DENIED` | The developer account is allowed to operate on the application. |
| `REMOTE_UNAVAILABLE` | Developer platform, ShellAPI, SSH, or DebugBridge is reachable. |
| `COMMAND_FAILED` | Docker, SSH, rsync, or remote command exit status. |
| `INTEGRITY_MISMATCH` | LPK entry, digest, signature, or OCI blob integrity. |
| `DEADLINE_EXCEEDED` | Image copy, app startup, or state wait exceeded its timeout. |

## Compatibility

Implementation baseline:

- package: `@lazycatcloud/lzc-cli`
- version: `2.0.9`
- integrity: `sha512-L+DUKBD5HrFctnqZ4a8vofXY7f5+4ukpfw4rSnNbeE9s48lsLOr3vvbaWZCDSR6xkivRYTovQMWKqcli6s8mUQ==`
- shasum: `88a3847bbd1c0c2e709cbc7a96fae52f9f832a85`

The compatibility target is lzc-cli file formats and service-facing semantics. Byte-identical archives are not required. The `version` package reports both this SDK's version and the exact lzc-cli reference version for future compatibility upgrades.

## Security

- Every blocking operation accepts `context.Context` except key generation, which writes only local key files.
- Streaming APIs do not close caller-owned readers or writers.
- Token persistence uses caller-selected paths; `auth/tokenfile` writes atomically with `0600` permissions.
- Errors and workflow events do not contain passwords, tokens, private keys, authorization headers, or raw credential-bearing responses.
- Archive parsing and extraction enforce configurable size, path, and entry-count limits and use Go 1.25 `os.Root` to constrain filesystem writes.
- The SDK does not return raw sensitive remote-command output in errors. Callers can route explicitly selected log streams into their own writers.
