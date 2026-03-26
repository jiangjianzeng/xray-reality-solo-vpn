import { useEffect, useMemo, useRef, useState, useTransition } from "react";
import { useForm, type UseFormRegisterReturn } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  useMutation,
  useQuery,
  useQueryClient
} from "@tanstack/react-query";
import {
  Activity,
  ChevronDown,
  Circle,
  Copy,
  KeyRound,
  LoaderCircle,
  LogOut,
  RefreshCw,
  Server,
  ShieldCheck,
  Trash2,
  UserPlus
} from "lucide-react";
import { api, type SetupStatus, type Client } from "@/lib/api";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Panel } from "@/components/ui/panel";
import { cn, formatLastSeen } from "@/lib/utils";
import {
  detectInitialLanguage,
  LANGUAGE_STORAGE_KEY,
  languageOptions,
  type Language,
  dateLocale
} from "@/lib/i18n";

type Notice = { tone: "success" | "error"; text: string } | null;

const copyErrorMessage: Record<Language, string> = {
  en: "Copy failed, please copy it manually",
  zh: "复制失败，请手动复制",
  ja: "コピーに失敗しました。手動でコピーしてください"
};

const statusMismatchMessage: Record<Language, string> = {
  en: "The new passwords do not match",
  zh: "两次输入的新密码不一致",
  ja: "新しいパスワードが一致しません"
};

const text = {
  en: {
    personalGateway: "Solo VPN",
    consoleTitle: "Solo VPN",
    consoleDescription:
      "A single-server self-hosted secure access workspace focused on setup, sign-in, client management, and runtime status.",
    missingRuntime: "Missing runtime configuration",
    loading: "Loading Solo VPN workspace...",
    auth: {
      setupTitle: "Create the admin account",
      setupDescription: "This is a one-time setup entry used only for the first administrator.",
      setupSubmit: "Initialize and sign in",
      lockedTitle: "Setup is locked",
      lockedDescription: "Public setup is disabled. Use the one-time setup link printed after installation.",
      setupExpiresAt: (value: string) => `The current setup entry expires at: ${value}`,
      loginTitle: "Sign in to the console",
      loginDescription: "After signing in, you can manage clients, sync Xray, and inspect runtime status.",
      adminUsername: "Admin username",
      adminPassword: "Admin password",
      account: "Username",
      password: "Password"
    },
    notices: {
      setupDone: "Initialization completed",
      loginDone: "Signed in successfully",
      logoutDone: "Signed out",
      syncDone: "Synchronization completed",
      clientCreated: "Client created",
      clientUpdated: "Client updated",
      clientDeleted: "Client deleted",
      passwordUpdated: "Password updated",
      passwordEndpointMissing: "The backend has not enabled /api/auth/password yet; the main workflow is unaffected.",
      copiedVless: "Copied VLESS link",
      copiedSubscription: "Copied Clash subscription URL"
    },
    dashboard: {
      workspace: "Solo VPN",
      title: "Personal Access Console",
      syncXray: "Sync Xray",
      logout: "Sign out",
      accountMenu: "Account",
      changePassword: "Change password",
      hidePassword: "Hide password form",
      xrayStatus: "Xray status",
      totalRx: "Total download",
      totalTx: "Total upload",
      activeClients: "Active clients",
      createClient: "Create client",
      clientName: "Client name",
      createAndSync: "Create and sync",
      panelUrl: "Panel URL",
      lineDomain: "Line Domain",
      serverAddress: "Server Address",
      realityTarget: "Reality Target",
      clientList: "Clients",
      items: (count: number) => `${count} items`,
      loading: "Loading...",
      empty: "No clients yet. Create the first line first.",
      passwordSettings: "Password settings",
      currentPassword: "Current password",
      newPassword: "New password",
      confirmPassword: "Confirm new password",
      updatePassword: "Update password"
    },
    client: {
      enabled: "Enabled",
      disabled: "Disabled",
      rx: "Download",
      tx: "Upload",
      downRate: "Download rate",
      upRate: "Upload rate",
      lastSeen: "Last active",
      clash: "Clash",
      resetToken: "Reset token",
      disable: "Disable",
      enable: "Enable",
      remove: "Delete"
    }
  },
  zh: {
    personalGateway: "Solo VPN",
    consoleTitle: "Solo VPN",
    consoleDescription: "单机自建安全接入工作台。只保留初始化、登录、客户端管理与运行状态，不做面向公众的共享接入平台或多租户分发系统。",
    missingRuntime: "缺少运行时配置",
    loading: "正在加载 Solo VPN 工作台...",
    auth: {
      setupTitle: "设置管理员账号密码",
      setupDescription: "这是一次性 setup 入口，仅用于创建首个管理员账号。",
      setupSubmit: "初始化并登录",
      lockedTitle: "初始化已锁定",
      lockedDescription: "当前不开放公开初始化。请使用安装完成时输出的一次性 setup 链接。",
      setupExpiresAt: (value: string) => `当前 setup 入口有效期至: ${value}`,
      loginTitle: "登录管理界面",
      loginDescription: "登录后可管理客户端、同步 Xray、查看运行状态。",
      adminUsername: "管理员账号",
      adminPassword: "管理员密码",
      account: "账号",
      password: "密码"
    },
    notices: {
      setupDone: "初始化完成",
      loginDone: "登录成功",
      logoutDone: "已退出登录",
      syncDone: "已完成同步",
      clientCreated: "客户端已创建",
      clientUpdated: "客户端已更新",
      clientDeleted: "客户端已删除",
      passwordUpdated: "密码已更新",
      passwordEndpointMissing: "后端尚未开放 /api/auth/password，主流程不受影响",
      copiedVless: "已复制 VLESS 链接",
      copiedSubscription: "已复制 Clash 订阅地址"
    },
    dashboard: {
      workspace: "Solo VPN",
      title: "个人接入控制台",
      syncXray: "同步 Xray",
      logout: "退出",
      accountMenu: "账号",
      changePassword: "修改密码",
      hidePassword: "收起密码区",
      xrayStatus: "Xray 状态",
      totalRx: "总下行",
      totalTx: "总上行",
      activeClients: "活跃客户端",
      createClient: "创建客户端",
      clientName: "客户端名称",
      createAndSync: "创建并同步",
      panelUrl: "面板地址",
      lineDomain: "线路域名",
      serverAddress: "服务器地址",
      realityTarget: "Reality 目标",
      clientList: "客户端列表",
      items: (count: number) => `${count} 项`,
      loading: "加载中...",
      empty: "还没有客户端，先创建第一条线路。",
      passwordSettings: "账号密码设置",
      currentPassword: "当前密码",
      newPassword: "新密码",
      confirmPassword: "确认新密码",
      updatePassword: "更新密码"
    },
    client: {
      enabled: "启用中",
      disabled: "已停用",
      rx: "下行",
      tx: "上行",
      downRate: "下行速率",
      upRate: "上行速率",
      lastSeen: "最后活跃",
      clash: "Clash",
      resetToken: "重置 Token",
      disable: "停用",
      enable: "启用",
      remove: "删除"
    }
  },
  ja: {
    personalGateway: "Solo VPN",
    consoleTitle: "Solo VPN",
    consoleDescription: "単一サーバー向けのセルフホスト型セキュアアクセス管理ワークスペースです。初期化、ログイン、クライアント管理、ランタイム状態に絞っています。",
    missingRuntime: "不足しているランタイム設定",
    loading: "Solo VPN ワークスペースを読み込み中...",
    auth: {
      setupTitle: "管理者アカウントを作成",
      setupDescription: "このワンタイム setup エントリは最初の管理者作成にのみ使われます。",
      setupSubmit: "初期化してログイン",
      lockedTitle: "セットアップはロックされています",
      lockedDescription: "公開 setup は無効です。インストール完了時に表示されたワンタイム setup リンクを使用してください。",
      setupExpiresAt: (value: string) => `現在の setup エントリの有効期限: ${value}`,
      loginTitle: "管理コンソールにログイン",
      loginDescription: "ログイン後にクライアント管理、Xray 同期、ランタイム状態の確認ができます。",
      adminUsername: "管理者ユーザー名",
      adminPassword: "管理者パスワード",
      account: "ユーザー名",
      password: "パスワード"
    },
    notices: {
      setupDone: "初期化が完了しました",
      loginDone: "ログインしました",
      logoutDone: "ログアウトしました",
      syncDone: "同期が完了しました",
      clientCreated: "クライアントを作成しました",
      clientUpdated: "クライアントを更新しました",
      clientDeleted: "クライアントを削除しました",
      passwordUpdated: "パスワードを更新しました",
      passwordEndpointMissing: "バックエンドで /api/auth/password はまだ有効化されていません。主なワークフローには影響しません。",
      copiedVless: "VLESS リンクをコピーしました",
      copiedSubscription: "Clash 購読 URL をコピーしました"
    },
    dashboard: {
      workspace: "Solo VPN",
      title: "個人回線コンソール",
      syncXray: "Xray を同期",
      logout: "ログアウト",
      accountMenu: "アカウント",
      changePassword: "パスワード変更",
      hidePassword: "パスワード欄を閉じる",
      xrayStatus: "Xray 状態",
      totalRx: "総ダウンロード",
      totalTx: "総アップロード",
      activeClients: "有効クライアント",
      createClient: "クライアントを作成",
      clientName: "クライアント名",
      createAndSync: "作成して同期",
      panelUrl: "パネル URL",
      lineDomain: "回線ドメイン",
      serverAddress: "サーバー接続先",
      realityTarget: "Reality ターゲット",
      clientList: "クライアント一覧",
      items: (count: number) => `${count} 件`,
      loading: "読み込み中...",
      empty: "クライアントがまだありません。最初の回線を作成してください。",
      passwordSettings: "パスワード設定",
      currentPassword: "現在のパスワード",
      newPassword: "新しいパスワード",
      confirmPassword: "新しいパスワード（確認）",
      updatePassword: "パスワードを更新"
    },
    client: {
      enabled: "有効",
      disabled: "停止中",
      rx: "ダウンロード",
      tx: "アップロード",
      downRate: "ダウンロード速度",
      upRate: "アップロード速度",
      lastSeen: "最終利用",
      clash: "Clash",
      resetToken: "トークン再発行",
      disable: "停止",
      enable: "有効化",
      remove: "削除"
    }
  }
} as const;

const setupSchema = z.object({
  username: z.string().min(3).max(32).regex(/^[a-zA-Z0-9_.-]+$/),
  password: z.string().min(8).max(128)
});

const loginSchema = setupSchema;

const clientSchema = z.object({
  name: z.string().trim().min(2).max(48)
});

const passwordSchema = z
  .object({
    currentPassword: z.string().min(8),
    newPassword: z.string().min(8),
    confirmPassword: z.string().min(8)
  })
  .refine((v) => v.newPassword === v.confirmPassword, {
    path: ["confirmPassword"],
    message: statusMismatchMessage.en
  });

function statusTone(state: string | null | undefined): "neutral" | "success" | "warn" {
  const normalized = String(state ?? "").toLowerCase();
  if (normalized === "running" || normalized === "ok") {
    return "success";
  }
  if (normalized === "stopped" || normalized === "missing" || normalized === "error") {
    return "warn";
  }
  return "neutral";
}

function useNotice() {
  const [notice, setNotice] = useState<Notice>(null);

  useEffect(() => {
    if (!notice) {
      return;
    }
    const timer = setTimeout(() => setNotice(null), 2800);
    return () => clearTimeout(timer);
  }, [notice]);

  return {
    notice,
    pushSuccess(text: string) {
      setNotice({ tone: "success", text });
    },
    pushError(text: string) {
      setNotice({ tone: "error", text });
    }
  };
}

function App() {
  const queryClient = useQueryClient();
  const noticeApi = useNotice();
  const [isPending, startTransition] = useTransition();
  const [language, setLanguage] = useState<Language>(() => detectInitialLanguage());
  const copyError = copyErrorMessage[language];
  const t = text[language];

  const setupQuery = useQuery({
    queryKey: ["setup"],
    queryFn: api.setupStatus
  });

  const sessionQuery = useQuery({
    queryKey: ["session"],
    queryFn: api.session,
    enabled: Boolean(setupQuery.data?.initialized)
  });

  const setupForm = useForm<z.infer<typeof setupSchema>>({
    resolver: zodResolver(setupSchema),
    defaultValues: { username: "", password: "" }
  });

  const loginForm = useForm<z.infer<typeof loginSchema>>({
    resolver: zodResolver(loginSchema),
    defaultValues: { username: "", password: "" }
  });

  const setupMutation = useMutation({
    mutationFn: api.setupInit,
    onSuccess: () => {
      noticeApi.pushSuccess(t.notices.setupDone);
      setupForm.reset();
      startTransition(() => {
        queryClient.invalidateQueries();
      });
    },
    onError: (error: Error) => noticeApi.pushError(error.message)
  });

  const loginMutation = useMutation({
    mutationFn: api.login,
    onSuccess: () => {
      noticeApi.pushSuccess(t.notices.loginDone);
      loginForm.reset();
      startTransition(() => {
        queryClient.invalidateQueries({ queryKey: ["session"] });
      });
    },
    onError: (error: Error) => noticeApi.pushError(error.message)
  });

  const loading = setupQuery.isLoading || (setupQuery.data?.initialized && sessionQuery.isLoading);
  const initialized = setupQuery.data?.initialized;
  const authenticated = sessionQuery.data?.authenticated;
  const setupAuthorized = setupQuery.data?.setupAuthorized;
  const refreshIntervalMs = Math.max(1000, setupQuery.data?.refreshIntervalMs ?? 5000);

  useEffect(() => {
    window.localStorage.setItem(LANGUAGE_STORAGE_KEY, language);
    document.documentElement.lang = language === "zh" ? "zh-CN" : language === "ja" ? "ja-JP" : "en";
    document.title = t.consoleTitle;
  }, [language, t.consoleTitle]);

  return (
    <div className="relative min-h-screen">
      <div className="mesh-overlay pointer-events-none absolute inset-0 opacity-55" />
      {noticeApi.notice ? (
        <div
          className={cn(
            "fixed left-1/2 top-4 z-50 w-[min(92vw,32rem)] -translate-x-1/2 rounded-xl border px-4 py-2 text-sm shadow-lg backdrop-blur",
            noticeApi.notice.tone === "success"
              ? "border-primary/30 bg-primary/10 text-primary"
              : "border-danger/30 bg-danger/10 text-danger"
          )}
        >
          {noticeApi.notice.text}
        </div>
      ) : null}

      {loading ? (
        <LoadingView language={language} onLanguageChange={setLanguage} />
      ) : initialized && authenticated ? (
        <DashboardView
          language={language}
          onLanguageChange={setLanguage}
          refreshIntervalMs={refreshIntervalMs}
          setupStatus={setupQuery.data!}
          username={sessionQuery.data?.admin?.username ?? "admin"}
          disabled={isPending}
          onNotifySuccess={noticeApi.pushSuccess}
          onNotifyError={noticeApi.pushError}
        />
      ) : (
        <AuthShell
          language={language}
          onLanguageChange={setLanguage}
          setupStatus={setupQuery.data ?? null}
          mode={initialized ? "login" : setupAuthorized ? "setup" : "locked"}
          setupForm={setupForm}
          loginForm={loginForm}
          setupPending={setupMutation.isPending}
          loginPending={loginMutation.isPending}
          onSetupSubmit={setupMutation.mutate}
          onLoginSubmit={loginMutation.mutate}
        />
      )}
    </div>
  );
}

function LanguageSwitcher({
  language,
  onChange
}: {
  language: Language;
  onChange: (language: Language) => void;
}) {
  return (
    <div className="flex items-center rounded-xl border border-border/70 bg-background/70 p-1">
      {languageOptions.map((option) => (
        <Button
          key={option.value}
          type="button"
          size="sm"
          variant={language === option.value ? "secondary" : "ghost"}
          className="h-8 px-2.5"
          onClick={() => onChange(option.value)}
        >
          {option.label}
        </Button>
      ))}
    </div>
  );
}

function LoadingView({
  language,
  onLanguageChange
}: {
  language: Language;
  onLanguageChange: (language: Language) => void;
}) {
  const t = text[language];
  return (
    <div className="mx-auto flex min-h-screen max-w-5xl items-center justify-center px-6 py-12">
      <div className="animate-slide-up text-center">
        <div className="mb-4 flex justify-center">
          <LanguageSwitcher language={language} onChange={onLanguageChange} />
        </div>
        <LoaderCircle className="mx-auto mb-3 size-8 animate-spin text-primary" />
        <p className="text-sm text-muted-foreground">{t.loading}</p>
      </div>
    </div>
  );
}

type AuthShellProps = {
  language: Language;
  onLanguageChange: (language: Language) => void;
  setupStatus: SetupStatus | null;
  mode: "setup" | "locked" | "login";
  setupForm: ReturnType<typeof useForm<z.infer<typeof setupSchema>>>;
  loginForm: ReturnType<typeof useForm<z.infer<typeof loginSchema>>>;
  setupPending: boolean;
  loginPending: boolean;
  onSetupSubmit: (input: z.infer<typeof setupSchema>) => void;
  onLoginSubmit: (input: z.infer<typeof loginSchema>) => void;
};

function AuthShell(props: AuthShellProps) {
  const {
    language,
    onLanguageChange,
    setupStatus,
    mode,
    setupForm,
    loginForm,
    setupPending,
    loginPending,
    onSetupSubmit,
    onLoginSubmit
  } = props;
  const t = text[language];

  return (
    <main className="mx-auto flex min-h-screen w-full max-w-6xl items-center px-4 py-7 sm:px-6">
      <div className="grid w-full gap-6 md:grid-cols-[1.06fr_0.94fr]">
        <section className="animate-slide-up rounded-3xl border border-border/70 bg-card/55 p-7 backdrop-blur-xl">
          <div className="mb-5 flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div>
              <p className="mb-3 inline-flex rounded-full bg-primary/12 px-3 py-1 text-xs font-semibold uppercase tracking-[0.14em] text-primary">
                {t.personalGateway}
              </p>
              <h1 className="text-3xl font-semibold leading-tight sm:text-4xl">{t.consoleTitle}</h1>
            </div>
            <LanguageSwitcher language={language} onChange={onLanguageChange} />
          </div>
          <p className="mt-3 max-w-xl text-sm text-muted-foreground sm:text-base">
            {t.consoleDescription}
          </p>
          <div className="mt-8 grid gap-3 text-sm sm:grid-cols-2">
            <Panel className="p-4">
              <p className="text-xs uppercase tracking-[0.12em] text-muted-foreground">Panel Domain</p>
              <p className="mt-2 font-mono text-xs sm:text-sm">{setupStatus?.panelDomain ?? "panel.example.com"}</p>
            </Panel>
            <Panel className="p-4">
              <p className="text-xs uppercase tracking-[0.12em] text-muted-foreground">Line Domain</p>
              <p className="mt-2 font-mono text-xs sm:text-sm">{setupStatus?.lineDomain ?? "line.example.com"}</p>
            </Panel>
          </div>
          {setupStatus?.missingRuntime?.length ? (
            <p className="mt-4 rounded-xl border border-danger/30 bg-danger/10 p-3 text-xs text-danger">
              {t.missingRuntime}: {setupStatus.missingRuntime.join(", ")}
            </p>
          ) : null}
        </section>

        <Panel className="animate-slide-up p-6 [animation-delay:120ms]">
          {mode === "setup" ? (
            <form className="space-y-4" onSubmit={setupForm.handleSubmit(onSetupSubmit)}>
              <div className="mb-1">
                <h2 className="text-xl font-semibold">{t.auth.setupTitle}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{t.auth.setupDescription}</p>
              </div>
              <Field control={setupForm.register("username")} label={t.auth.adminUsername} placeholder="admin" />
              <Field
                control={setupForm.register("password")}
                label={t.auth.adminPassword}
                placeholder={language === "en" ? "At least 8 characters" : language === "ja" ? "8 文字以上" : "至少 8 位"}
                type="password"
              />
              <Button disabled={setupPending} className="w-full">
                {setupPending ? <LoaderCircle className="mr-2 size-4 animate-spin" /> : null}
                {t.auth.setupSubmit}
              </Button>
            </form>
          ) : mode === "locked" ? (
            <div className="space-y-4">
              <div className="mb-1">
                <h2 className="text-xl font-semibold">{t.auth.lockedTitle}</h2>
                <p className="mt-1 text-sm text-muted-foreground">
                  {t.auth.lockedDescription}
                </p>
              </div>
              {setupStatus?.setupExpiresAt ? (
                <p className="rounded-xl bg-muted/50 p-3 text-xs text-muted-foreground">
                  {t.auth.setupExpiresAt(new Date(setupStatus.setupExpiresAt).toLocaleString(dateLocale(language)))}
                </p>
              ) : null}
            </div>
          ) : (
            <form className="space-y-4" onSubmit={loginForm.handleSubmit(onLoginSubmit)}>
              <div className="mb-1">
                <h2 className="text-xl font-semibold">{t.auth.loginTitle}</h2>
                <p className="mt-1 text-sm text-muted-foreground">{t.auth.loginDescription}</p>
              </div>
              <Field control={loginForm.register("username")} label={t.auth.account} placeholder="admin" />
              <Field control={loginForm.register("password")} label={t.auth.password} placeholder="******" type="password" />
              <Button disabled={loginPending} className="w-full">
                {loginPending ? <LoaderCircle className="mr-2 size-4 animate-spin" /> : null}
                {language === "en" ? "Sign in" : language === "ja" ? "ログイン" : "登录"}
              </Button>
            </form>
          )}
        </Panel>
      </div>
    </main>
  );
}

function Field({
  control,
  label,
  placeholder,
  type = "text"
}: {
  control: UseFormRegisterReturn;
  label: string;
  placeholder: string;
  type?: string;
}) {
  return (
    <label className="block space-y-2">
      <Label>{label}</Label>
      <Input type={type} placeholder={placeholder} {...control} autoComplete="off" />
    </label>
  );
}

type DashboardProps = {
  language: Language;
  onLanguageChange: (language: Language) => void;
  refreshIntervalMs: number;
  setupStatus: SetupStatus;
  username: string;
  disabled: boolean;
  onNotifySuccess: (text: string) => void;
  onNotifyError: (text: string) => void;
};

function DashboardView({
  language,
  onLanguageChange,
  refreshIntervalMs,
  setupStatus,
  username,
  disabled,
  onNotifySuccess,
  onNotifyError
}: DashboardProps) {
  const queryClient = useQueryClient();
  const t = text[language];
  const passwordPanelRef = useRef<HTMLDivElement | null>(null);
  const accountMenuRef = useRef<HTMLDivElement | null>(null);
  const [showPasswordPanel, setShowPasswordPanel] = useState(false);
  const [accountMenuOpen, setAccountMenuOpen] = useState(false);
  const clientForm = useForm<z.infer<typeof clientSchema>>({
    resolver: zodResolver(clientSchema),
    defaultValues: { name: "" }
  });
  const passwordForm = useForm<z.infer<typeof passwordSchema>>({
    resolver: zodResolver(passwordSchema),
    defaultValues: {
      currentPassword: "",
      newPassword: "",
      confirmPassword: ""
    }
  });

  const dashboardQuery = useQuery({
    queryKey: ["dashboard"],
    queryFn: api.dashboard,
    refetchInterval: refreshIntervalMs
  });

  const clientsQuery = useQuery({
    queryKey: ["clients"],
    queryFn: api.clients,
    refetchInterval: refreshIntervalMs
  });

  const logoutMutation = useMutation({
    mutationFn: api.logout,
    onSuccess: () => {
      onNotifySuccess(t.notices.logoutDone);
      queryClient.invalidateQueries();
    },
    onError: (error: Error) => onNotifyError(error.message)
  });

  const createMutation = useMutation({
    mutationFn: api.createClient,
    onSuccess: () => {
      onNotifySuccess(t.notices.clientCreated);
      clientForm.reset();
      queryClient.invalidateQueries({ queryKey: ["clients"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (error: Error) => onNotifyError(error.message)
  });

  const patchMutation = useMutation({
    mutationFn: ({ id, payload }: { id: number; payload: { enabled?: boolean; rotateToken?: boolean } }) =>
      api.patchClient(id, payload),
    onSuccess: () => {
      onNotifySuccess(t.notices.clientUpdated);
      queryClient.invalidateQueries({ queryKey: ["clients"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (error: Error) => onNotifyError(error.message)
  });

  const deleteMutation = useMutation({
    mutationFn: api.deleteClient,
    onSuccess: () => {
      onNotifySuccess(t.notices.clientDeleted);
      queryClient.invalidateQueries({ queryKey: ["clients"] });
      queryClient.invalidateQueries({ queryKey: ["dashboard"] });
    },
    onError: (error: Error) => onNotifyError(error.message)
  });

  const passwordMutation = useMutation({
    mutationFn: api.updatePassword,
    onSuccess: () => {
      onNotifySuccess(t.notices.passwordUpdated);
      passwordForm.reset();
    },
    onError: (error: Error) => {
      if (error.message.includes("404")) {
        onNotifyError(t.notices.passwordEndpointMissing);
        return;
      }
      onNotifyError(error.message);
    }
  });

  const summary = dashboardQuery.data?.summary ?? {
    panelBaseUrl: setupStatus.panelBaseUrl,
    panelDomain: setupStatus.panelDomain,
    lineDomain: setupStatus.lineDomain,
    lineServerAddress: setupStatus.lineServerAddress,
    xrayTarget: setupStatus.xrayTarget,
    serviceState: setupStatus.serviceState,
    refreshIntervalMs,
    clientCount: setupStatus.clientCount,
    activeClientCount: setupStatus.activeClientCount
  };

  const clients = useMemo(() => clientsQuery.data?.clients ?? [], [clientsQuery.data?.clients]);

  function openPasswordPanel() {
    setShowPasswordPanel(true);
    setAccountMenuOpen(false);
    requestAnimationFrame(() => {
      passwordPanelRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
    });
  }

  useEffect(() => {
    function handlePointerDown(event: MouseEvent) {
      if (!accountMenuRef.current?.contains(event.target as Node)) {
        setAccountMenuOpen(false);
      }
    }

    if (!accountMenuOpen) {
      return;
    }

    document.addEventListener("mousedown", handlePointerDown);
    return () => document.removeEventListener("mousedown", handlePointerDown);
  }, [accountMenuOpen]);

  return (
    <div className="mx-auto flex min-h-screen w-full max-w-7xl flex-col gap-4 px-3 py-4 sm:px-5 sm:py-6">
      <header className="animate-slide-up relative z-30 overflow-visible rounded-2xl border border-border/70 bg-card/70 p-4 backdrop-blur">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div>
            <p className="text-xs uppercase tracking-[0.14em] text-muted-foreground">{t.dashboard.workspace}</p>
            <h1 className="mt-1 text-xl font-semibold sm:text-2xl">{t.dashboard.title}</h1>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge tone={statusTone(dashboardQuery.data?.status.state ?? summary.serviceState)} className="gap-2">
              <Circle
                className={cn(
                  "size-2 fill-current stroke-none",
                  statusTone(dashboardQuery.data?.status.state ?? summary.serviceState) === "success"
                    ? "text-primary"
                    : "text-muted-foreground"
                )}
              />
              {dashboardQuery.data?.status.state ?? summary.serviceState}
            </Badge>
            <div ref={accountMenuRef} className="relative z-40">
              <Button
                variant="ghost"
                className="gap-2"
                disabled={disabled}
                onClick={() => setAccountMenuOpen((open) => !open)}
              >
                {username}
                <ChevronDown className={cn("size-4 transition-transform", accountMenuOpen ? "rotate-180" : "")} />
              </Button>
              {accountMenuOpen ? (
              <div className="absolute right-0 top-[calc(100%+0.75rem)] z-[80] min-w-64 rounded-2xl border border-border/80 bg-card/95 p-3 shadow-[0_22px_44px_rgba(15,23,42,0.14)] backdrop-blur">
                <div className="rounded-xl border border-border/70 bg-background/80 p-1">
                  <div className="grid grid-cols-3 gap-1">
                    {languageOptions.map((option) => (
                      <button
                        key={option.value}
                        type="button"
                        className={cn(
                          "flex h-9 items-center justify-center rounded-lg px-2 text-sm font-medium transition",
                          language === option.value
                            ? "border border-border bg-card text-foreground shadow-sm"
                            : "text-muted-foreground hover:bg-muted hover:text-foreground"
                        )}
                        onClick={() => onLanguageChange(option.value)}
                      >
                        <span className="whitespace-nowrap">{option.label}</span>
                      </button>
                    ))}
                  </div>
                </div>
                <div className="my-3 border-t border-border/70" />
                <button
                  type="button"
                  className="flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-medium hover:bg-muted"
                  onClick={openPasswordPanel}
                >
                  <KeyRound className="size-4 shrink-0 text-primary" />
                  <span className="whitespace-nowrap">
                    {showPasswordPanel ? t.dashboard.hidePassword : t.dashboard.changePassword}
                  </span>
                </button>
                <button
                  type="button"
                  className="mt-1 flex w-full items-center gap-3 rounded-xl px-3 py-2.5 text-left text-sm font-medium hover:bg-muted"
                  onClick={() => {
                    setAccountMenuOpen(false);
                    logoutMutation.mutate();
                  }}
                  disabled={logoutMutation.isPending || disabled}
                >
                  <LogOut className="size-4 shrink-0" />
                  <span className="whitespace-nowrap">{t.dashboard.logout}</span>
                </button>
              </div>
              ) : null}
            </div>
          </div>
        </div>
      </header>

      <section className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
        <StatTile title={t.dashboard.xrayStatus} value={dashboardQuery.data?.status.state ?? summary.serviceState} icon={Server} />
        <StatTile title={t.dashboard.totalRx} value={dashboardQuery.data?.traffic.totalRxHuman ?? setupStatus.totalRxHuman} icon={Activity} />
        <StatTile title={t.dashboard.totalTx} value={dashboardQuery.data?.traffic.totalTxHuman ?? setupStatus.totalTxHuman} icon={Activity} />
        <StatTile
          title={t.dashboard.activeClients}
          value={`${summary.activeClientCount} / ${summary.clientCount}`}
          icon={ShieldCheck}
        />
      </section>

      <section className="grid gap-3 lg:grid-cols-[1.2fr_1.8fr]">
        <Panel className="animate-slide-up space-y-4">
          <h2 className="text-base font-semibold">{t.dashboard.createClient}</h2>
          <form
            className="space-y-3"
            onSubmit={clientForm.handleSubmit((values) => createMutation.mutate(values))}
          >
            <Field control={clientForm.register("name")} label={t.dashboard.clientName} placeholder={language === "en" ? "For example: iPhone 16 Pro" : language === "ja" ? "例: iPhone 16 Pro" : "例如: iPhone 16 Pro"} />
            <Button className="w-full" disabled={createMutation.isPending}>
              {createMutation.isPending ? (
                <LoaderCircle className="mr-2 size-4 animate-spin" />
              ) : (
                <UserPlus className="mr-2 size-4" />
              )}
              {t.dashboard.createAndSync}
            </Button>
          </form>
          <div className="rounded-xl bg-muted/50 p-3 text-xs leading-6 text-muted-foreground">
            <p>{t.dashboard.panelUrl}: <span className="font-mono">{summary.panelBaseUrl}</span></p>
            <p>{t.dashboard.lineDomain}: <span className="font-mono">{summary.lineDomain}</span></p>
            <p>{t.dashboard.serverAddress}: <span className="font-mono">{summary.lineServerAddress}</span></p>
            <p>{t.dashboard.realityTarget}: <span className="font-mono">{summary.xrayTarget}</span></p>
          </div>
        </Panel>

        <Panel className="animate-slide-up [animation-delay:80ms]">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-base font-semibold">{t.dashboard.clientList}</h2>
            <span className="text-xs text-muted-foreground">{t.dashboard.items(clients.length)}</span>
          </div>
          {clientsQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">{t.dashboard.loading}</p>
          ) : clients.length === 0 ? (
            <p className="rounded-xl border border-dashed border-border p-4 text-sm text-muted-foreground">
              {t.dashboard.empty}
            </p>
          ) : (
            <div className="scroll-soft max-h-[45vh] space-y-2 overflow-auto pr-1">
              {clients.map((client) => (
                <ClientRow
                  key={client.id}
                  language={language}
                  client={client}
                  busy={patchMutation.isPending || deleteMutation.isPending}
                  onNotifySuccess={onNotifySuccess}
                  onNotifyError={onNotifyError}
                  onToggle={(nextEnabled) =>
                    patchMutation.mutate({
                      id: client.id,
                      payload: { enabled: nextEnabled }
                    })
                  }
                  onRotate={() =>
                    patchMutation.mutate({
                      id: client.id,
                      payload: { rotateToken: true }
                    })
                  }
                  onDelete={() => deleteMutation.mutate(client.id)}
                />
              ))}
            </div>
          )}
        </Panel>
      </section>

      {showPasswordPanel ? (
      <Panel ref={passwordPanelRef} className="animate-slide-up [animation-delay:120ms]">
        <div className="mb-3 flex items-center gap-2">
          <KeyRound className="size-4 text-primary" />
          <h2 className="text-base font-semibold">{t.dashboard.passwordSettings}</h2>
          <Button variant="ghost" size="sm" className="ml-auto" onClick={() => setShowPasswordPanel(false)}>
            {t.dashboard.hidePassword}
          </Button>
        </div>
        <form
          className="grid gap-3 lg:grid-cols-3"
          onSubmit={passwordForm.handleSubmit((values) =>
            passwordMutation.mutate({
              currentPassword: values.currentPassword,
              newPassword: values.newPassword
            })
          )}
        >
          <Field
            control={passwordForm.register("currentPassword")}
            label={t.dashboard.currentPassword}
            placeholder={t.dashboard.currentPassword}
            type="password"
          />
          <Field control={passwordForm.register("newPassword")} label={t.dashboard.newPassword} placeholder={language === "en" ? "At least 8 characters" : language === "ja" ? "8 文字以上" : "至少 8 位"} type="password" />
          <Field
            control={passwordForm.register("confirmPassword")}
            label={t.dashboard.confirmPassword}
            placeholder={language === "en" ? "Enter again" : language === "ja" ? "もう一度入力" : "再次输入"}
            type="password"
          />
          <div className="sm:col-span-3">
            <Button variant="secondary" disabled={passwordMutation.isPending}>
              {passwordMutation.isPending ? <LoaderCircle className="mr-2 size-4 animate-spin" /> : null}
              {t.dashboard.updatePassword}
            </Button>
          </div>
        </form>
      </Panel>
      ) : null}
    </div>
  );
}

function StatTile({
  title,
  value,
  icon: Icon
}: {
  title: string;
  value: string;
  icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <Panel className="animate-slide-up p-4">
      <div className="mb-2 flex items-center justify-between">
        <p className="text-xs uppercase tracking-[0.12em] text-muted-foreground">{title}</p>
        <Icon className="size-4 text-primary" />
      </div>
      <p className="font-mono text-sm sm:text-base">{value}</p>
    </Panel>
  );
}

type ClientRowProps = {
  language: Language;
  client: Client;
  busy: boolean;
  onNotifySuccess: (text: string) => void;
  onNotifyError: (text: string) => void;
  onToggle: (enabled: boolean) => void;
  onRotate: () => void;
  onDelete: () => void;
};

function ClientRow({
  language,
  client,
  busy,
  onNotifySuccess,
  onNotifyError,
  onToggle,
  onRotate,
  onDelete
}: ClientRowProps) {
  const t = text[language];
  async function copyText(text: string, message: string) {
    try {
      await navigator.clipboard.writeText(text);
      onNotifySuccess(message);
    } catch {
      onNotifyError(copyErrorMessage[language]);
    }
  }

  return (
    <article className="rounded-2xl border border-border/70 bg-background/80 p-4 shadow-[0_1px_0_rgba(15,23,42,0.02)]">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div>
          <p className="text-base font-semibold tracking-tight">{client.name}</p>
          <p className="mt-0.5 font-mono text-xs text-muted-foreground">{client.slug}</p>
        </div>
        <Badge tone={client.enabled ? "success" : "warn"} className="px-2 py-0 text-[11px] tracking-[0.08em]">
          {client.enabled ? t.client.enabled : t.client.disabled}
        </Badge>
      </div>

      <div className="mt-4 grid gap-3 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-end">
        <dl className="grid grid-cols-2 gap-x-6 gap-y-2 text-sm text-muted-foreground xl:grid-cols-3">
          <div>
            <dt className="text-xs uppercase tracking-[0.08em] text-muted-foreground/80">{t.client.rx}</dt>
            <dd className="mt-1 text-base text-foreground/80">{client.rxHuman}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-[0.08em] text-muted-foreground/80">{t.client.tx}</dt>
            <dd className="mt-1 text-base text-foreground/80">{client.txHuman}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-[0.08em] text-muted-foreground/80">{t.client.downRate}</dt>
            <dd className="mt-1 text-base text-foreground/80">{client.rxBpsHuman}</dd>
          </div>
          <div>
            <dt className="text-xs uppercase tracking-[0.08em] text-muted-foreground/80">{t.client.upRate}</dt>
            <dd className="mt-1 text-base text-foreground/80">{client.txBpsHuman}</dd>
          </div>
          <div className="col-span-2 xl:col-span-2">
            <dt className="text-xs uppercase tracking-[0.08em] text-muted-foreground/80">{t.client.lastSeen}</dt>
            <dd className="mt-1 whitespace-nowrap text-base text-foreground/80">{formatLastSeen(client.last_seen_at, language)}</dd>
          </div>
        </dl>

        <div className="flex items-center justify-end lg:pb-0.5">
          <Button size="sm" variant="ghost" disabled={busy} onClick={() => onToggle(!client.enabled)}>
            {client.enabled ? t.client.disable : t.client.enable}
          </Button>
        </div>
      </div>

      <div className="mt-4 border-t border-border/60 pt-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="secondary" disabled={busy} onClick={() => copyText(client.shareLink, t.notices.copiedVless)}>
              <Copy className="mr-1 size-3.5" />
              VLESS
            </Button>
            <Button
              size="sm"
              variant="secondary"
              disabled={busy}
              onClick={() => copyText(client.mihomoSubscriptionUrl, t.notices.copiedSubscription)}
            >
              <Copy className="mr-1 size-3.5" />
              {t.client.clash}
            </Button>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="ghost" disabled={busy} onClick={() => onRotate()}>
              <RefreshCw className="mr-1 size-3.5" />
              {t.client.resetToken}
            </Button>
            <Button
              size="sm"
              variant="ghost"
              className="text-danger hover:bg-danger/10 hover:text-danger"
              disabled={busy}
              onClick={() => onDelete()}
            >
              <Trash2 className="mr-1 size-3.5" />
              {t.client.remove}
            </Button>
          </div>
        </div>
      </div>
    </article>
  );
}

export default App;
