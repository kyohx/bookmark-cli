# bookmark-cli

[![Test](https://github.com/kyohx/bookmark-cli/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/kyohx/bookmark-cli/actions/workflows/test.yml)

`bookmark-sample` WebAPI を操作する Go 製 CLI です。  
`/token` で認証し、トークンは OS キーチェーンに保存します。

## 概要

- 認証: Username/Password ログイン + Refresh Token 更新
- セッション保存: Keychain / Credential Manager / Secret Service
- 主なコマンド: `login`, `logout`, `whoami`, `auth status`, `list`, `get`, `add`, `update`, `delete`, `api-version`
- バージョン確認: `version` / `--version` / `api-version`
- セキュリティ方針: パスワード非保存、401時の単回リトライ、HTTPタイムアウト適用
- APIエラー: JSONレスポンスが返ればエラー本文も表示

対応仕様: [Bookmark API OpenAPI](https://github.com/kyohx/bookmark-sample/blob/ee30d5ad8eec01c30d7b51a6101bc14009efbbeb/openapi.json)（`0.12.0.260922`）。既存の認証・ブックマーク操作を対象とし、ユーザー管理・ブラックリスト管理のCLIは提供していません。

## 必要要件

- Go 1.22+
- 接続先の `bookmark-sample` API

## 設定

デフォルト設定ファイル:

- macOS: `~/Library/Application Support/bookmark-cli/config.toml`
- Linux: `~/.config/bookmark-cli/config.toml`
- Windows: `%AppData%/bookmark-cli/config.toml`

例:

```toml
[profile.default]
base_url = "http://localhost:8000"
timeout_seconds = 10
scopes = ["read", "write"]

[profile.work]
base_url = "https://bookmark.example.com"
timeout_seconds = 15
scopes = ["read"]
```

`[profile.<プロファイル名>]` ごとにデフォルト設定を持てます。`--profile`, `--base-url`, `--timeout`, `--config` で上書きできます。`--show-profile` を付けると、起動時に実際に採用されるプロファイル内容を stderr に表示します。セッションはプロファイル単位で OS キーチェーンに保存されます。

## 使用方法

### ヘルプ

```bash
go run ./cmd/bookmark --help
go run ./cmd/bookmark --version
go run ./cmd/bookmark version
go run ./cmd/bookmark api-version
go run ./cmd/bookmark --profile work --show-profile version
```

### ログイン / ログアウト

```bash
# 対話入力（推奨）
go run ./cmd/bookmark login --username testuser

# 標準入力からパスワードを渡す
printf '%s\n' 'your-password' | go run ./cmd/bookmark login --username testuser --password-stdin

# スコープ指定（未指定時は選択中プロファイルの scopes を利用）
go run ./cmd/bookmark login --username testuser --scope read --scope write

# プロファイル指定
go run ./cmd/bookmark --profile work login --username testuser

go run ./cmd/bookmark logout
```

### ログイン状態確認

```bash
go run ./cmd/bookmark auth status
go run ./cmd/bookmark whoami
```

### ブックマーク操作

```bash
# 一覧
go run ./cmd/bookmark list --page 1 --size 10
go run ./cmd/bookmark list --tag work --tag test

# 単件取得
go run ./cmd/bookmark get <hashed_id>

# 追加（memoは省略すると空文字）
go run ./cmd/bookmark add --url "https://example.com" --memo "sample" --tag work --tag test

# 更新（memoのみ）
go run ./cmd/bookmark update <hashed_id> --memo "updated memo"

# 更新（tagsのみ）
go run ./cmd/bookmark update <hashed_id> --tag private --tag test

# 削除
go run ./cmd/bookmark delete <hashed_id>
```

`add` / `get` / `update` はブックマーク全体をJSONで出力します。`add` の出力は従来の `hashed_id` のみから全項目に、`update` は `Updated.` からJSONに変わります。

入力制約: `page` は1以上、`size` は1〜100、IDは64桁の小文字16進数、URLは400文字以内の絶対URI、memoは空文字を許容し400文字以内、tagsは1〜10件（各1〜100文字）です。更新時に省略した項目は送信せず、`--memo ""` でメモを空にできます。

## ビルド方法

```bash
go mod tidy
go build -o bookmark ./cmd/bookmark
```

実行:

```bash
./bookmark --help
```

## テスト方法

```bash
go test ./...
```

主なテスト対象:

- `internal/auth`: ログイン（`/token`）、refresh
- `internal/session`: keyring 保存/読込/削除
- `internal/api`: 401時の refresh + 単回リトライ
