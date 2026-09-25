/**
 * One page of a collection.
 */
export interface Page<T> {
  /** The rows of this page. It is never null, and an empty collection is `[]`. */
  items: T[];
  /** The address of this page. */
  self: string;
  /** The address of the first page of the same collection. */
  first: string;
  /** Absent on the last page, which is how a reader stops. */
  next?: string;
  /** Absent on the first page. */
  prev?: string;
}
