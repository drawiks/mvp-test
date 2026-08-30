export interface WailsRuntime {
  EventsOn(event: string, cb: (...args: unknown[]) => void): void;
  EventsOff(event: string): void;
  OnFileDrop?(callback: (x: number, y: number, paths: string[] | null) => void, useDropTarget?: boolean): string | number;
  OnFileDropOff?(): void;
}

declare global {
  interface Window {
    runtime?: WailsRuntime;
  }
}

export const inWails = (): boolean => typeof window !== "undefined" && !!window.runtime;

/** Subscribe to a Wails event, no-op outside the desktop runtime. */
export function onEvent<T extends unknown[]>(event: string, cb: (...args: T) => void): void {
  if (!inWails()) return;
  window.runtime!.EventsOn(event, cb as (...args: unknown[]) => void);
}

export function offEvent(event: string): void {
  if (!inWails()) return;
  window.runtime!.EventsOff(event);
}