# bookmark-cli

[![Test](https://github.com/kyohx/bookmark-cli/actions/workflows/test.yml/badge.svg?branch=main)](https://github.com/kyohx/bookmark-cli/actions/workflows/test.yml)

`bookmark-sample` WebAPI を操作する Go 製 CLI です。  
`/token` で認証し、トークンは OS キーチェーンに保存します。

## 概要

- 認証: Username/Password ログイン + Refresh Token 更新
- セッション保存: Keychain / Credential Manager / Secret Service
- 主なコマンド: `login`, `logout`, `whoami`, `auth status`, `list`, `get`, `add`, `update`, `delete`, `user add/get/list/update`, `session list/revoke`, `api-version`
- バージョン確認: `version` / `--version` / `api-version`
- セキュリティ方針: パスワード非保存、401時の単回リトライ、HTTPタイムアウト適用
- APIエラー: JSONレスポンスが返ればエラー本文も表示

対応仕様: [Bookmark API OpenAPI](https://github.com/kyohx/bookmark-sample/blob/13aa099234193b7525ccbfc1a63b9c5ab2eda5ea/openapi.json)（`0.13.0.261003`）。認証・ブックマーク操作、ユーザーの追加・取得・一覧・更新、ユーザーのリフレッシュセッションの一覧・失効に対応しています。

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

### ユーザー操作

```bash
# 追加（管理者権限が必要）
go run ./cmd/bookmark user add test_user --authority 2
printf '%s\n' 'new-password' | go run ./cmd/bookmark user add test_user --authority 2 --password-stdin

# 単件取得・一覧（どちらも管理者権限が必要）
go run ./cmd/bookmark user get test_user
go run ./cmd/bookmark user list --page 1 --size 10

# 更新（指定した項目だけ送信）
go run ./cmd/bookmark user update test_user --authority 1 --disabled=false
go run ./cmd/bookmark user update test_user --name new_name
go run ./cmd/bookmark user update test_user --password
printf '%s\n' 'new-password' | go run ./cmd/bookmark user update test_user --password-stdin
```

パスワードは通常、画面に表示せず入力します。`--password-stdin` は標準入力から読み取ります。`--authority` は 0（権限なし）、1（読み取り）、2（読み書き）、9（管理者）のいずれかを指定します。ユーザー名は英数字とアンダースコアで1〜32文字、パスワードは8〜64文字です。`user update` では `--disabled=false` も明示的に送信できます。API の仕様上、自分自身の名前・権限・無効化状態は変更できず、管理者以外は自分の情報のみ更新できます。現行 API にユーザー削除はありません。

### セッション操作

```bash
# ユーザーの有効期限内のリフレッシュセッションを一覧表示（管理者権限が必要）
go run ./cmd/bookmark session list test_user

# 一覧の id を指定してリフレッシュトークンを失効（管理者権限が必要）
go run ./cmd/bookmark session revoke test_user <session_id>

# プロファイル指定
go run ./cmd/bookmark --profile work session list test_user
```

`session list` は `id`, `created_at`, `last_used_at`, `expires_at`, `revoked`, `user_agent` を含むJSON配列を出力します。セッションがない場合は `[]`、User-Agentがない場合は `user_agent: null` になります。一覧APIにページ指定はありません。`session revoke` は成功時に `Revoked.` と表示します。

ユーザー名は英数字とアンダースコアで1〜32文字、セッションIDは1〜128文字です。どちらの操作にも管理者としてのログインが必要です。失効の対象はサーバー上のリフレッシュトークンです。`logout` は選択中プロファイルのローカル認証情報を削除します。

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
