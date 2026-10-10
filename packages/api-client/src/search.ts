import type { Transport, Query } from "./client";

export type SearchResultType =
  | "request"
  | "user"
  | "team"
  | "project"
  | "policy"
  | "model"
  | "provider"
  | "conversation"
  | "gateway_credential";

export interface SearchResultItem {
  type: SearchResultType;
  id: string;
  name: string;
  description?: string;
  url: string;
}

export interface SearchResponse {
  results: SearchResultItem[];
  total: number;
}

export interface SearchOptions {
  q: string;
  types?: SearchResultType[];
  limit?: number;
}

export interface SearchModule {
  search(options: SearchOptions): Promise<SearchResponse>;
}

export function createSearchModule(transport: Transport): SearchModule {
  return {
    async search(options: SearchOptions): Promise<SearchResponse> {
      const query: Query = {
        q: options.q,
        limit: options.limit ?? 20,
      };
      if (options.types && options.types.length > 0) {
        query.types = options.types;
      }
      return transport.requestAt<SearchResponse>("/v1/search", {
        method: "GET",
        query,
      });
    },
  };
}