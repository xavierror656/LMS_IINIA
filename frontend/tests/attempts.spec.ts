import {test,expect,type Page} from "@playwright/test";

test("AT6 docente autoriza reentrega, estudiante conserva historial y libro usa último envío",async({page,browser})=>{
 test.skip(process.env.E2E_ACADEMIC!=="1","Requiere Go/PostgreSQL sintético");test.setTimeout(120_000);
 const origin=process.env.E2E_BASE_URL??"http://localhost:4321";
 async function login(p:Page,user:string){await p.goto('/login');await p.getByLabel('Tu usuario').fill(user);await p.getByLabel('Tu contraseña').fill(user==='profe'?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);await p.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(p).toHaveURL(user==='profe'?/\/teacher$/:/\/courses$/);}
 await login(page,'profe');
 const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];
 const module=(await(await page.request.get(`/api/v1/teacher/courses/${course.id}/activities`)).json()).modules[0];
 const created=await page.request.post(`/api/v1/teacher/courses/${course.id}/activities`,{headers:{Origin:origin},data:{moduleId:module.id,type:'assignment',title:`Aprender otra vez ${Date.now()}`,description:'Dos oportunidades',body:'Explica una idea y luego mejórala.'}});expect(created.status()).toBe(201);const activity=await created.json();
 await page.goto(`/teacher/activities/${activity.id}`);await page.getByRole('link',{name:'Configurar intentos',exact:true}).click();
 await page.getByLabel('Máximo de intentos (incluye el primero)').fill('2');await page.getByRole('button',{name:'Guardar borrador',exact:true}).click();
 await expect(page.locator('academic-form')).toHaveAttribute('data-version','2');
 await page.getByRole('link',{name:'Volver a la actividad'}).click();await page.getByRole('button',{name:'Publicar actividad',exact:true}).click();
 await expect(page.getByText('Última versión publicada: 2.',{exact:false})).toBeVisible();
 const published=await(await page.request.get(`/api/v1/teacher/activities/${activity.id}`)).json();
 const context=await browser.newContext({viewport:{width:768,height:1024},reducedMotion:'reduce'});const student=await context.newPage();
 try{
  await login(student,'luna');await student.goto(`/lessons/${published.lessonId}`);
  await student.getByLabel('Escribe tu trabajo').fill('Mi primera idea');await student.getByRole('button',{name:'Guardar borrador',exact:true}).click();await expect(student.locator('[data-status]')).toContainText('Borrador guardado');
  await student.getByRole('button',{name:'Enviar al docente'}).click();await expect(student.getByRole('heading',{name:'Tu trabajo está enviado'})).toBeVisible();
  const first=(await(await student.request.get(`/api/v1/lessons/${published.lessonId}/submission`)).json()).submission;
  await page.goto(`/teacher/activities/${activity.id}?submissionId=${first.id}`);
  await page.getByLabel('Calificación de Luna (0 a 100)').fill('80');await page.getByLabel('Comentario para Luna').fill('Mejora tu explicación');await page.getByRole('button',{name:'Guardar calificación'}).click();
  await expect(page.locator('academic-form[data-kind="grade"] [data-status]')).toContainText('Borrador guardado');await page.getByRole('button',{name:'Publicar devolución'}).click();await expect(page.getByText('Devolución publicada',{exact:true})).toBeVisible();
  await page.getByText('Autorizar otro intento para Luna',{exact:true}).click();await page.getByLabel('Motivo de reapertura').fill('Practica con la devolución');
  await page.getByRole('button',{name:'Autorizar nuevo intento',exact:true}).click();
  await expect.poll(async()=>{const r=await student.request.get(`/api/v1/lessons/${published.lessonId}/submission`);return (await r.json()).submission.attempt;}).toBe(2);
  await student.reload();await expect(student.getByLabel('Escribe tu trabajo')).toHaveValue('');await expect(student.getByText('Intento 2 ·',{exact:false})).toBeVisible();
  await student.getByRole('link',{name:'Ver mis intentos anteriores'}).click();await student.getByRole('link',{name:'Intento 1 · Enviado',exact:true}).click();
  await expect(student.getByText('Mi primera idea',{exact:true})).toBeVisible();await expect(student.getByText('Calificación: 80 / 100',{exact:true})).toBeVisible();await expect(student.getByLabel('Escribe tu trabajo')).toHaveCount(0);
  await expect(student.getByLabel('Tu progreso')).toContainText('Luna');await student.screenshot({path:'../docs/screenshots/attempt-history-tablet.png',fullPage:true,animations:'disabled'});
  await student.getByRole('link',{name:'Volver al intento actual'}).click();await student.getByLabel('Escribe tu trabajo').fill('Mi segunda idea mejorada');await student.getByRole('button',{name:'Guardar borrador',exact:true}).click();await expect(student.locator('[data-status]')).toContainText('Borrador guardado');await student.getByRole('button',{name:'Enviar al docente'}).click();await expect(student.getByRole('heading',{name:'Tu trabajo está enviado'})).toBeVisible();
  const second=(await(await student.request.get(`/api/v1/lessons/${published.lessonId}/submission`)).json()).submission;
  await page.goto(`/teacher/activities/${activity.id}?submissionId=${second.id}`);await expect(page.getByText('Mi segunda idea mejorada',{exact:true})).toBeVisible();
  await page.getByText('Autorizar otro intento para Luna',{exact:true}).click();await page.getByLabel('Motivo de reapertura').fill('No permitido por límite');await page.getByRole('button',{name:'Autorizar nuevo intento',exact:true}).click();await expect(page.locator('academic-form[data-kind="reopen"] [data-status]')).toContainText('Tu texto permanece');await expect(page.getByLabel('Motivo de reapertura')).toHaveValue('No permitido por límite');
  let columnPage=1;let found=false;
  do{const book=await(await page.request.get(`/api/v1/teacher/courses/${course.id}/gradebook?activityPage=${columnPage}`)).json();found=book.activities.some((a:{activityId:number})=>a.activityId===activity.id);if(found){const row=book.rows.find((r:{alias:string})=>r.alias==='Luna');const cell=row.cells.find((c:{activityId:number})=>c.activityId===activity.id);expect(cell.submissionId).toBe(second.id);expect(cell.state).toBe('submitted');break;}if(columnPage*10>=book.totalActivities)break;columnPage++;}while(columnPage<100);
  expect(found).toBeTruthy();
  // Discard only the unsaved test reason through the same UI guard as a user.
  page.once('dialog',dialog=>dialog.accept());await page.goto(`/teacher/courses/${course.id}/gradebook?activityPage=${columnPage}`);
  await expect(page.getByRole('columnheader',{name:published.title,exact:false})).toBeVisible();
 }finally{await context.close();}
});
