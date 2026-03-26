# xray-reality-solo-vpn

[English](./README.md) | [简体中文](./README.zh-CN.md) | 日本語

`xray-reality-solo-vpn` は、個人運用向けの単一 VPS 用セルフホスト型セキュアアクセス管理パネルです。パネル内の製品名として `Solo VPN` を使います。

## 上流リファレンス

- Xray-core 上流リポジトリ: <https://github.com/XTLS/Xray-core>
- REALITY 上流リポジトリ: <https://github.com/XTLS/REALITY>

このプロジェクトは `Xray-core + VLESS + REALITY` を前提に構築されていますが、上流リポジトリのミラーでも公式 XTLS プロジェクトでもありません。プロトコル挙動、各種パラメータの意味、互換性の判断は上流リポジトリを参照してください。

## プロダクト範囲

- 単一マシン構成
- 初回管理者セットアップ
- ログインとセッション管理
- クライアントの作成 / 停止 / 削除
- `vless://` 共有リンク生成
- Mihomo / Clash Meta サブスクリプション配布
- 基本的なトラフィック可視化と最終利用時刻表示
- Xray ランタイム設定の再生成と同期
- パネルからの管理者パスワード変更

## 技術スタック

- `Go` 管理バックエンド
- `React + Vite + Tailwind + shadcn/ui` スタイルのフロントエンド
- 組み込み `SQLite`
- `Xray-core + VLESS + REALITY`
- `systemd + Caddy + Nginx stream` によるホスト配備

## ディレクトリ構成

- `cmd/manager/`
  Go エントリポイント
- `internal/`
  設定、認証、HTTP API、SQLite、サブスクリプション、ランタイム同期
- `web/`
  フロントエンドソースとビルド成果物
- `scripts/install.sh`
  対話式ホストインストール
- `scripts/check.sh`
  残留物とポート占有のチェック
- `scripts/cleanup.sh`
  旧デプロイの削除
- `deploy/`
  systemd / Caddy / Nginx / Xray テンプレート
- `scripts/bootstrap-reality-env.sh`
  `XRAY_PRIVATE_KEY`、`XRAY_PUBLIC_KEY`、`SESSION_SECRET` の生成
- `generated/server.json`
  生成済み Xray ランタイム設定
- `data/manager.db`
  SQLite データベース

## ホストインストール

対象の Ubuntu VPS で実行します:

```bash
./scripts/check.sh
./scripts/install.sh
```

`install.sh` は必要な値だけを質問し、`/etc/xray-reality-solo-vpn/app.env` を生成し、ホストサービスをセットアップしてワンタイム setup URL を表示します。

## デプロイ成果物の要件

サーバーへアップロードするプロジェクトには、事前に次の 2 つのビルド成果物を含めておく必要があります。

- `web/dist/`
- `build/manager-linux-amd64`

想定するデプロイ手順:

1. ローカルでフロントエンドとバックエンドをビルド
2. 上記 2 つの成果物を含む完全なプロジェクトディレクトリをサーバーへアップロード
3. サーバー上で以下を実行:

```bash
./scripts/check.sh
./scripts/install.sh
```

デプロイ時に不足しているフロントエンド/バックエンド成果物をサーバー側で補完ビルドする前提にはしないでください。

## ホストサービス

- `xray-reality-solo-vpn.service`
  Go API と静的フロントエンドを `127.0.0.1:3000` で提供
- `xray.service`
  Reality サーバーを `127.0.0.1:2443` で提供
- `caddy.service`
  パネル HTTPS
- `nginx.service`
  `443/tcp` の SNI 振り分け

## ドメインと接続先

- `PANEL_DOMAIN`
  パネル用ドメイン。例: `panel.example.com`
- `LINE_DOMAIN`
  UI とサブスクリプションに表示する論理ドメイン
- `LINE_SERVER_ADDRESS`
  クライアントが実際に接続する先。fake-ip DNS や TUN ループがある場合はサーバーのグローバル IP を使う方が安全です。

## セットアップフロー

- 公開 `/setup` は既定でロック
- インストーラはワンタイム URL を表示:
  `https://<panel-domain>/_/setup/<token>`
- その URL を開いたブラウザだけが初回管理者作成を実行可能
- セットアップ成功後、その URL は失効し、以後は `/login` を使用

## ローカル開発

バックエンド:

```bash
go test ./...
go run ./cmd/manager
```

フロントエンド:

```bash
npm --prefix web install
npm --prefix web run dev
npm --prefix web run build
```

Docker / Compose デプロイ経路は廃止され、現在は `scripts/install.sh` によるホストスクリプト配備のみをサポートします。

## デプロイメモ

- `.env`、`generated/`、`data/` は Git に含めない
- `panel.example.com` と `line.example.com` は通常同じ VPS を指す
- サーバー正常でも回線が不安定なら、まず DNS、fake-ip、TUN ループ、VPS 回線品質を確認

## ライセンス

このプロジェクトはデュアルライセンスです。以下のいずれかを選んで利用できます。

- MIT（`LICENSE-MIT`）
- Apache License 2.0（`LICENSE-APACHE`）
