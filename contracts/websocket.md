# WebSocket v1: /ws/code
Cookie de sesión, Origin exacto, rol student e inscripción verificada al iniciar. Sin credenciales en URL.
Todos los eventos: {v:1,type,requestId,runId?,seq?,payload:{...}}. requestId 1..64 caracteres; servidor genera UUID runId; seq creciente por ejecución desde 1.
Cliente run.start: payload {lessonId: entero positivo,language: javascript|python,code: string <=16384 bytes}; run.cancel: runId obligatorio, payload {}. No user_id.
Servidor run.accepted: payload {simulated:true}; run.stdout/run.stderr: {text:string}; run.finished: {status:completed|cancelled}; run.failed: {code:string,message:string}. Error de protocolo sin ejecución: run.failed sin runId, seq=0. Ejecución activa produce error busy; cancelación ajena invalid_run.
Una ejecución y conexión por usuario; mensajes <=24 KiB, salida <=100 líneas en cliente, cola servidor acotada, timeout 10s, ping cada 20s y lectura 45s. Desconexión/apagado cancela runner y cierra conexión. Sesión revocada/expirada invalida conexión. No reconectar ni reejecutar automáticamente.
Mock emite texto fijo etiquetado como simulación: nunca evalúa ni inspecciona código para inventar una ejecución.
