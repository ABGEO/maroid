/** One series of a column and line chart. Each map keys a value by its date, YYYY-MM-DD. */
export interface ChartSeries {
	key: string;
	/** The values drawn as columns on the left axis. A date with no value draws no column. */
	columns: ReadonlyMap<string, number>;
	/** The values drawn as a line on the right axis. */
	line: ReadonlyMap<string, number>;
	/** The fill class of the columns, for example `fill-primary`. */
	columnClass: string;
	/** The stroke class of the line, for example `stroke-accent`. */
	lineClass: string;
}
