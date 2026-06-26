export type LinkItem = {
  id: string;
  slug: string;
  short_url: string;
  target_url: string;
  title: string | null;
  created_at: string;
  expires_at: string | null;
  is_active: boolean;
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
};

export type UpdateLinkInput = {
  target_url?: string;
  title?: string | null;
  expires_at?: string | null;
  is_active?: boolean;
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
};
