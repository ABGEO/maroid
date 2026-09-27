import type { Page } from '@maroid/plugin-sdk';

export type PageReader<T> = (link?: string) => Promise<Page<T> | null>;

export interface Pager<T> {
  readonly page: Page<T> | null;
  readonly status: 'loading' | 'ready' | 'error';
  readonly hasPrevious: boolean;
  readonly hasNext: boolean;
  /** Reads the page that the pager last asked for, the first page at the start. */
  reload(): Promise<void>;
  next(): Promise<void>;
  previous(): Promise<void>;
}

export function createPager<T>(read: PageReader<T>): Pager<T> {
  let page = $state<Page<T> | null>(null);
  let status = $state<'loading' | 'ready' | 'error'>('loading');
  let current: string | undefined;

  async function load(link?: string): Promise<void> {
    current = link;
    status = 'loading';

    try {
      const answered = await read(link);
      if (answered === null) {
        return;
      }

      page = answered;
      status = 'ready';
    } catch (error) {
      console.error('Failed to load a page', error);
      status = 'error';
    }
  }

  async function follow(link: string | undefined): Promise<void> {
    if (link !== undefined) {
      await load(link);
    }
  }

  return {
    get page() {
      return page;
    },
    get status() {
      return status;
    },
    get hasPrevious() {
      return page?.prev !== undefined;
    },
    get hasNext() {
      return page?.next !== undefined;
    },
    reload: () => load(current),
    next: () => follow(page?.next),
    previous: () => follow(page?.prev)
  };
}
