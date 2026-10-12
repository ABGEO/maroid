import type { Reading } from "../api";

/** The usage and the value of one meter on one date. */
export interface MeterPoint {
  /** Absent for the earliest reading of a meter. */
  usage?: number;
  value: number;
}

/** The readings of the chart, grouped by date and by meter. */
export interface ChartSeries {
  /** The reading dates, YYYY-MM-DD, the oldest first. */
  dates: string[];
  /** The serial numbers of the meters, sorted. */
  meters: string[];
  /** The point of each meter on each date: points.get(date)?.get(serial). */
  points: Map<string, Map<string, MeterPoint>>;
}

/** The daisyUI classes that draw one meter. */
export interface MeterTone {
  column: string;
  line: string;
  swatch: string;
}

// Tailwind finds a class only where the source names it in full, so no class here
// is built from parts.
const TONES: readonly MeterTone[] = [
  { column: "fill-primary", line: "stroke-primary", swatch: "bg-primary" },
  {
    column: "fill-secondary",
    line: "stroke-secondary",
    swatch: "bg-secondary",
  },
  { column: "fill-accent", line: "stroke-accent", swatch: "bg-accent" },
  { column: "fill-info", line: "stroke-info", swatch: "bg-info" },
];

export function chartSeries(readings: Reading[]): ChartSeries {
  const points = new Map<string, Map<string, MeterPoint>>();
  const meters = new Set<string>();

  for (const reading of readings) {
    meters.add(reading.serial_number);

    let ofDate = points.get(reading.reading_date);
    if (ofDate === undefined) {
      ofDate = new Map();
      points.set(reading.reading_date, ofDate);
    }

    ofDate.set(reading.serial_number, {
      usage: reading.usage,
      value: reading.value,
    });
  }

  return {
    dates: [...points.keys()].sort(),
    meters: [...meters].sort(),
    points,
  };
}

/** The tone of the meter at the index of the sorted serial numbers. */
export function meterTone(index: number): MeterTone {
  return TONES[index % TONES.length];
}
