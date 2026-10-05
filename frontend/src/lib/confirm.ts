export type ConfirmOptions = {
  title?: string;
  description: string;
  confirmLabel?: string;
  cancelLabel?: string;
  destructive?: boolean;
};

declare global {
  interface Window {
    aulaquestConfirm?: (options: ConfirmOptions) => Promise<boolean>;
  }
}

// Uses the shadcn AlertDialog host when it is mounted and falls back to the
// native dialog so a destructive action is never left without confirmation.
export async function confirmAction(input: string | ConfirmOptions): Promise<boolean> {
  const options = typeof input === "string" ? { description: input } : input;
  if (typeof window === "undefined") return false;
  if (window.aulaquestConfirm) return window.aulaquestConfirm(options);
  return window.confirm(options.description);
}
