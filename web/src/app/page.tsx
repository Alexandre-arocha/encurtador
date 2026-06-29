"use client";

import type { FormEvent, ReactNode, TextareaHTMLAttributes } from "react";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  Activity,
  BarChart3,
  Check,
  Copy,
  Download,
  ExternalLink,
  Filter,
  Link2,
  Loader2,
  Pencil,
  Plus,
  QrCode,
  RefreshCw,
  Save,
  Search,
  Settings2,
  Trash2,
} from "lucide-react";
import QRCode from "qrcode";
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

import { Badge, Button, Field, IconButton, Input, Select } from "@/components/ui";
import { APIClient, defaultAPIKey, defaultBaseURL } from "@/lib/api";
import type { LinkFilters, LinkItem, LinkStats, LinkStatusFilter, StatsRange } from "@/lib/types";
import { dateOnly, fromDatetimeLocal, toDatetimeLocal } from "@/lib/utils";

const palette = ["#0f766e", "#2563eb", "#d97706", "#7c3aed", "#be123c", "#475569", "#0f172a"];
const defaultFilters: Required<Pick<LinkFilters, "q" | "status" | "tag" | "campaign">> = {
  q: "",
  status: "",
  tag: "",
  campaign: "",
};

type GrowthForm = {
  targetURL: string;
  slug: string;
  title: string;
  campaign: string;
  tags: string;
  utmSource: string;
  utmMedium: string;
  utmCampaign: string;
  utmTerm: string;
  utmContent: string;
  notes: string;
  expiresAt: string;
};

type EditForm = Omit<GrowthForm, "slug"> & {
  isActive: boolean;
};

const emptyCreateForm: GrowthForm = {
  targetURL: "",
  slug: "",
  title: "",
  campaign: "",
  tags: "",
  utmSource: "",
  utmMedium: "",
  utmCampaign: "",
  utmTerm: "",
  utmContent: "",
  notes: "",
  expiresAt: "",
};

export default function Home() {
  const [baseURL, setBaseURL] = useState(defaultBaseURL());
  const [apiKey, setAPIKey] = useState(defaultAPIKey());
  const [links, setLinks] = useState<LinkItem[]>([]);
  const [totalLinks, setTotalLinks] = useState(0);
  const [selectedID, setSelectedID] = useState<string>("");
  const [stats, setStats] = useState<LinkStats | null>(null);
  const [range, setRange] = useState<StatsRange>("7d");
  const [filters, setFilters] = useState(defaultFilters);
  const [createForm, setCreateForm] = useState<GrowthForm>(emptyCreateForm);
  const [editForm, setEditForm] = useState<EditForm | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const [statsBusy, setStatsBusy] = useState(false);
  const [message, setMessage] = useState<string>("");
  const [copied, setCopied] = useState(false);
  const [qrDataURL, setQrDataURL] = useState("");

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
  const browserData = stats?.browsers ?? [];
  const osData = stats?.operating_systems ?? [];
  const cityData = stats?.cities ?? [];
  const referrerData = stats?.top_referrers ?? [];
  const hasDailyClicks = dailyData.some((point) => point.clicks > 0);

  const loadLinks = useCallback(async () => {
    setBusy(true);
    setMessage("");
    try {
      const response = await api.listLinks({ ...filters, limit: 100, offset: 0 });
      setLinks(response.data);
      setTotalLinks(response.total);
      setSelectedID((current) => (response.data.some((item) => item.id === current) ? current : response.data[0]?.id || ""));
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao carregar links");
    } finally {
      setBusy(false);
    }
  }, [api, filters]);

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
      campaign: selected.campaign ?? "",
      tags: selected.tags.join(", "),
      utmSource: selected.utm_source ?? "",
      utmMedium: selected.utm_medium ?? "",
      utmCampaign: selected.utm_campaign ?? "",
      utmTerm: selected.utm_term ?? "",
      utmContent: selected.utm_content ?? "",
      notes: selected.notes ?? "",
      expiresAt: toDatetimeLocal(selected.expires_at),
      isActive: selected.is_active,
    });
  }, [selected]);

  useEffect(() => {
    let active = true;
    setQrDataURL("");
    if (!selected) return;
    void QRCode.toDataURL(selected.short_url, {
      margin: 1,
      width: 224,
      color: { dark: "#0f172a", light: "#ffffff" },
    }).then((value) => {
      if (active) setQrDataURL(value);
    });
    return () => {
      active = false;
    };
  }, [selected]);

  async function handleCreate(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setBusy(true);
    setMessage("");
    try {
      const created = await api.createLink({
        target_url: createForm.targetURL,
        slug: createForm.slug || undefined,
        title: nullable(createForm.title),
        expires_at: fromDatetimeLocal(createForm.expiresAt),
        campaign: nullable(createForm.campaign),
        tags: parseTags(createForm.tags),
        utm_source: nullable(createForm.utmSource),
        utm_medium: nullable(createForm.utmMedium),
        utm_campaign: nullable(createForm.utmCampaign),
        utm_term: nullable(createForm.utmTerm),
        utm_content: nullable(createForm.utmContent),
        notes: nullable(createForm.notes),
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
        title: editForm.title,
        expires_at: fromDatetimeLocal(editForm.expiresAt),
        is_active: editForm.isActive,
        campaign: editForm.campaign,
        tags: parseTags(editForm.tags),
        utm_source: editForm.utmSource,
        utm_medium: editForm.utmMedium,
        utm_campaign: editForm.utmCampaign,
        utm_term: editForm.utmTerm,
        utm_content: editForm.utmContent,
        notes: editForm.notes,
      });
      setLinks((items) => items.map((item) => (item.id === updated.id ? { ...item, ...updated } : item)));
      setMessage("Link atualizado");
      void loadLinks();
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
      setTotalLinks((current) => Math.max(0, current - 1));
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

  function downloadQR() {
    if (!selected || !qrDataURL) return;
    const anchor = document.createElement("a");
    anchor.href = qrDataURL;
    anchor.download = `qr-${selected.slug}.png`;
    anchor.click();
  }

  async function exportCSV() {
    setBusy(true);
    setMessage("");
    try {
      const blob = await api.exportLinksCSV(filters);
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = "encurtador-links.csv";
      anchor.click();
      URL.revokeObjectURL(url);
      setMessage("CSV exportado");
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "Erro ao exportar CSV");
    } finally {
      setBusy(false);
    }
  }

  function applyUTMToCreate() {
    setCreateForm((form) => ({ ...form, targetURL: withUTM(form.targetURL, form) }));
  }

  function applyUTMToEdit() {
    setEditForm((form) => (form ? { ...form, targetURL: withUTM(form.targetURL, form) } : form));
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
              <h1 className="truncate text-lg font-semibold text-slate-950">Encurtador Growth Ops</h1>
              <p className="truncate text-sm text-muted-foreground">{baseURL}</p>
            </div>
          </div>
          <div className="flex items-center gap-2">
            {message && <span className="max-w-[42vw] truncate text-sm text-slate-600">{message}</span>}
            <IconButton label="Recarregar" onClick={() => void loadLinks()} disabled={busy}>
              <RefreshCw className={`h-4 w-4 ${busy ? "animate-spin" : ""}`} aria-hidden />
            </IconButton>
            <IconButton label="Exportar CSV" onClick={() => void exportCSV()} disabled={busy}>
              <Download className="h-4 w-4" aria-hidden />
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

      <section className="border-b border-border bg-white">
        <form
          className="mx-auto grid max-w-7xl gap-3 px-4 py-3 sm:px-6 lg:grid-cols-[1.3fr_170px_170px_170px_auto]"
          onSubmit={(event) => {
            event.preventDefault();
            void loadLinks();
          }}
        >
          <Field label="Busca">
            <div className="relative">
              <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" aria-hidden />
              <Input
                className="pl-9"
                value={filters.q}
                onChange={(event) => setFilters((current) => ({ ...current, q: event.target.value }))}
                placeholder="slug, destino, título"
              />
            </div>
          </Field>
          <Field label="Status">
            <Select
              value={filters.status}
              onChange={(event) => setFilters((current) => ({ ...current, status: event.target.value as LinkStatusFilter }))}
            >
              <option value="">todos</option>
              <option value="active">ativos</option>
              <option value="inactive">inativos</option>
              <option value="expired">expirados</option>
            </Select>
          </Field>
          <Field label="Tag">
            <Input value={filters.tag} onChange={(event) => setFilters((current) => ({ ...current, tag: event.target.value }))} />
          </Field>
          <Field label="Campanha">
            <Input
              value={filters.campaign}
              onChange={(event) => setFilters((current) => ({ ...current, campaign: event.target.value }))}
            />
          </Field>
          <div className="flex items-end gap-2">
            <Button type="submit" disabled={busy} className="w-full lg:w-auto">
              <Filter className="h-4 w-4" aria-hidden />
              Filtrar
            </Button>
          </div>
        </form>
      </section>

      <div className="mx-auto grid max-w-7xl gap-0 px-4 py-5 sm:px-6 xl:grid-cols-[430px_1fr]">
        <aside className="border border-border bg-white shadow-panel xl:border-r-0">
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
            <div className="grid gap-3 sm:grid-cols-2">
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
            <div className="grid gap-3 sm:grid-cols-2">
              <Field label="Título">
                <Input value={createForm.title} onChange={(event) => setCreateForm((form) => ({ ...form, title: event.target.value }))} />
              </Field>
              <Field label="Campanha">
                <Input
                  value={createForm.campaign}
                  onChange={(event) => setCreateForm((form) => ({ ...form, campaign: event.target.value }))}
                />
              </Field>
            </div>
            <Field label="Tags">
              <Input
                placeholder="ads, br, lançamento"
                value={createForm.tags}
                onChange={(event) => setCreateForm((form) => ({ ...form, tags: event.target.value.toLowerCase() }))}
              />
            </Field>
            <UTMFields form={createForm} setForm={(updater) => setCreateForm(updater)} onApply={applyUTMToCreate} />
            <Button type="submit" disabled={busy}>
              {busy ? <Loader2 className="h-4 w-4 animate-spin" aria-hidden /> : <Plus className="h-4 w-4" aria-hidden />}
              Criar
            </Button>
          </form>

          <div className="flex items-center justify-between border-b border-border px-4 py-3 text-sm">
            <span className="font-medium text-slate-700">{totalLinks} links</span>
            <span className="text-muted-foreground">{links.length} na página</span>
          </div>
          <div className="max-h-[calc(100vh-430px)] overflow-y-auto p-2">
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
                <div className="flex flex-wrap items-center gap-2 text-xs text-slate-500">
                  <span>{item.total_clicks} cliques</span>
                  {item.campaign && <span>{item.campaign}</span>}
                  {item.last_clicked_at && <span>{dateOnly(item.last_clicked_at)}</span>}
                </div>
                {item.tags.length > 0 && (
                  <div className="flex flex-wrap gap-1">
                    {item.tags.slice(0, 4).map((tag) => (
                      <span key={tag} className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-700">
                        {tag}
                      </span>
                    ))}
                  </div>
                )}
              </button>
            ))}
          </div>
        </aside>

        <section className="min-w-0 border border-border bg-white shadow-panel">
          {!selected ? (
            <div className="flex min-h-[560px] items-center justify-center text-sm text-muted-foreground">Selecione um link</div>
          ) : (
            <div className="grid gap-0">
              <div className="border-b border-border p-4">
                <div className="flex flex-wrap items-start justify-between gap-3">
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <h2 className="break-all text-xl font-semibold text-slate-950">/{selected.slug}</h2>
                      <Badge tone={selected.is_active ? "good" : "warn"}>{selected.is_active ? "ativo" : "inativo"}</Badge>
                      {selected.campaign && <Badge>{selected.campaign}</Badge>}
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

              <div className="grid gap-0 2xl:grid-cols-[1fr_360px]">
                <div className="min-w-0 border-b border-border p-4 2xl:border-b-0 2xl:border-r">
                  <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
                    <div className="flex items-center gap-2">
                      <Activity className="h-4 w-4 text-teal-700" aria-hidden />
                      <h3 className="text-sm font-semibold uppercase text-slate-600">Analytics</h3>
                    </div>
                    <RangeSelect value={range} onChange={setRange} />
                  </div>

                  <div className="mb-4 grid gap-3 sm:grid-cols-4">
                    <Metric label="Cliques" value={stats?.total_clicks ?? selected.total_clicks} />
                    <Metric label="Início" value={stats?.start_day ?? "—"} />
                    <Metric label="Fim" value={stats?.end_day ?? "—"} />
                    <Metric label="Último clique" value={dateOnly(selected.last_clicked_at)} />
                  </div>

                  <div className="h-72 rounded-md border border-border p-3">
                    {statsBusy ? (
                      <LoadingBlock />
                    ) : !hasDailyClicks ? (
                      <EmptyBlock>Sem cliques no período</EmptyBlock>
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
                      <PieBreakdown data={deviceData} empty="Sem dispositivos" />
                    </ChartBox>
                    <ChartBox title="Países">
                      <BarBreakdown data={countryData} empty="Sem países" color="#2563eb" />
                    </ChartBox>
                    <ChartBox title="Browsers">
                      <BarBreakdown data={browserData} empty="Sem browsers" color="#d97706" />
                    </ChartBox>
                    <ChartBox title="Sistemas">
                      <PieBreakdown data={osData} empty="Sem sistemas" />
                    </ChartBox>
                    <ChartBox title="Cidades">
                      <BarBreakdown data={cityData} empty="Sem cidades" color="#7c3aed" />
                    </ChartBox>
                    <ReferrerBox data={referrerData} />
                  </div>
                </div>

                <div className="grid content-start gap-4 p-4">
                  <section className="grid gap-3 rounded-md border border-border p-3">
                    <div className="flex items-center gap-2">
                      <QrCode className="h-4 w-4 text-slate-700" aria-hidden />
                      <h3 className="text-sm font-semibold uppercase text-slate-600">QR</h3>
                    </div>
                    <div className="flex justify-center rounded-md border border-border bg-white p-3">
                      {qrDataURL ? <img src={qrDataURL} alt={`QR code de ${selected.slug}`} className="h-44 w-44" /> : <LoadingBlock />}
                    </div>
                    <Button type="button" variant="secondary" onClick={downloadQR} disabled={!qrDataURL}>
                      <Download className="h-4 w-4" aria-hidden />
                      PNG
                    </Button>
                  </section>

                  <form className="grid content-start gap-3 rounded-md border border-border p-3" onSubmit={handleSaveEdit}>
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
                        <div className="grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
                          <Field label="Título">
                            <Input
                              value={editForm.title}
                              onChange={(event) => setEditForm((form) => form && { ...form, title: event.target.value })}
                            />
                          </Field>
                          <Field label="Campanha">
                            <Input
                              value={editForm.campaign}
                              onChange={(event) => setEditForm((form) => form && { ...form, campaign: event.target.value })}
                            />
                          </Field>
                        </div>
                        <Field label="Tags">
                          <Input
                            value={editForm.tags}
                            onChange={(event) => setEditForm((form) => form && { ...form, tags: event.target.value.toLowerCase() })}
                          />
                        </Field>
                        <UTMFields
                          form={editForm}
                          setForm={(updater) => setEditForm((current) => (current ? updater(current) : current))}
                          onApply={applyUTMToEdit}
                        />
                        <div className="grid gap-3 sm:grid-cols-2 2xl:grid-cols-1">
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
                        </div>
                        <Field label="Notas">
                          <TextArea value={editForm.notes} onChange={(event) => setEditForm((form) => form && { ...form, notes: event.target.value })} />
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
            </div>
          )}
        </section>
      </div>
    </main>
  );
}

function UTMFields<T extends GrowthForm | EditForm>({
  form,
  setForm,
  onApply,
}: {
  form: T;
  setForm: (fn: (form: T) => T) => void;
  onApply: () => void;
}) {
  return (
    <section className="grid gap-3 rounded-md border border-border bg-slate-50 p-3">
      <div className="flex items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-sm font-semibold uppercase text-slate-600">
          <Link2 className="h-4 w-4 text-teal-700" aria-hidden />
          UTM
        </div>
        <Button type="button" variant="secondary" onClick={onApply}>
          Aplicar
        </Button>
      </div>
      <div className="grid gap-3 sm:grid-cols-2">
        <Field label="Source">
          <Input value={form.utmSource} onChange={(event) => setForm((current) => ({ ...current, utmSource: event.target.value }))} />
        </Field>
        <Field label="Medium">
          <Input value={form.utmMedium} onChange={(event) => setForm((current) => ({ ...current, utmMedium: event.target.value }))} />
        </Field>
        <Field label="Campaign">
          <Input value={form.utmCampaign} onChange={(event) => setForm((current) => ({ ...current, utmCampaign: event.target.value }))} />
        </Field>
        <Field label="Term">
          <Input value={form.utmTerm} onChange={(event) => setForm((current) => ({ ...current, utmTerm: event.target.value }))} />
        </Field>
      </div>
      <Field label="Content">
        <Input value={form.utmContent} onChange={(event) => setForm((current) => ({ ...current, utmContent: event.target.value }))} />
      </Field>
    </section>
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

function PieBreakdown({ data, empty }: { data: { key: string; clicks: number }[]; empty: string }) {
  if (data.length === 0) return <EmptyBlock>{empty}</EmptyBlock>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <PieChart>
        <Pie data={data} dataKey="clicks" nameKey="key" innerRadius={42} outerRadius={76} paddingAngle={2}>
          {data.map((entry, index) => (
            <Cell key={entry.key} fill={palette[index % palette.length]} />
          ))}
        </Pie>
        <Tooltip />
      </PieChart>
    </ResponsiveContainer>
  );
}

function BarBreakdown({ data, empty, color }: { data: { key: string; clicks: number }[]; empty: string; color: string }) {
  if (data.length === 0) return <EmptyBlock>{empty}</EmptyBlock>;
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart data={data} margin={{ top: 8, right: 12, bottom: 0, left: 0 }}>
        <CartesianGrid stroke="#e2e8f0" strokeDasharray="3 3" />
        <XAxis dataKey="key" tick={{ fontSize: 12 }} />
        <YAxis allowDecimals={false} tick={{ fontSize: 12 }} width={32} />
        <Tooltip />
        <Bar dataKey="clicks" fill={color} radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  );
}

function ReferrerBox({ data }: { data: { referrer: string | null; clicks: number }[] }) {
  return (
    <div className="h-64 rounded-md border border-border">
      <div className="border-b border-border px-3 py-2 text-sm font-semibold text-slate-700">Referrers</div>
      {data.length === 0 ? (
        <div className="flex h-[calc(100%-38px)] items-center justify-center text-sm text-muted-foreground">Sem referrer</div>
      ) : (
        <div className="h-[calc(100%-38px)] divide-y divide-border overflow-y-auto">
          {data.map((item) => (
            <div key={item.referrer ?? "direct"} className="grid grid-cols-[1fr_auto] gap-3 px-3 py-2 text-sm">
              <span className="min-w-0 break-all text-slate-700">{item.referrer ?? "direto"}</span>
              <span className="font-medium text-slate-950">{item.clicks}</span>
            </div>
          ))}
        </div>
      )}
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

function TextArea(props: TextareaHTMLAttributes<HTMLTextAreaElement>) {
  return (
    <textarea
      className="min-h-24 rounded-md border border-border bg-white px-3 py-2 text-sm text-slate-900 outline-none transition placeholder:text-slate-400 focus:border-teal-600 focus:ring-2 focus:ring-teal-600/15"
      {...props}
    />
  );
}

function parseTags(raw: string) {
  return raw
    .split(",")
    .map((tag) => tag.trim().toLowerCase())
    .filter(Boolean);
}

function nullable(value: string) {
  const trimmed = value.trim();
  return trimmed === "" ? null : trimmed;
}

function withUTM(rawURL: string, form: Pick<GrowthForm, "utmSource" | "utmMedium" | "utmCampaign" | "utmTerm" | "utmContent">) {
  try {
    const url = new URL(rawURL);
    setParam(url, "utm_source", form.utmSource);
    setParam(url, "utm_medium", form.utmMedium);
    setParam(url, "utm_campaign", form.utmCampaign);
    setParam(url, "utm_term", form.utmTerm);
    setParam(url, "utm_content", form.utmContent);
    return url.toString();
  } catch {
    return rawURL;
  }
}

function setParam(url: URL, key: string, value: string) {
  const trimmed = value.trim();
  if (trimmed) {
    url.searchParams.set(key, trimmed);
  } else {
    url.searchParams.delete(key);
  }
}
