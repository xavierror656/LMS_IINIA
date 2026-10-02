# Admin demo — LMS-014
Solo servidor Node local, no implementado en Go. Cuenta admin / aulaquest-demo. Sesión HttpOnly existente; User demo amplía role con admin.
GET /api/v1/admin/overview: sesión admin, devuelve {users:[{id,username,alias,role}],courses:[{id,slug,title,description,icon,lessonCount}],plugins:2}. Catálogo sintético acotado, sin credenciales ni identificadores de sesión.
POST /api/v1/admin/plugins/:id: sesión admin, Origin local y cuerpo/resultado idéntico a contracts/demo-plugins.md.
GET /api/v1/plugins admite también admin. Endpoints teacher siguen exclusivos de teacher. Admin no recibe progreso estudiantil ni ejecuta laboratorio como estudiante.
Errores 401/403/400/404/500 con formato demo existente, no-store. UI /admin y /admin/plugins solo en PUBLIC_DEMO_MODE=true. Consulta de usuarios/cursos sin edición ni borrado.
