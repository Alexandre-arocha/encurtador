export type LinkStatusFilter = "" | "active" | "inactive" | "expired";

export type LinkItem = {
  id: string;
  slug: string;
  short_url: string;
  target_url: string;
  title: string | null;
  created_at: string;
  updated_at: string;
  expires_at: string | null;
  is_active: boolean;
  campaign: string | null;
  tags: string[];
  utm_source: string | null;
  utm_medium: string | null;
  utm_campaign: string | null;
  utm_term: string | null;
  utm_content: string | null;
  notes: string | null;
  total_clicks: number;
  last_clicked_at: string | null;
};

export type LinkFilters = {
  q?: string;
  status?: LinkStatusFilter;
  tag?: string;
  campaign?: string;
  limit?: number;
  offset?: number;
};

export type ListLinksResponse = {
  data: LinkItem[];
  total: number;
  limit: number;
  offset: number;
};

export type CreateLinkInput = {
  target_url: string;
  slug?: string;
  title?: string | null;
  expires_at?: string | null;
  campaign?: string | null;
  tags?: string[];
  utm_source?: string | null;
  utm_medium?: string | null;
  utm_campaign?: string | null;
  utm_term?: string | null;
  utm_content?: string | null;
  notes?: string | null;
};

export type UpdateLinkInput = {
  target_url?: string;
  title?: string | null;
  expires_at?: string | null;
  is_active?: boolean;
  campaign?: string | null;
  tags?: string[];
  utm_source?: string | null;
  utm_medium?: string | null;
  utm_campaign?: string | null;
  utm_term?: string | null;
  utm_content?: string | null;
  notes?: string | null;
};

export type DailyPoint = {
  day: string;
  clicks: number;
};

export type ReferrerPoint = {
  referrer: string | null;
  clicks: number;
};

export type BreakdownPoint = {
  key: string;
  clicks: number;
};

export type StatsRange = "7d" | "30d" | "all";

export type LinkStats = {
  range: StatsRange;
  start_day: string;
  end_day: string;
  total_clicks: number;
  daily: DailyPoint[];
  top_referrers: ReferrerPoint[];
  devices: BreakdownPoint[];
  countries: BreakdownPoint[];
  browsers: BreakdownPoint[];
  operating_systems: BreakdownPoint[];
  cities: BreakdownPoint[];
};
