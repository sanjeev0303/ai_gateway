"use client";

import * as React from "react";

import { buildSeries, breakdownFor, filterRequests, totalsFor } from "./derive";
import { EMPTY_FILTERS, filtersAreEmpty } from "./types";
import type {
  Breakdown,
  DateRange,
  Totals,
  TrafficFilters,
  TrafficSeries,
} from "./types";

const FILTER_KEYS = [
  "personIds",
  "teamIds",
  "projectIds",
  "modelIds",
  "providerIds",
  "kinds",
  "environments",
  "statuses",
] as const;

type FilterKey = (typeof FILTER_KEYS)[number];

interface DemoStoreValue {
  range: DateRange;
  setRange: (range: DateRange) => void;
  filters: TrafficFilters;
  setSearch: (search: string) => void;
  toggleFilter: <K extends FilterKey>(key: K, id: TrafficFilters[K][number]) => void;
  setFilterValues: <K extends FilterKey>(key: K, values: TrafficFilters[K]) => void;
  clearFilter: (key: FilterKey) => void;
  reset: () => void;
  isFiltered: boolean;
  activeFilterCount: number;
  series: TrafficSeries;
  totals: Totals;
  breakdown: Breakdown;
  visibleRequests: ReturnType<typeof filterRequests>;
}

const DemoStoreContext = React.createContext<DemoStoreValue | null>(null);

/**
 * Holds the range and filters every analytics surface shares, so a filter set
 * on Overview is still set when you open Cost. Derivation stays in the pure
 * demo module; this only decides what to derive from.
 */
export function DemoDataProvider({ children }: { children: React.ReactNode }) {
  const [range, setRange] = React.useState<DateRange>("7d");
  const [filters, setFilters] = React.useState<TrafficFilters>(EMPTY_FILTERS);

  const setSearch = React.useCallback((search: string) => {
    setFilters((current) => ({ ...current, search }));
  }, []);

  const toggleFilter = React.useCallback(
    <K extends FilterKey>(key: K, id: TrafficFilters[K][number]) => {
      setFilters((current) => {
        const current_ = current[key] as string[];
        const next = current_.includes(id as string)
          ? current_.filter((value) => value !== id)
          : [...current_, id as string];
        return { ...current, [key]: next } as TrafficFilters;
      });
    },
    [],
  );

  const setFilterValues = React.useCallback(
    <K extends FilterKey>(key: K, values: TrafficFilters[K]) => {
      setFilters((current) => ({ ...current, [key]: values }));
    },
    [],
  );

  const clearFilter = React.useCallback((key: FilterKey) => {
    setFilters((current) => ({ ...current, [key]: [] }) as TrafficFilters);
  }, []);

  const reset = React.useCallback(() => setFilters(EMPTY_FILTERS), []);

  const isFiltered = !filtersAreEmpty(filters);

  const activeFilterCount = React.useMemo(
    () => FILTER_KEYS.reduce((total, key) => total + filters[key].length, 0),
    [filters],
  );

  const series = React.useMemo(() => buildSeries(range, filters), [range, filters]);
  const totals = React.useMemo(() => totalsFor(range, filters), [range, filters]);
  const breakdown = React.useMemo(() => breakdownFor(range, filters), [range, filters]);
  const visibleRequests = React.useMemo(() => filterRequests(filters), [filters]);

  const value = React.useMemo<DemoStoreValue>(
    () => ({
      range,
      setRange,
      filters,
      setSearch,
      toggleFilter,
      setFilterValues,
      clearFilter,
      reset,
      isFiltered,
      activeFilterCount,
      series,
      totals,
      breakdown,
      visibleRequests,
    }),
    [
      range,
      filters,
      setSearch,
      toggleFilter,
      setFilterValues,
      clearFilter,
      reset,
      isFiltered,
      activeFilterCount,
      series,
      totals,
      breakdown,
      visibleRequests,
    ],
  );

  return <DemoStoreContext.Provider value={value}>{children}</DemoStoreContext.Provider>;
}

export function useDemoStore(): DemoStoreValue {
  const value = React.useContext(DemoStoreContext);
  if (!value) {
    throw new Error("useDemoStore must be used inside DemoDataProvider");
  }
  return value;
}
