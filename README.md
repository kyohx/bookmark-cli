# bookmark-cli

`bookmark-sample` WebAPI を操作する Go 製 CLI です。  
`/token` で認証し、トークンは OS キーチェーンに保存します。

## 概要

- 認証: Username/Password ログイン + Refresh Token 更新
- セッション保存: Keychain / Credential Manager / Secret Service
- 主なコマンド: `login`, `logout`, `whoami`, `auth status`, `list`, `add`, `update`, `delete`
- セキュリティ方針: パスワード非保存、401時の単回リトライ、HTTPタイムアウト適用

## 必要要件

- Go 1.22+
- 接続先の `bookmark-sample` API

## 設定

デフォルト設定ファイル:

- macOS: `~/Library/Application Support/bookmark-cli/config.yaml`
- Linux: `~/.config/bookmark-cli/config.yaml`
- Windows: `%AppData%/bookmark-cli/config.yaml`

例:

```yaml
base_url: "http://localhost:8000"
timeout_seconds: 10
scopes:
  - read
  - write
```

`--base-url`, `--timeout`, `--config` で上書きできます。

## 使用方法

### ヘルプ

```bash
go run ./cmd/bookmark --help
```

### ログイン / ログアウト

```bash
# 対話入力（推奨）
go run ./cmd/bookmark login --username testuser

# 標準入力からパスワードを渡す
printf '%s\n' 'your-password' | go run ./cmd/bookmark login --username testuser --password-stdin

# スコープ指定（未指定時は config.yaml の scopes を利用）
go run ./cmd/bookmark login --username testuser --scope read --scope write

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

# 追加
go run ./cmd/bookmark add --url "https://example.com" --memo "sample" --tag work --tag test

# 更新（memoのみ）
go run ./cmd/bookmark update <hashed_id> --memo "updated memo"

# 更新（tagsのみ）
go run ./cmd/bookmark update <hashed_id> --tag private --tag test

# 削除
go run ./cmd/bookmark delete <hashed_id>
```

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
