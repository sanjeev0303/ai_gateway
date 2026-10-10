"use client";

import * as React from "react";
import { useSearchParams, useRouter, usePathname } from "next/navigation";
import { DATE_RANGES, type DateRange } from "@/lib/constants";

const DATE_RANGE_PARAM = "range";
const DEFAULT_RANGE: DateRange = "7d";

interface UseDateRangeReturn {
  range: DateRange;
  setRange: (range: DateRange) => void;
  isDefault: boolean;
}

export function useDateRange(): UseDateRangeReturn {
  const searchParams = useSearchParams();
  const router = useRouter();
  const pathname = usePathname();

  const initialRange = React.useMemo(() => {
    const param = searchParams.get(DATE_RANGE_PARAM);
    if (param && DATE_RANGES.includes(param as DateRange)) {
      return param as DateRange;
    }
    return DEFAULT_RANGE;
  }, [searchParams]);

  const [range, setRangeState] = React.useState<DateRange>(initialRange);

  React.useEffect(() => {
    const param = searchParams.get(DATE_RANGE_PARAM);
    if (param && DATE_RANGES.includes(param as DateRange)) {
      setRangeState(param as DateRange);
    } else {
      setRangeState(DEFAULT_RANGE);
    }
  }, [searchParams]);

  const setRange = React.useCallback(
    (newRange: DateRange) => {
      setRangeState(newRange);

      const params = new URLSearchParams(searchParams.toString());
      if (newRange === DEFAULT_RANGE) {
        params.delete(DATE_RANGE_PARAM);
      } else {
        params.set(DATE_RANGE_PARAM, newRange);
      }

      const newUrl = `${pathname}?${params.toString()}`;
      router.push(newUrl, { scroll: false });
    },
    [router, pathname, searchParams]
  );

  return {
    range,
    setRange,
    isDefault: range === DEFAULT_RANGE,
  };
}

export function useDateRangeFromUrl(): DateRange {
  const searchParams = useSearchParams();
  const param = searchParams.get(DATE_RANGE_PARAM);
  if (param && DATE_RANGES.includes(param as DateRange)) {
    return param as DateRange;
  }
  return DEFAULT_RANGE;
}