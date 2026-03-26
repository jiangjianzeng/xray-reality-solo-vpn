#!/usr/bin/env bash

normalize_lang() {
  case "${1:-en}" in
    1) printf 'en' ;;
    2) printf 'zh' ;;
    3) printf 'ja' ;;
    zh|zh-CN|zh_cn) printf 'zh' ;;
    ja|ja-JP|ja_jp) printf 'ja' ;;
    *) printf 'en' ;;
  esac
}

select_lang_if_needed() {
  if [[ -n "${SOLO_VPN_LANG:-}" ]]; then
    SOLO_VPN_LANG="$(normalize_lang "${SOLO_VPN_LANG}")"
    LINEGATE_LANG="${SOLO_VPN_LANG}"
    export SOLO_VPN_LANG LINEGATE_LANG
    return
  fi

  if [[ -n "${LINEGATE_LANG:-}" ]]; then
    LINEGATE_LANG="$(normalize_lang "${LINEGATE_LANG}")"
    SOLO_VPN_LANG="${LINEGATE_LANG}"
    export SOLO_VPN_LANG LINEGATE_LANG
    return
  fi

  if [[ -t 0 ]]; then
    local input
    printf '%s\n' "Select language / 选择语言 / 言語を選択:"
    printf '%s\n' "  1) English"
    printf '%s\n' "  2) 中文"
    printf '%s\n' "  3) 日本語"
    printf '%s' "> [1]: "
    read -r input || true
    LINEGATE_LANG="$(normalize_lang "${input:-1}")"
  else
    LINEGATE_LANG="en"
  fi
  SOLO_VPN_LANG="${LINEGATE_LANG}"
  export SOLO_VPN_LANG LINEGATE_LANG
}

translate() {
  local key="$1"
  case "${LINEGATE_LANG:-en}:$key" in
    en:install_intro_title) printf 'Solo VPN host installer' ;;
    zh:install_intro_title) printf 'Solo VPN 宿主机安装脚本' ;;
    ja:install_intro_title) printf 'Solo VPN ホストインストーラ' ;;

    en:install_intro_body) printf 'This installer prepares a self-hosted secure access environment for a single server and administrator.' ;;
    zh:install_intro_body) printf '该安装脚本用于为单台服务器和单管理员场景部署自建安全接入环境。' ;;
    ja:install_intro_body) printf 'このインストーラは、単一サーバー・単一管理者向けのセルフホスト型セキュアアクセス環境を構築します。' ;;

    en:install_intro_notice) printf 'Use only with servers and devices you own or are authorized to control, and confirm your use complies with applicable rules in your environment.' ;;
    zh:install_intro_notice) printf '请仅将其用于你自有或经授权控制的服务器与设备，并自行确认使用方式符合所在环境的适用规则。' ;;
    ja:install_intro_notice) printf '自ら所有する、または正当に管理権限を持つサーバーおよび端末にのみ使用し、利用環境で適用されるルールへの適合を各自で確認してください。' ;;

    en:install_panel_domain) printf 'Enter the panel domain, for example panel.example.com' ;;
    zh:install_panel_domain) printf '请输入面板域名，例如 panel.example.com' ;;
    ja:install_panel_domain) printf 'パネル用ドメインを入力してください。例: panel.example.com' ;;

    en:install_line_domain) printf 'Enter the line domain, for example line.example.com' ;;
    zh:install_line_domain) printf '请输入线路域名，例如 line.example.com' ;;
    ja:install_line_domain) printf '回線用ドメインを入力してください。例: line.example.com' ;;

    en:install_line_server_address) printf 'Enter the authorized client dial address, either a domain or the server public IP' ;;
    zh:install_line_server_address) printf '请输入授权设备实际连接地址，可填域名或服务器公网 IP' ;;
    ja:install_line_server_address) printf '認可済み端末が実際に接続する宛先を入力してください。ドメインまたはサーバーのグローバル IP を指定できます' ;;

    en:install_reality_target) printf 'Enter the REALITY target, for example www.cloudflare.com:443' ;;
    zh:install_reality_target) printf '请输入 REALITY 目标站，例如 www.cloudflare.com:443' ;;
    ja:install_reality_target) printf 'REALITY のターゲットを入力してください。例: www.cloudflare.com:443' ;;

    en:install_acme_email) printf 'Enter the email address used for certificate issuance' ;;
    zh:install_acme_email) printf '请输入用于证书申请的邮箱' ;;
    ja:install_acme_email) printf '証明書発行に使うメールアドレスを入力してください' ;;

    en:install_setup_ttl) printf 'Enter the one-time setup link TTL in minutes' ;;
    zh:install_setup_ttl) printf '请输入一次性 setup 链接有效期（分钟）' ;;
    ja:install_setup_ttl) printf 'ワンタイム setup リンクの有効期限（分）を入力してください' ;;

    en:install_invalid_control_chars) printf 'Invalid input: control characters or terminal escape sequences are not allowed.' ;;
    zh:install_invalid_control_chars) printf '输入无效：不允许控制字符或终端转义序列。' ;;
    ja:install_invalid_control_chars) printf '入力が無効です。制御文字や端末のエスケープシーケンスは使用できません。' ;;

    en:install_invalid_panel_domain) printf 'Invalid panel domain. Enter a hostname such as panel.example.com.' ;;
    zh:install_invalid_panel_domain) printf '面板域名格式无效，请输入类似 panel.example.com 的主机名。' ;;
    ja:install_invalid_panel_domain) printf 'パネル用ドメインの形式が正しくありません。panel.example.com のようなホスト名を入力してください。' ;;

    en:install_invalid_line_domain) printf 'Invalid line domain. Enter a hostname such as line.example.com.' ;;
    zh:install_invalid_line_domain) printf '线路域名格式无效，请输入类似 line.example.com 的主机名。' ;;
    ja:install_invalid_line_domain) printf '回線用ドメインの形式が正しくありません。line.example.com のようなホスト名を入力してください。' ;;

    en:install_invalid_line_server_address) printf 'Invalid connection address. Enter a hostname or IPv4 address.' ;;
    zh:install_invalid_line_server_address) printf '连接地址格式无效，请输入主机名或 IPv4 地址。' ;;
    ja:install_invalid_line_server_address) printf '接続先の形式が正しくありません。ホスト名または IPv4 アドレスを入力してください。' ;;

    en:install_invalid_reality_target) printf 'Invalid REALITY target. Use host:port, for example www.cloudflare.com:443.' ;;
    zh:install_invalid_reality_target) printf 'REALITY 目标格式无效，请使用 host:port，例如 www.cloudflare.com:443。' ;;
    ja:install_invalid_reality_target) printf 'REALITY ターゲットの形式が正しくありません。www.cloudflare.com:443 のように host:port 形式で入力してください。' ;;

    en:install_invalid_acme_email) printf 'Invalid email address format.' ;;
    zh:install_invalid_acme_email) printf '邮箱格式无效。' ;;
    ja:install_invalid_acme_email) printf 'メールアドレスの形式が正しくありません。' ;;

    en:install_invalid_setup_ttl) printf 'Invalid setup TTL. Enter an integer between 1 and 1440.' ;;
    zh:install_invalid_setup_ttl) printf 'setup 链接有效期格式无效，请输入 1 到 1440 之间的整数。' ;;
    ja:install_invalid_setup_ttl) printf 'setup リンク有効期限の形式が正しくありません。1 から 1440 の整数を入力してください。' ;;

    en:install_missing_manager) printf 'Missing required artifact: build/manager-linux-amd64' ;;
    zh:install_missing_manager) printf '缺少必需产物：build/manager-linux-amd64' ;;
    ja:install_missing_manager) printf '必須成果物がありません: build/manager-linux-amd64' ;;

    en:install_missing_web_dist) printf 'Missing required artifact: web/dist' ;;
    zh:install_missing_web_dist) printf '缺少必需产物：web/dist' ;;
    ja:install_missing_web_dist) printf '必須成果物がありません: web/dist' ;;

    en:install_missing_artifact_hint) printf 'Upload the full project with both prebuilt artifacts before running install.sh.' ;;
    zh:install_missing_artifact_hint) printf '请先上传包含这两个预构建产物的完整项目，再执行 install.sh。' ;;
    ja:install_missing_artifact_hint) printf 'install.sh を実行する前に、2 つの事前ビルド成果物を含む完全なプロジェクトをアップロードしてください。' ;;

    en:install_panel_url) printf 'Panel URL' ;;
    zh:install_panel_url) printf '面板地址' ;;
    ja:install_panel_url) printf 'パネル URL' ;;

    en:install_setup_url) printf 'Setup URL' ;;
    zh:install_setup_url) printf '初始化链接' ;;
    ja:install_setup_url) printf 'Setup URL' ;;

    en:install_setup_expires) printf 'Setup URL expires in' ;;
    zh:install_setup_expires) printf '初始化链接有效期' ;;
    ja:install_setup_expires) printf 'Setup URL の有効期限' ;;

    en:install_minutes) printf 'minutes' ;;
    zh:install_minutes) printf '分钟' ;;
    ja:install_minutes) printf '分' ;;

    en:install_login_url) printf 'Login URL' ;;
    zh:install_login_url) printf '登录地址' ;;
    ja:install_login_url) printf 'ログイン URL' ;;

    en:install_ticket_hint) printf 'View setup ticket' ;;
    zh:install_ticket_hint) printf '查看 setup ticket' ;;
    ja:install_ticket_hint) printf 'setup ticket を確認' ;;

    en:install_service_hint) printf 'Check service status' ;;
    zh:install_service_hint) printf '服务状态检查' ;;
    ja:install_service_hint) printf 'サービス状態確認' ;;

    en:install_finish_note) printf 'Review the generated setup link and initialize only the intended administrator account.' ;;
    zh:install_finish_note) printf '请核对生成的一次性 setup 链接，并仅为预期的管理员账号执行初始化。' ;;
    ja:install_finish_note) printf '生成されたワンタイム setup リンクを確認し、想定した管理者アカウントに対してのみ初期化を行ってください。' ;;

    en:update_intro_title) printf 'Solo VPN host updater' ;;
    zh:update_intro_title) printf 'Solo VPN 宿主机更新脚本' ;;
    ja:update_intro_title) printf 'Solo VPN ホスト更新スクリプト' ;;

    en:update_intro_body) printf 'This updater publishes a new release from the current repository checkout and switches the live current symlink.' ;;
    zh:update_intro_body) printf '该更新脚本会基于当前仓库内容发布新的 release，并切换线上 current 软链接。' ;;
    ja:update_intro_body) printf 'この更新スクリプトは、現在のリポジトリ内容から新しい release を発行し、稼働中の current シンボリックリンクを切り替えます。' ;;

    en:update_intro_notice) printf 'Run this on the target host after git pull, using a checkout that already contains build/manager-linux-amd64 and web/dist.' ;;
    zh:update_intro_notice) printf '请在目标服务器执行 git pull 后运行，并确保当前仓库已包含 build/manager-linux-amd64 与 web/dist。' ;;
    ja:update_intro_notice) printf '対象ホストで git pull 後に実行し、現在のリポジトリに build/manager-linux-amd64 と web/dist が含まれていることを確認してください。' ;;

    en:update_root_required) printf 'This script must be run as root (or via sudo).' ;;
    zh:update_root_required) printf '该脚本必须以 root 身份运行（或通过 sudo 执行）。' ;;
    ja:update_root_required) printf 'このスクリプトは root 権限（または sudo）で実行する必要があります。' ;;

    en:update_release_note) printf 'Published release directory' ;;
    zh:update_release_note) printf '已发布 release 目录' ;;
    ja:update_release_note) printf '発行済み release ディレクトリ' ;;

    en:update_current_note) printf 'Current symlink now points to' ;;
    zh:update_current_note) printf '当前 current 软链接已指向' ;;
    ja:update_current_note) printf 'current シンボリックリンクの新しい参照先' ;;

    en:update_restart_note) printf 'Restarted service' ;;
    zh:update_restart_note) printf '已重启服务' ;;
    ja:update_restart_note) printf '再起動したサービス' ;;

    en:check_blocked_path) printf 'Found old residue path: %s' "$2" ;;
    zh:check_blocked_path) printf '发现旧残留路径: %s' "$2" ;;
    ja:check_blocked_path) printf '古い残留パスが見つかりました: %s' "$2" ;;

    en:check_blocked_service) printf 'Found old systemd service: %s.service' "$2" ;;
    zh:check_blocked_service) printf '发现旧 systemd 服务: %s.service' "$2" ;;
    ja:check_blocked_service) printf '古い systemd サービスが見つかりました: %s.service' "$2" ;;

    en:check_blocked_port) printf 'Port %s is already occupied: %s' "$2" "$3" ;;
    zh:check_blocked_port) printf '端口 %s 已被占用: %s' "$2" "$3" ;;
    ja:check_blocked_port) printf 'ポート %s は既に使用中です: %s' "$2" "$3" ;;

    en:check_fail_hint) printf 'Please run cleanup.sh first, or continue with install.sh --clean after confirming.' ;;
    zh:check_fail_hint) printf '请先执行 cleanup.sh，或确认后使用 install.sh --clean 继续。' ;;
    ja:check_fail_hint) printf '先に cleanup.sh を実行するか、確認のうえ install.sh --clean を使って続行してください。' ;;

    en:check_pass) printf 'PASS: no blocking residue was found.' ;;
    zh:check_pass) printf 'PASS: 未发现会阻断安装的旧残留。' ;;
    ja:check_pass) printf 'PASS: インストールを妨げる残留物は見つかりませんでした。' ;;

    en:cleanup_finished) printf 'Cleanup finished. Backup saved to %s' "$2" ;;
    zh:cleanup_finished) printf '清理完成，备份保存在 %s' "$2" ;;
    ja:cleanup_finished) printf 'クリーンアップが完了しました。バックアップ保存先: %s' "$2" ;;

    en:reset_setup_usage) printf 'A new one-time setup ticket has been issued for administrator initialization.' ;;
    zh:reset_setup_usage) printf '已重新签发用于管理员初始化的一次性 setup ticket。' ;;
    ja:reset_setup_usage) printf '管理者初期化用の新しいワンタイム setup ticket を再発行しました。' ;;

    *) printf '%s' "$key" ;;
  esac
}
