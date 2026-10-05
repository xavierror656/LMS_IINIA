import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import type { ConfirmOptions } from "@/lib/confirm";

export default function ConfirmDialog({
  options,
  onFinish,
}: {
  options: ConfirmOptions;
  onFinish: (ok: boolean) => void;
}) {
  return (
    <AlertDialog
      open
      onOpenChange={(open: boolean) => {
        if (!open) onFinish(false);
      }}
    >
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>
            {options.title ?? "¿Confirmas la acción?"}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {options.description}
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel
            className="min-h-11 px-5"
            onClick={() => onFinish(false)}
          >
            {options.cancelLabel ?? "Cancelar"}
          </AlertDialogCancel>
          <AlertDialogAction
            variant={options.destructive ? "destructive" : "default"}
            className="min-h-11 px-5"
            onClick={() => onFinish(true)}
          >
            {options.confirmLabel ?? "Confirmar"}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
