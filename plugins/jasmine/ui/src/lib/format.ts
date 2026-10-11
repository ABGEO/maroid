/** A moment of the hub, in the zone and the form of the browser. */
export function formatMoment(value: string): string {
  return new Date(value).toLocaleString();
}

/** The value of a datetime-local input as a UTC moment, or undefined when empty. */
export function toMoment(local: string): string | undefined {
  return local === "" ? undefined : new Date(local).toISOString();
}

/** A moment as the value of a datetime-local input, in the zone of the browser. */
export function toLocalInput(value: string | Date): string {
  const moment = new Date(value);
  moment.setMinutes(moment.getMinutes() - moment.getTimezoneOffset());

  return moment.toISOString().slice(0, 16);
}

/** The current time as the value of a datetime-local input, the latest it takes. */
export function localNow(): string {
  return toLocalInput(new Date());
}

/** A count of days, with the singular for one. */
export function formatDays(count: number): string {
  return count === 1 ? "1 day" : `${count} days`;
}
