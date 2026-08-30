import { mount, unmount, type Component } from 'svelte';
import type { RouteMounter } from './types.js';

export function defineRoute<P extends Record<string, unknown> = Record<string, never>>(
  load: () => Promise<{ default: Component<P> }>,
): RouteMounter {
  return (target, props = {}) => {
    let instance: ReturnType<typeof mount> | undefined;
    let cancelled = false;

    load().then(({ default: component }) => {
      if (cancelled) return;
      instance = mount(component, { target, props: props as P });
    });

    return () => {
      cancelled = true;
      if (instance) unmount(instance);
    };
  };
}
