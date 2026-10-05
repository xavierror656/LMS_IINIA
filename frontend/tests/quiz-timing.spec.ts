import{test,expect,type Page}from'@playwright/test';
test('QT7-QT13 fechas, temporizador, excepción de tiempo y revisión tras cierre',async({page,browser})=>{
 test.skip(process.env.E2E_ACADEMIC!=='1','Requiere Go/PostgreSQL sintético');test.setTimeout(150_000);const origin=process.env.E2E_BASE_URL??'http://localhost:4321';
 async function login(p:Page,user:string){await p.goto('/login');await p.getByLabel('Tu usuario').fill(user);await p.getByLabel('Tu contraseña').fill(user==='profe'?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);await p.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(p).toHaveURL(user==='profe'?/\/teacher$/:/\/courses$/);}
 const date=(offset:number)=>new Date(Date.now()+offset*60_000).toISOString().slice(0,19);
 await login(page,'profe');const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];const stamp=Date.now();
 const module=(await(await page.request.get(`/api/v1/teacher/courses/${course.id}/activities`)).json()).modules[0];
 const qr=await page.request.post(`/api/v1/teacher/courses/${course.id}/questions`,{headers:{Origin:origin},data:{name:`Ave ${stamp}`,type:'single_choice',prompt:'Elige el animal que vuela.',options:['Pájaro','Pez'],correctChoices:[0],acceptedAnswers:[],caseSensitive:false,explanation:'Solución privada del cuestionario.'}});
 expect(qr.status()).toBe(201);const q1=await qr.json();
 const created=await page.request.post(`/api/v1/teacher/courses/${course.id}/activities`,{headers:{Origin:origin},data:{moduleId:module.id,type:'quiz',title:`Cuestionario con tiempo ${stamp}`,description:'Prueba de calendario y temporizador',body:'Responde con calma y guarda antes de enviar.'}});
 expect(created.status()).toBe(201);const activityId=(await created.json()).id;const ap=`/teacher/activities/${activityId}`;
 // QT9: el compositor acepta el tiempo máximo por intento y la revisión tras el cierre.
 await page.goto(ap);await page.getByRole('link',{name:'Configurar cuestionario',exact:true}).click();
 await page.getByLabel('Máximo de intentos',{exact:true}).fill('2');await page.getByLabel(/Tiempo máximo para resolver/).fill('1');await page.getByLabel('Cuándo mostrar las soluciones').selectOption('after_close');
 await page.getByLabel(`${q1.content.name} · versión 1`,{exact:true}).check();await page.getByRole('button',{name:'Guardar configuración del cuestionario'}).click();
 await expect(page.locator('quiz-composer')).toHaveAttribute('data-version','2');await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('button',{name:'Publicar actividad'}).click();
 // QT7: el calendario del cuestionario se guarda en UTC con vocabulario propio.
 await page.getByRole('link',{name:'Configurar fechas',exact:true}).click();await expect(page.getByRole('heading',{name:'Fechas del cuestionario',exact:true})).toBeVisible();
 await page.getByLabel('Apertura (UTC)',{exact:true}).fill(date(-60));await page.getByLabel('Vencimiento (UTC)',{exact:true}).fill(date(30));await page.getByLabel('Cierre (UTC)',{exact:true}).fill(date(60));
 await page.getByRole('button',{name:'Guardar borrador'}).click();await expect(page.locator('academic-form[data-kind="schedule"]')).toHaveAttribute('data-version','3');
 await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('button',{name:'Publicar actividad'}).click();
 await expect(page.getByRole('link',{name:'Gestionar tiempos',exact:true})).toBeVisible();
 let activity=await(await page.request.get(`/api/v1${ap}`)).json();
 const context=await browser.newContext({viewport:{width:768,height:1024},reducedMotion:'reduce'});const student=await context.newPage();
 try{
  await login(student,'luna');await student.goto(`/lessons/${activity.lessonId}`);const before=await(await student.request.get('/api/v1/me/progress')).json();
  // QT13: el alumno ve el calendario, el límite publicado y el tiempo que indica el servidor.
  await expect(student.getByText(/Tiempo máximo: 1 minuto por intento/)).toBeVisible();
  await student.getByRole('button',{name:'Comenzar intento 1',exact:true}).click();await expect(student.getByRole('heading',{name:'Intento 1',exact:true})).toBeVisible();
  const remaining=Number(await student.locator('quiz-attempt').getAttribute('data-remaining'));expect(remaining).toBeGreaterThan(55);expect(remaining).toBeLessThanOrEqual(60);
  await expect(student.locator('quiz-attempt [data-clock]')).toHaveText(/^\d{1,2}:\d{2}$/);
  await student.getByLabel('Pájaro',{exact:true}).check();await student.getByRole('button',{name:'Guardar respuestas',exact:true}).click();await expect(student.locator('quiz-attempt [data-status]')).toContainText('Respuestas guardadas');
  const secrets=await(await student.request.get(`/api/v1/lessons/${activity.lessonId}/quiz`)).text();expect(secrets).not.toContain('correctChoices');expect(secrets).not.toContain('Solución privada');
  await student.screenshot({path:'../docs/screenshots/quiz-timing-student-tablet.png',fullPage:true,animations:'disabled'});
  // QT12: la nota la calcula el servidor y after_close no revela antes del cierre.
  const attemptId=await student.locator('quiz-attempt').getAttribute('data-id');const version=Number(await student.locator('quiz-attempt').getAttribute('data-version'));
  // QT12: el navegador no puede declarar su nota; un campo desconocido se rechaza.
  const tampered=await student.request.post(`/api/v1/lessons/${activity.lessonId}/quiz/attempts/${attemptId}/submit`,{headers:{Origin:origin},data:{version,score:0}});
  expect(tampered.status()).toBe(400);
  const submitted=await student.request.post(`/api/v1/lessons/${activity.lessonId}/quiz/attempts/${attemptId}/submit`,{headers:{Origin:origin},data:{version}});
  expect(submitted.ok()).toBeTruthy();expect((await submitted.json()).score).toBe(100);
  await student.reload();await expect(student.getByText('Resultado de este intento: 100 / 100',{exact:true})).toBeVisible();
  await expect(student.getByRole('heading',{name:/Revisión de la pregunta/})).toHaveCount(0);
  // QT11: ningún alumno puede concederse tiempo ni fechas.
  const me=await(await student.request.get('/api/v1/auth/session')).json();
  const forbidden=await student.request.put(`/api/v1${ap}/quiz-extensions/${me.id}`,{headers:{Origin:origin},data:{version:0,dueAt:null,closesAt:null,extraSeconds:600,reason:'Sin permiso'}});
  expect(forbidden.status()).toBe(403);
  // QT11: el docente concede minutos extra y cierre ampliado solo a su alumna vinculada.
  await page.getByRole('link',{name:'Gestionar tiempos',exact:true}).click();await expect(page.getByText('Tiempo máximo publicado: 1 minuto por intento.',{exact:true})).toBeVisible();
  await expect(page.getByRole('heading',{name:'Luna',exact:true})).toBeVisible();await expect(page.getByRole('heading',{name:'Sol',exact:true})).toHaveCount(0);
  await page.getByLabel('Nuevo cierre (UTC)',{exact:true}).fill(date(90));await page.getByLabel('Minutos extra para resolver (0 a 240)',{exact:true}).fill('5');await page.getByLabel('Motivo de la excepción o revocación').fill('Necesita más tiempo para terminar');
  await page.getByRole('button',{name:'Guardar excepción de tiempo'}).click();await expect(page.locator('academic-form[data-kind="quiz-extension"]')).toHaveAttribute('data-version','1');
  await page.screenshot({path:'../docs/screenshots/quiz-timing-extensions-tablet.png',fullPage:true,animations:'disabled'});
  // La excepción amplía el intento siguiente: un minuto publicado más cinco concedidos.
  await student.reload();await expect(student.getByText(/Tienes una excepción de tiempo/)).toBeVisible();
  await student.getByRole('button',{name:'Comenzar intento 2',exact:true}).click();await expect(student.getByRole('heading',{name:'Intento 2',exact:true})).toBeVisible();
  const extended=Number(await student.locator('quiz-attempt').getAttribute('data-remaining'));expect(extended).toBeGreaterThan(295);expect(extended).toBeLessThanOrEqual(360);
  await student.getByLabel('Pájaro',{exact:true}).check();await student.getByRole('button',{name:'Guardar respuestas',exact:true}).click();await expect(student.locator('quiz-attempt [data-status]')).toContainText('Respuestas guardadas');
  await student.getByRole('button',{name:'Enviar cuestionario',exact:true}).click();await expect(student.getByText('Resultado de este intento: 100 / 100',{exact:true})).toBeVisible();
  // QT8: adelantar la apertura y el cierre bloquea el comienzo sin quitar el acceso de lectura.
  await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('link',{name:'Configurar fechas',exact:true}).click();
  await page.getByLabel('Apertura (UTC)',{exact:true}).fill(date(30));await page.getByLabel('Vencimiento (UTC)',{exact:true}).fill(date(60));await page.getByLabel('Cierre (UTC)',{exact:true}).fill(date(90));
  await page.getByRole('button',{name:'Guardar borrador'}).click();await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('button',{name:'Publicar actividad'}).click();
  await student.reload();await expect(student.getByText(/todavía no abre/)).toBeVisible();await expect(student.getByRole('button',{name:/Comenzar intento/})).toHaveCount(0);
  // QT11: revocar la excepción deja de ampliar el cierre del cuestionario.
  await page.getByRole('link',{name:'Gestionar tiempos',exact:true}).click();
  await page.getByLabel('Nuevo cierre (UTC)',{exact:true}).fill('');await page.getByLabel('Minutos extra para resolver (0 a 240)',{exact:true}).fill('0');await page.getByLabel('Motivo de la excepción o revocación').fill('La excepción terminó');
  await page.getByRole('button',{name:'Guardar excepción de tiempo'}).click();await expect(page.locator('academic-form[data-kind="quiz-extension"]')).toHaveAttribute('data-version','2');
  await student.reload();await expect(student.getByText(/Tienes una excepción de tiempo/)).toHaveCount(0);
  // QT8: con el cierre en el pasado el cuestionario queda cerrado sin quitar la lectura.
  await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('link',{name:'Configurar fechas',exact:true}).click();
  await page.getByLabel('Apertura (UTC)',{exact:true}).fill(date(-120));await page.getByLabel('Vencimiento (UTC)',{exact:true}).fill(date(-10));await page.getByLabel('Cierre (UTC)',{exact:true}).fill(date(-5));
  await page.getByRole('button',{name:'Guardar borrador'}).click();await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('button',{name:'Publicar actividad'}).click();
  await student.reload();await expect(student.getByText(/ya cerró/)).toBeVisible();await expect(student.getByRole('button',{name:/Comenzar intento/})).toHaveCount(0);
  // La revisión tras el cierre no se revela retroactivamente porque el intento congeló su propio cierre.
  await expect(student.getByRole('heading',{name:/Revisión de la pregunta/})).toHaveCount(0);
  // El temporizador y el calendario no conceden recompensas.
  activity=await(await page.request.get(`/api/v1${ap}`)).json();expect(activity.quizConfig.timeLimitSeconds).toBe(60);expect(activity.quizConfig.reviewPolicy).toBe('after_close');
  const after=await(await student.request.get('/api/v1/me/progress')).json();expect(after.xp).toBe(before.xp);expect(after.gems).toBe(before.gems);expect(after.stars).toBe(before.stars);expect(after.completed).toBe(before.completed+1);
  await page.goto(`/teacher/activities/${activityId}/quiz-results`);await expect(page.getByRole('link',{name:/Revisar intento 1 de Luna/})).toBeVisible();await expect(page.getByRole('link',{name:/Revisar intento 2 de Luna/})).toBeVisible();
 }finally{await context.close();}
});
test('QT13 el contador del cliente deja de permitir guardar al agotarse el tiempo',async({page,browser})=>{
 test.skip(process.env.E2E_ACADEMIC!=='1','Requiere Go/PostgreSQL sintético');test.setTimeout(120_000);const origin=process.env.E2E_BASE_URL??'http://localhost:4321';
 async function login(p:Page,user:string){await p.goto('/login');await p.getByLabel('Tu usuario').fill(user);await p.getByLabel('Tu contraseña').fill(user==='profe'?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);await p.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(p).toHaveURL(user==='profe'?/\/teacher$/:/\/courses$/);}
 const iso=(offset:number)=>new Date(Date.now()+offset*60_000).toISOString();
 await login(page,'profe');const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];const stamp=Date.now();
 const module=(await(await page.request.get(`/api/v1/teacher/courses/${course.id}/activities`)).json()).modules[0];
 const qr=await page.request.post(`/api/v1/teacher/courses/${course.id}/questions`,{headers:{Origin:origin},data:{name:`Reloj ${stamp}`,type:'true_false',prompt:'El sol es una estrella.',options:['Verdadero','Falso'],correctChoices:[0],acceptedAnswers:[],caseSensitive:false,explanation:'Solución privada del reloj.'}});
 expect(qr.status()).toBe(201);const q=await qr.json();
 const created=await page.request.post(`/api/v1/teacher/courses/${course.id}/activities`,{headers:{Origin:origin},data:{moduleId:module.id,type:'quiz',title:`Reloj ${stamp}`,description:'Contador del cliente',body:'Responde antes de que se agote el tiempo.'}});
 expect(created.status()).toBe(201);const activity=await created.json();const ap=`/teacher/activities/${activity.id}`;
 let current=await(await page.request.put(`/api/v1${ap}/quiz-config`,{headers:{Origin:origin},data:{version:activity.version,maxAttempts:1,weight:1,items:[{questionId:q.id,version:1,weight:1}],gradePolicy:'last',reviewPolicy:'never',timeLimitSeconds:60}})).json();
 current=await(await page.request.put(`/api/v1${ap}/schedule`,{headers:{Origin:origin},data:{version:current.version,opensAt:iso(-60),dueAt:iso(30),closesAt:iso(60)}})).json();
 // La lección solo existe después de publicar la actividad.
 const published=await page.request.post(`/api/v1${ap}/publish`,{headers:{Origin:origin},data:{version:current.version}});
 expect(published.ok()).toBeTruthy();const lessonId=(await published.json()).lessonId as number;expect(lessonId).toBeGreaterThan(0);
 const context=await browser.newContext({viewport:{width:768,height:1024},reducedMotion:'reduce'});const student=await context.newPage();
 try{
  await login(student,'luna');
  await student.goto(`/lessons/${lessonId}`);
  await student.getByRole('button',{name:'Comenzar intento 1',exact:true}).click();
  await expect(student.getByRole('heading',{name:'Intento 1',exact:true})).toBeVisible();
  await expect(student.locator('quiz-attempt [data-clock]')).toHaveText(/^[01]:[0-5]\d$/);
  await student.getByLabel('Verdadero',{exact:true}).check();
  await student.getByRole('button',{name:'Guardar respuestas',exact:true}).click();await expect(student.locator('quiz-attempt [data-status]')).toContainText('Respuestas guardadas');
  // El minuto real deja que el contador del cliente y el plazo del servidor coincidan.
  await expect(student.locator('quiz-attempt [data-status]')).toContainText('Se agotó tu tiempo',{timeout:90_000});
  await expect(student.locator('quiz-attempt [data-clock]')).toHaveText('0:00');
  await expect(student.getByRole('button',{name:'Guardar respuestas',exact:true})).toBeDisabled();
  await expect(student.getByRole('button',{name:'Enviar cuestionario',exact:true})).toBeDisabled();
  await student.screenshot({path:'../docs/screenshots/quiz-timing-expired-tablet.png',fullPage:true,animations:'disabled'});
  // El servidor cierra el intento con las respuestas guardadas y su nota canónica.
  // El contador del cliente es informativo, así que se espera a que el propio
  // servidor confirme el cierre antes de reconciliar la página.
  await expect(async()=>{const state=await(await student.request.get(`/api/v1/lessons/${lessonId}/quiz`)).json();expect(state.attempt.status).toBe('finished');}).toPass({timeout:20_000});
  await student.reload();
  await expect(student.getByText('Resultado de este intento: 100 / 100',{exact:true})).toBeVisible({timeout:30_000});
  await expect(student.getByRole('button',{name:/Comenzar intento/})).toHaveCount(0);
  await expect(student.getByRole('heading',{name:/Revisión de la pregunta/})).toHaveCount(0);
 }finally{await context.close();}
});
