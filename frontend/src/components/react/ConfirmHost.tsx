import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ComponentType,
} from "react";
import type { ConfirmOptions } from "@/lib/confirm";

// El diálogo (Radix + shadcn) se descarga solo al primer uso: las páginas
// autenticadas no pagan su costo si nadie confirma una acción destructiva.
type DialogProps = { options: ConfirmOptions; onFinish: (ok: boolean) => void };

export default function ConfirmHost() {
  const [options, setOptions] = useState<ConfirmOptions | null>(null);
  const [Dialog, setDialog] = useState<ComponentType<DialogProps> | null>(null);
  const resolver = useRef<((ok: boolean) => void) | null>(null);

  const finish = useCallback((ok: boolean) => {
    setOptions(null);
    resolver.current?.(ok);
    resolver.current = null;
  }, []);

  useEffect(() => {
    window.aulaquestConfirm = async (input) => {
      if (!Dialog) {
        try {
          const module = await import("./ConfirmDialog");
          setDialog(() => module.default);
        } catch {
          return window.confirm(input.description);
        }
      }
      return new Promise<boolean>((resolve) => {
        resolver.current = resolve;
        setOptions(input);
      });
    };
    return () => {
      window.aulaquestConfirm = undefined;
    };
  }, [Dialog]);

  if (!options || !Dialog) return null;
  return <Dialog options={options} onFinish={finish} />;
}
