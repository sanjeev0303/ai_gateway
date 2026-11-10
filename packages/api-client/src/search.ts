import type { components } from "@ai-governance/contracts";
import type { Transport, Query } from "./client";

export type SearchResultType = components["schemas"]["SearchResultType"];
export type SearchResultItem = components["schemas"]["SearchResultItem"];
export type SearchResponse = components["schemas"]["SearchResponse"];

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