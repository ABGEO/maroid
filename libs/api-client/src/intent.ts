const KEY_BYTES = 16;

/**
 * The key of one write that creates, held for as long as a form means one
 * record. A repeat of that write reaches the hub under the same key, so the hub
 * answers the earlier result instead of creating a second record.
 */
export interface WriteIntent {
  /**
   * The key for this body. The same body keeps the key of the last call, and a
   * different body takes a new one, so one key never carries two bodies.
   */
  keyFor(body: unknown): string;
  /** Drop the key once a write lands, so the next submit creates a new record. */
  settle(): void;
}

function makeKey(): string {
  const bytes = crypto.getRandomValues(new Uint8Array(KEY_BYTES));

  return Array.from(bytes, (byte) => byte.toString(16).padStart(2, '0')).join('');
}

export function createWriteIntent(): WriteIntent {
  let held: { key: string; body: string } | null = null;

  return {
    keyFor(body: unknown) {
      const serialized = JSON.stringify(body) ?? '';
      if (held === null || held.body !== serialized) {
        held = { key: makeKey(), body: serialized };
      }

      return held.key;
    },

    settle() {
      held = null;
    }
  };
}
