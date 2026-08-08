# bookmark-sample WebAPI向け CLI 設計書（Go）

## 1. 目的
- `bookmark-sample` のWebAPIを安全に利用するCLIを実装する。
- 要件は **モダン / シンプル / セキュリティ堅牢**。
- `/token` API を使う対話ログインを主経路にし、秘密情報はOS保護領域で管理する。

## 2. 非機能要件
- **Security First**: 秘密情報を平文保存しない。ログに機密を出さない。
- **Simplicity**: 薄い責務分離（auth/session/api/cmd）で保守容易。
- **Reliability**: タイムアウト・再試行・401時の単回リフレッシュを実装。
- **Portability**: macOS/Windows/Linuxで同一実装方針。

## 3. 採用技術
- CLIフレームワーク: `cobra`
- HTTP: `net/http` + `context`
- Tokenモデル: `golang.org/x/oauth2` の `oauth2.Token` を流用（`/token` 応答をマッピング）
- 秘密情報保管: `github.com/zalando/go-keyring`
- 設定ファイル（非機密）: `~/.config/bookmark-cli/config.yaml`

## 4. ディレクトリ構成
```text
bookmark-cli/
  cmd/bookmark/
    main.go
    root.go
    login.go
    logout.go
    whoami.go
    bookmark_list.go
    bookmark_add.go
    bookmark_delete.go
  internal/
    auth/
      service.go
      password_login.go
      token_source.go
    session/
      keyring_store.go
    api/
      client.go
      middleware_auth.go
    config/
      config.go
    cli/
      output.go
      error.go
```

## 5. 認証方式
### 5.1 第一候補: `/token` API + CLI対話入力
1. `bookmark login` で `username` を受け取る（引数または対話）
2. `password` は非表示入力（`term.ReadPassword`）で取得
3. CLIが `/token` へTLS経由で送信
4. `access_token` / `refresh_token` / `token_type` を取得
5. 取得トークンをkeyringへ保存（パスワードは即時破棄・非保存）

### 5.2 代替案
- Device Authorization Grant（将来のSSO連携時）
- Personal Access Token方式（`bookmark login --token-stdin`）は限定用途でサポート

## 6. セッション保存ポリシー
- 保存先は **OSキーチェーンのみ**（Keychain / Credential Manager / Secret Service）。
- `config.yaml` は非機密のみ:
  - `base_url`
  - `client_id`（必要な場合）
  - `scopes`（必要な場合）
- 保存キー名（例）:
  - service: `bookmark-cli`
  - user: `<api_base_url>#<subject or user_id>`

## 7. APIクライアント設計
### 7.1 振る舞い
- 全リクエストに `context.WithTimeout`（例: 10秒）
- Authorizationヘッダは `Bearer <access_token>`
- `401 Unauthorized` 受信時:
  1. refresh tokenで再取得
  2. 同一リクエストを **1回だけ** 再実行
  3. 失敗時は再ログインを促す

### 7.2 同時実行制御
- refresh処理は `sync.Mutex` で直列化し、同時401時の多重refreshを防止。

## 8. 主要インターフェース
```go
type TokenStore interface {
    Save(ctx context.Context, token *oauth2.Token) error
    Load(ctx context.Context) (*oauth2.Token, error)
    Delete(ctx context.Context) error
}

type AuthService interface {
    Login(ctx context.Context, username string, password []byte, scopes []string) error
    Logout(ctx context.Context) error
    TokenSource(ctx context.Context) (oauth2.TokenSource, error)
}

type APIClient interface {
    Do(ctx context.Context, req *http.Request) (*http.Response, error)
}
```

## 9. CLIコマンド
- `bookmark login`  
  `/token` APIで認証し、セッションをkeyringに保存。
- `bookmark login --username <name>`  
  ユーザー名のみ引数指定し、パスワードは非表示プロンプト入力。
- `bookmark login --password-stdin`  
  CIやパイプ入力用（履歴に残る引数でのパスワード指定は禁止）。
- `bookmark login --scope read --scope write`  
  必要最小スコープでのトークン要求。未指定時は`config.yaml`の`scopes`を使用。
- `bookmark logout`  
  keyring上のセッションを削除。
- `bookmark whoami`  
  APIに問い合わせてログイン状態を表示。
- `bookmark auth status`  
  トークン有効期限・再認証要否を表示。
- `bookmark list/add/update/delete ...`  
  業務コマンド群（API呼び出し）。

## 10. エラーハンドリング方針
- ユーザー向けメッセージと内部エラーを分離。
- 代表エラー:
  - `ErrNotLoggedIn`
  - `ErrTokenExpired`
  - `ErrRefreshFailed`
  - `ErrUnauthorized`
  - `ErrAPIUnavailable`
- エラーメッセージにトークン/パスワード/Authorizationヘッダを含めない。

## 11. ログ/監査方針
- デフォルトは最小ログ（INFO中心）。
- `--debug` でもトークンと認証ペイロードは必ずマスク。
- HTTPダンプは機密ヘッダを除外する。

## 12. セキュリティチェックリスト
- [x] トークン平文保存なし
- [x] パスワード保存なし（メモリ保持最小、送信後破棄）
- [x] TLS検証無効化コードなし
- [x] すべてのHTTPにtimeoutあり
- [x] 401再試行は1回のみ
- [x] logoutで確実にセッション削除
- [x] ログ/エラーに機密情報なし
- [x] 最小スコープ運用

## 13. 実装ステップ
1. [x] `config` と `TokenStore(keyring)` を実装
2. [x] `AuthService(Login/Logout/TokenSource)` を実装
3. [x] `api.Client` に401時refresh再試行を実装
4. [x] `login/logout/whoami/auth status` コマンド実装
5. [x] bookmark業務コマンド（list/add/update/delete）を実装
6. [x] 単体テスト（auth/session/api）を追加

---
この設計は、現状の `/token` API を最短で活かしつつ、将来のSSO連携にも移行しやすい最小構成を意図している。
