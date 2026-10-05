# Incremento 10: límites por usuario y proxy confiable (TSEC1)

Especificación previa al código. TSEC1, relacionado con LMS-021/022. Nace de un fallo observado: el limitador global por IP produjo 429 al encadenar suites de pruebas, y en un aula real todos los estudiantes detrás de una misma salida NAT comparten esa cuota. No debilita ninguna protección para pasar pruebas y no acredita el 98 %.

## Problema y alcance

El limitador global actual usa la IP del par como clave con 180 solicitudes por minuto. Una página SSR privada se despliega en varias llamadas al API, así que un aula con treinta estudiantes tras una sola IP agota la cuota colectiva y recibe 429 en tráfico legítimo. Además, `infra/nginx.conf` no reenvía `X-Forwarded-For`: bajo Compose todos los clientes comparten la IP del contenedor proxy, con lo que el problema es total, y confiar en esa cabecera sin verificar el par permitiría eludir el límite rotándola.

Este incremento cambia únicamente **quién** consume cada cuota y **cómo** se obtiene la IP real. No cambia la autorización, ni los contratos, ni el esquema de datos.

## Reglas y aceptación

- RL1: el tráfico autenticado se limita por **usuario**. La clave es el identificador de la sesión ya resuelta, así que no añade consultas. La cuota de un estudiante no consume la de otro ni la del resto del aula, aunque compartan salida a Internet. Presupuesto configurable; por defecto 600 solicitudes por minuto y por usuario, dimensionado para el despliegue SSR real de una página de lección.
- RL2: el tráfico no autenticado se limita por **IP real**. Cubre el grupo público `/api/v1` —hoy el inicio de sesión— con 180 solicitudes por minuto por defecto, valor idéntico al anterior, de modo que nada se relaja. Los límites específicos existentes se conservan sin cambios: inicio de sesión 10 por minuto, adjuntos 20 por minuto por usuario e intentos de actividad 30 por minuto por usuario.
- RL3: la IP real solo se toma de cabeceras de proxy **confiable**. `TRUSTED_PROXIES` es una lista separada por comas de direcciones o rangos CIDR; vacía por defecto. Si el par directo pertenece a esa lista, se honra `X-Forwarded-For`; si no, se usa el par y la cabecera se descarta. La clave se valida siempre como IP real: un valor no analizable cae al par directo, de modo que rotar la cabecera desde un cliente no confiable no cambia la cuota ni permite eludirla.
- RL4: el proxy reenvía la IP del cliente **sobrescribiendo** la cabecera (`$remote_addr`), nunca añadiendo (`$proxy_add_x_forwarded_for`), para que el cliente no pueda inyectar valores. Compose configura el rango de la red interna como proxy confiable; en producción debe fijarse el rango real del ingress.
- RL5: la respuesta 429 informa `Retry-After` con los segundos de la ventana —el valor conservador que un cliente bien educado debe esperar—, además del mensaje comprensible que ya existe. El cliente no cambia.
- RL6: `/healthz` y `/readyz` dejan de estar limitados a propósito: un sondeo de orquestación no debe consumir la cuota de nadie ni provocar falsos no saludables. La ruta WebSocket conserva un límite por IP, ya que antes estaba cubierta por el limitador global y no se retira ninguna protección.
- RL7: verificación. Integración: dos usuarios distintos con la misma IP tienen presupuestos independientes y agotar uno no afecta al otro; el mismo usuario sostiene una ráfaga que antes producía 429; `X-Forwarded-For` se ignora desde un par no confiable y se honra desde uno confiable; una cabecera con valor inválido cae al par; el inicio de sesión conserva su límite estricto; toda respuesta 429 incluye `Retry-After`. Navegador: encadenar las suites intensivas existentes sin 429. Documentar en README, `.env.example` y un documento de validación propio.

## Contratos y datos

Sin cambios de esquema ni de OpenAPI: son límites de transporte. Se añaden tres variables de entorno con validación al arrancar —`RATE_LIMIT_USER`, `RATE_LIMIT_IP` y `TRUSTED_PROXIES`—, con valores por defecto que no relajan lo existente y error explícito si el valor es inválido en lugar de silenciarlo. `infra/nginx.conf` y `docker-compose.yml` se alinean con RL4.

## Validación y límites

No se acreditan límites distribuidos entre réplicas: el almacén del limitador es en memoria por proceso, igual que antes; añadir réplicas exigiría un almacén compartido y queda pendiente. Tampoco se acredita defensa contra un atacante con muchas cuentas o muchas sesiones, ni sustituye a un WAF o a límites en el borde: este incremento corrige el reparto entre usuarios legítimos y la confianza en la IP, no añade protección perimetral.
