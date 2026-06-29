import type { CreateLinkInput, LinkFilters, LinkItem, LinkStats, ListLinksResponse, StatsRange, UpdateLinkInput } from "./types";

const fallbackBaseURL = "http://localhost:8080";

export function defaultBaseURL() {
  return process.env.NEXT_PUBLIC_API_BASE_URL ?? fallbackBaseURL;
}

export function defaultAPIKey() {
  return process.env.NEXT_PUBLIC_API_KEY ?? "dev-api-key-change-me";
}

export class APIClient {
  constructor(
    private readonly baseURL: string,
    private readonly apiKey: string,
  ) {}

  async listLinks(filters: LinkFilters = {}) {
    return this.request<ListLinksResponse>(`/api/links${queryString(filters)}`);
  }

  async createLink(input: CreateLinkInput) {
    return this.request<LinkItem>("/api/links", {
      method: "POST",
      body: JSON.stringify(input),
    });
  }

  async updateLink(id: string, input: UpdateLinkInput) {
    return this.request<LinkItem>(`/api/links/${id}`, {
      method: "PATCH",
      body: JSON.stringify(input),
    });
  }

  async deleteLink(id: string) {
    await this.request<void>(`/api/links/${id}`, { method: "DELETE" });
  }

  async exportLinksCSV(filters: LinkFilters = {}) {
    const response = await fetch(`${this.baseURL}/api/links/export.csv${queryString(filters)}`, {
      headers: {
        "X-API-Key": this.apiKey,
      },
    });
    if (!response.ok) {
      const text = await response.text();
      let message = "Erro ao exportar CSV";
      try {
        const data = text ? JSON.parse(text) : null;
        message = data?.error?.message ?? message;
      } catch {
        if (text) message = text;
      }
      throw new Error(message);
    }
    return response.blob();
  }

  async getStats(id: string, range: StatsRange) {
    return this.request<LinkStats>(`/api/links/${id}/stats?range=${range}`);
  }

  private async request<T>(path: string, init: RequestInit = {}): Promise<T> {
    const response = await fetch(`${this.baseURL}${path}`, {
      ...init,
      headers: {
        "Content-Type": "application/json",
        "X-API-Key": this.apiKey,
        ...init.headers,
      },
    });

    if (response.status === 204) {
      return undefined as T;
    }

    const text = await response.text();
    const data = text ? JSON.parse(text) : null;
    if (!response.ok) {
      const message = data?.error?.message ?? "Erro ao chamar a API";
      throw new Error(message);
    }
    return data as T;
  }
}

function queryString(filters: LinkFilters) {
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(filters)) {
    if (value === undefined || value === null || value === "") continue;
    params.set(key, String(value));
  }
  const query = params.toString();
  return query ? `?${query}` : "";
}
