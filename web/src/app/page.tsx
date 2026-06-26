"use client";

import type { FormEvent, ReactNode } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Activity,
  BarChart3,
  Check,
  Copy,
  ExternalLink,
  Loader2,
  Pencil,
  Plus,
  RefreshCw,
  Save,
  Settings2,
  Trash2,
} from "lucide-react";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Pie,
  PieChart,
  ResponsiveContainer,
  Tooltip,
  XAxis,
  YAxis,
} from "recharts";

import { APIClient, defaultAPIKey, defaultBaseURL } from "@/lib/api";
import type { LinkItem, LinkStats, StatsRange } from "@/lib/types";
import { dateOnly, fromDatetimeLocal, toDatetimeLocal } from "@/lib/utils";
import { Badge, Button, Field, IconButton, Input, Select } from "@/components/ui";

const palette = ["#0f766e", "#2563eb", "#d97706", "#7c3aed", "#be123c", "#475569"];

type CreateForm = {
  targetURL: string;
  slug: string;
  title: string;
  expiresAt: string;
};

type EditForm = {
  targetURL: string;
  title: string;
  expiresAt: string;
  isActive: boolean;
};

const emptyCreateForm: CreateForm = {
  targetURL: "",
  slug: "",
  title: "",
  expiresAt: "",
};

export default function Home() {
  const [baseURL, setBaseURL] = useState(defaultBaseURL());
  const [apiKey, setAPIKey] = useState(defaultAPIKey());
  const [links, setLinks] = useState<LinkItem[]>([]);
  const [selectedID, setSelectedID] = useState<string>("");
  const [stats, setStats] = useState<LinkStats | null>(null);
  const [range, setRange] = useState<StatsRange>("7d");
  const [createForm, setCreateForm] = useState<CreateForm>(emptyCreateForm);
  const [editForm, setEditForm] = useState<EditForm | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [statsBusy, setStatsBusy] = useState(false);
  const [message, setMessage] = useState<string>("");
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    const savedBaseURL = localStorage.getItem("encurtador.baseURL");
    const savedAPIKey = localStorage.getItem("encurtador.apiKey");
    if (savedBaseURL) setBaseURL(savedBaseURL);
    if (savedAPIKey) setAPIKey(savedAPIKey);
  }, []);

  useEffect(() => {
    localStorage.setItem("encurtador.baseURL", baseURL);
    localStorage.setItem("encurtador.apiKey", apiKey);
  }, [apiKey, baseURL]);

  const api = useMemo(() => new APIClient(baseURL.replace(/\/$/, ""), apiKey), [apiKey, baseURL]);
  const selected = links.find((item) => item.id === selectedID) ?? links[0] ?? null;
  const dailyData = stats?.daily ?? [];
  const deviceData = stats?.devices ?? [];
  const countryData = stats?.countries ?? [];
  const referrerData = stats?.top_referrers ?? [];
  const hasDailyClicks = dailyData.some((point) => point.clicks > 0);

  const loadLinks = useCallback(async () => {
    setBusy(true);
    setMessage("");
    try {
      const response = await api.listLinks();
      setLinks(response.data);
      setSelectedID((current) => current || response.data[0]?.id || "");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao carregar links");
    } finally {
      setBusy(false);
    }
  }, [api]);

  const loadStats = useCallback(
    async (linkID: string, selectedRange: StatsRange) => {
      setStatsBusy(true);
      try {
        setStats(await api.getStats(linkID, selectedRange));
      } catch (error) {
        setMessage(error instanceof Error ? error.message : "Erro ao carregar estatísticas");
      } finally {
        setStatsBusy(false);
      }
    },
    [api],
  );

  useEffect(() => {
    void loadLinks();
  }, [loadLinks]);

  useEffect(() => {
    if (!selected) {
      setStats(null);
      return;
    }
    void loadStats(selected.id, range);
  }, [loadStats, range, selected]);

  useEffect(() => {
    if (!selected) {
      setEditForm(null);
      return;
    }
    setEditForm({
      targetURL: selected.target_url,
      title: selected.title ?? "",
      expiresAt: toDatetimeLocal(selected.expires_at),
      isActive: selected.is_active,
    });
  }, [selected]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setMessage("");
    try {
      const created = await api.createLink({
        target_url: createForm.targetURL,
        slug: createForm.slug || undefined,
        title: createForm.title || null,
        expires_at: fromDatetimeLocal(createForm.expiresAt),
      });
      setCreateForm(emptyCreateForm);
      await loadLinks();
      setSelectedID(created.id);
      setMessage("Link criado");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao criar link");
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveEdit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selected || !editForm) return;

    setBusy(true);
    setMessage("");
    try {
      const updated = await api.updateLink(selected.id, {
        target_url: editForm.targetURL,
        title: editForm.title || null,
        expires_at: fromDatetimeLocal(editForm.expiresAt),
        is_active: editForm.isActive,
      });
      setLinks((items) => items.map((item) => (item.id === updated.id ? updated : item)));
      setMessage("Link atualizado");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao salvar link");
    } finally {
      setBusy(false);
    }
  }

  async function handleDelete() {
    if (!selected) return;
    setBusy(true);
    setMessage("");
    try {
      await api.deleteLink(selected.id);
      const remaining = links.filter((item) => item.id !== selected.id);
      setLinks(remaining);
      setSelectedID(remaining[0]?.id ?? "");
      setStats(null);
      setMessage("Link removido");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao remover link");
    } finally {
      setBusy(false);
    }
  }

  async function copyShortURL() {
    if (!selected) return;
    await navigator.clipboard.writeText(selected.short_url);
    setCopied(true);
    window.setTimeout(() => setCopied(false), 1400);
  }

  return (
    <main className="min-h-screen">
      <header className="border-b border-border bg-white">
        <div className="mx-auto flex max-w-7xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
          <div className="flex min-w-0 items-center gap-3">
            <div className="flex h-9 w-9 items-center justify-center rounded-md bg-teal-700 text-white">
              <BarChart3 className="h-5 w-5" aria-hidden />
            </div>
            <div className="min-w-0">
              <h1 className="truncate text-lg font-semibold text-slate-950">Encurtador</h1>
              <p className="truncate text-sm text-muted-foreground">{baseURL}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {message && <span className="max-w-[42vw] truncate text-sm text-slate-600">{message}</span>}
            <IconButton label="Recarregar" onClick={() => void loadLinks()} disabled={busy}>
              <RefreshCw className={`h-4 w-4 ${busy ? "animate-spin" : ""}`} aria-hidden />
            </IconButton>
            <IconButton label="Configurações" onClick={() => setSettingsOpen((value) => !value)}>
              <Settings2 className="h-4 w-4" aria-hidden />
            </IconButton>
          </div>
        </div>
      </header>

      {settingsOpen && (
        <section className="border-b border-border bg-slate-50">
          <div className="mx-auto grid max-w-7xl gap-3 px-4 py-4 sm:grid-cols-[1fr_1fr_auto] sm:px-6">
            <Field label="API">
              <Input value={baseURL} onChange={(event) => setBaseURL(event.target.value)} />
            </Field>
            <Field label="Chave">
              <Input value={apiKey} onChange={(event) => setAPIKey(event.target.value)} type="password" />
            </Field>
            <div className="flex items-end">
              <Button variant="secondary" onClick={() => void loadLinks()} className="w-full sm:w-auto">
                <RefreshCw className="h-4 w-4" aria-hidden />
                Aplicar
              </Button>
            </div>
          </div>
        </section>
      )}

      <div className="mx-auto grid max-w-7xl gap-0 px-4 py-5 sm:px-6 lg:grid-cols-[390px_1fr]">
        <aside className="border border-border bg-white shadow-panel lg:border-r-0">
          <form className="grid gap-3 border-b border-border p-4" onSubmit={handleCreate}>
            <div className="flex items-center justify-between gap-2">
              <h2 className="text-sm font-semibold uppercase text-slate-600">Novo link</h2>
              <Badge tone="good">ativo</Badge>
            </div>
            <Field label="Destino">
              <Input
                required
                placeholder="https://..."
                value={createForm.targetURL}
                onChange={(event) => setCreateForm((form) => ({ ...form, targetURL: event.target.value }))}
              />
            </Field>
            <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-1">
              <Field label="Slug">
                <Input
                  placeholder="opcional"
                  value={createForm.slug}
                  onChange={(event) => setCreateForm((form) => ({ ...form, slug: event.target.value }))}
                />
              </Field>
              <Field label="Expiração">
                <Input
                  type="datetime-local"
                  value={createForm.expiresAt}
                  onChange={(event) => setCreateForm((form) => ({ ...form, expiresAt: event.target.value }))}
                />
              </Field>
            </div>
            <Field label="Título">
              <Input
                placeholder="opcional"
                value={createForm.title}
                onChange={(event) => setCreateForm((form) => ({ ...form, title: event.target.value }))}
              />
            </Field>
            <Button type="submit" disabled={busy}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Plus className="h-4 w-4" aria-hidden />}
              Criar
            </Button>
          </form>

          <div className="max-h-[calc(100vh-310px)] overflow-y-auto p-2">
            {links.length === 0 && (
              <div className="px-3 py-8 text-center text-sm text-muted-foreground">Nenhum link encontrado</div>
            )}
            {links.map((item) => (
              <button
                key={item.id}
                onClick={() => setSelectedID(item.id)}
                className={`mb-2 grid w-full gap-2 rounded-md border p-3 text-left transition ${
                  selected?.id === item.id
                    ? "border-teal-600 bg-teal-50"
                    : "border-border bg-white hover:border-slate-400 hover:bg-slate-50"
                }`}
              >
                <div className="flex min-w-0 items-center justify-between gap-2">
                  <span className="truncate font-medium text-slate-950">/{item.slug}</span>
                  <Badge tone={item.is_active ? "good" : "warn"}>{item.is_active ? "ativo" : "inativo"}</Badge>
                </div>
                <span className="break-all text-xs text-muted-foreground">{item.target_url}</span>
                <span className="text-xs text-slate-500">{dateOnly(item.created_at)}</span>
              </button>
            ))}
          </div>
        </aside>

        <section className="min-w-0 border border-border bg-white shadow-panel">
          {!selected ? (
            <div className="flex min-h-[520px] items-center justify-center text-sm text-muted-foreground">
              Selecione um link
            </div>
          ) : (
            <div className="grid gap-0">
              <div className="border-b border-border p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="break-all text-xl font-semibold text-slate-950">/{selected.slug}</h2>
                      <Badge tone={selected.is_active ? "good" : "warn"}>{selected.is_active ? "ativo" : "inativo"}</Badge>
                    </div>
                    <p className="mt-1 break-all text-sm text-muted-foreground">{selected.target_url}</p>
                  </div>
                  <div className="flex items-center gap-2">
                    <IconButton label={copied ? "Copiado" : "Copiar"} onClick={() => void copyShortURL()}>
                      {copied ? <Check className="h-4 w-4" aria-hidden /> : <Copy className="h-4 w-4" aria-hidden />}
                    </IconButton>
                    <IconButton label="Abrir" onClick={() => window.open(selected.short_url, "_blank", "noopener,noreferrer")}>
                      <ExternalLink className="h-4 w-4" aria-hidden />
                    </IconButton>
                    <IconButton label="Excluir" className="text-rose-700 hover:bg-rose-50" onClick={() => void handleDelete()}>
                      <Trash2 className="h-4 w-4" aria-hidden />
                    </IconButton>
                  </div>
                </div>
              </div>

              <div className="grid gap-0 xl:grid-cols-[1fr_330px]">
                <div className="min-w-0 border-b border-border p-4 xl:border-b-0 xl:border-r">
                  <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                      <Activity className="h-4 w-4 text-teal-700" aria-hidden />
                      <h3 className="text-sm font-semibold uppercase text-slate-600">Analytics</h3>
                    </div>
                    <RangeSelect value={range} onChange={setRange} />
                  </div>

                  <div className="mb-4 grid gap-3 sm:grid-cols-3">
                    <Metric label="Cliques" value={stats?.total_clicks ?? 0} />
                    <Metric label="Início" value={stats?.start_day ?? "—"} />
                    <Metric label="Fim" value={stats?.end_day ?? "—"} />
                  </div>

                  <div className="h-72 rounded-md border border-border p-3">
                    {statsBusy ? (
                      <LoadingBlock />
                    ) : !hasDailyClicks ? (
                      <EmptyBlock>Sem cliques no periodo</EmptyBlock>
                    ) : (
                      <ResponsiveContainer width="100%" height="100%">
                        <AreaChart data={dailyData} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                          <defs>
                            <linearGradient id="clicksFill" x1="0" x2="0" y1="0" y2="1">
                              <stop offset="5%" stopColor="#0f766e" stopOpacity={0.35} />
                              <stop offset="95%" stopColor="#0f766e" stopOpacity={0.02} />
                            </linearGradient>
                          </defs>
                          <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
                          <XAxis dataKey="day" tick={{ fontSize: 12 }} />
                          <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={32} />
                          <Tooltip />
                          <Area type="monotone" dataKey="clicks" stroke="#0f766e" strokeWidth={2} fill="url(#clicksFill)" />
                        </AreaChart>
                      </ResponsiveContainer>
                    )}
                  </div>

                  <div className="mt-4 grid gap-4 lg:grid-cols-2">
                    <ChartBox title="Dispositivos">
                      {deviceData.length === 0 ? (
                        <EmptyBlock>Sem dispositivos</EmptyBlock>
                      ) : (
                        <ResponsiveContainer width="100%" height="100%">
                          <PieChart>
                            <Pie data={deviceData} dataKey="clicks" nameKey="key" innerRadius={42} outerRadius={76} paddingAngle={2}>
                              {deviceData.map((entry, index) => (
                                <Cell key={entry.key} fill={palette[index % palette.length]} />
                              ))}
                            </Pie>
                            <Tooltip />
                          </PieChart>
                        </ResponsiveContainer>
                      )}
                    </ChartBox>
                    <ChartBox title="Países">
                      {countryData.length === 0 ? (
                        <EmptyBlock>Sem paises</EmptyBlock>
                      ) : (
                        <ResponsiveContainer width="100%" height="100%">
                          <BarChart data={countryData} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
                            <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
                            <XAxis dataKey="key" tick={{ fontSize: 12 }} />
                            <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={32} />
                            <Tooltip />
                            <Bar dataKey="clicks" fill="#2563eb" radius={[4, 4, 0, 0]} />
                          </BarChart>
                        </ResponsiveContainer>
                      )}
                    </ChartBox>
                  </div>

                  <div className="mt-4 rounded-md border border-border">
                    <div className="border-b border-border px-3 py-2 text-sm font-semibold text-slate-700">Referrers</div>
                    {referrerData.length === 0 ? (
                      <div className="px-3 py-5 text-sm text-muted-foreground">Sem referrer</div>
                    ) : (
                      <div className="divide-y divide-border">
                        {referrerData.map((item) => (
                          <div key={item.referrer ?? "direct"} className="grid grid-cols-[1fr_auto] gap-3 px-3 py-2 text-sm">
                            <span className="min-w-0 break-all text-slate-700">{item.referrer ?? "direto"}</span>
                            <span className="font-medium text-slate-950">{item.clicks}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>

                <form className="grid content-start gap-3 p-4" onSubmit={handleSaveEdit}>
                  <div className="flex items-center gap-2">
                    <Pencil className="h-4 w-4 text-blue-700" aria-hidden />
                    <h3 className="text-sm font-semibold uppercase text-slate-600">Editar</h3>
                  </div>
                  {editForm && (
                    <>
                      <Field label="Destino">
                        <Input
                          required
                          value={editForm.targetURL}
                          onChange={(event) => setEditForm((form) => form && { ...form, targetURL: event.target.value })}
                        />
                      </Field>
                      <Field label="Título">
                        <Input
                          value={editForm.title}
                          onChange={(event) => setEditForm((form) => form && { ...form, title: event.target.value })}
                        />
                      </Field>
                      <Field label="Expiração">
                        <Input
                          type="datetime-local"
                          value={editForm.expiresAt}
                          onChange={(event) => setEditForm((form) => form && { ...form, expiresAt: event.target.value })}
                        />
                      </Field>
                      <Field label="Status">
                        <Select
                          value={editForm.isActive ? "active" : "inactive"}
                          onChange={(event) =>
                            setEditForm((form) => form && { ...form, isActive: event.target.value === "active" })
                          }
                        >
                          <option value="active">ativo</option>
                          <option value="inactive">inativo</option>
                        </Select>
                      </Field>
                      <Button type="submit" disabled={busy}>
                        <Save className="h-4 w-4" aria-hidden />
                        Salvar
                      </Button>
                    </>
                  )}
                </form>
              </div>
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function RangeSelect({ value, onChange }: { value: StatsRange; onChange: (value: StatsRange) => void }) {
  return (
    <div className="inline-grid grid-cols-3 rounded-md border border-border bg-white p-1">
      {(["7d", "30d", "all"] as StatsRange[]).map((item) => (
        <button
          key={item}
          type="button"
          onClick={() => onChange(item)}
          className={`h-8 rounded px-3 text-sm font-medium transition ${
            value === item ? "bg-slate-900 text-white" : "text-slate-600 hover:bg-slate-100"
          }`}
        >
          {item}
        </button>
      ))}
    </div>
  );
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-md border border-border bg-slate-50 px-3 py-2">
      <div className="text-xs font-medium uppercase text-muted-foreground">{label}</div>
      <div className="mt-1 truncate text-xl font-semibold text-slate-950">{value}</div>
    </div>
  );
}

function ChartBox({ title, children }: { title: string; children: ReactNode }) {
  return (
    <div className="h-64 rounded-md border border-border p-3">
      <div className="mb-2 text-sm font-semibold text-slate-700">{title}</div>
      <div className="h-[calc(100%-28px)]">{children}</div>
    </div>
  );
}

function LoadingBlock() {
  return (
    <div className="flex h-full items-center justify-center text-muted-foreground">
      <Loader2 className="h-5 w-5 animate-spin" aria-hidden />
    </div>
  );
}

function EmptyBlock({ children }: { children: ReactNode }) {
  return <div className="flex h-full items-center justify-center text-sm text-muted-foreground">{children}</div>;
}
