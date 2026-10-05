import{test,expect,type Page}from'@playwright/test';
test('GS7 docente activa la entrega grupal y el alumno la comparte con su grupo',async({page,browser})=>{
 test.skip(process.env.E2E_ACADEMIC!=='1','Requiere Go/PostgreSQL sintético');test.setTimeout(180_000);const origin=process.env.E2E_BASE_URL??'http://localhost:4321';
 async function login(p:Page,user:string){await p.goto('/login');await p.getByLabel('Tu usuario').fill(user);await p.getByLabel('Tu contraseña').fill(user==='profe'?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);await p.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(p).toHaveURL(user==='profe'?/\/teacher$/:/\/courses$/);}
 await login(page,'profe');
 const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];
 const listed=await(await page.request.get(`/api/v1/teacher/courses/${course.id}/activities?page=1`)).json();
 const stamp=Date.now();const team=`Equipo ${stamp}`;
 const created=await(await page.request.post(`/api/v1/teacher/courses/${course.id}/activities`,{headers:{Origin:origin},data:{moduleId:listed.modules[0].id,type:'assignment',title:`Trabajo en equipo ${stamp}`,body:'Resuelvan el reto en equipo.'}})).json();
 expect(created.id).toBeTruthy();
 // GS1: el docente activa el modo grupal en el borrador y sobrevive a la recarga.
 await page.goto(`/teacher/activities/${created.id}`);
 await expect(page.getByLabel('Entrega grupal compartida')).not.toBeChecked();
 await page.getByLabel('Entrega grupal compartida').check();
 await page.getByRole('button',{name:'Guardar modo de entrega'}).click();
 await expect(page.getByLabel('Entrega grupal compartida')).toBeChecked();
 await expect(page.getByText('Ahora mismo la entrega es grupal.')).toBeVisible();
 await page.screenshot({path:'../docs/screenshots/group-mode-tablet.png',fullPage:true,animations:'disabled'});
 // Publicar no es lo que se prueba aquí: se hace por API y se toma la lección publicada.
 const draftActivity=await(await page.request.get(`/api/v1/teacher/activities/${created.id}`)).json();
 const published=await(await page.request.post(`/api/v1/teacher/activities/${created.id}/publish`,{headers:{Origin:origin},data:{version:draftActivity.version}})).json();
 const lessonId=published.lessonId;expect(lessonId).toBeTruthy();
 const lessonVersion=Number(draftActivity.version);
 // Limpieza: un grupo de una ejecución anterior dejaría a Luna dentro de un equipo
 // y el aviso de "sin grupo" no aparecería. Un grupo con entregas no se puede
 // eliminar, así que primero se saca a Luna y después se intenta borrar.
 for(let p=1;p<=5;p++){
   const listed=await(await page.request.get(`/api/v1/teacher/courses/${course.id}/groups?page=${p}`)).json();
   const row=listed.roster.find((r:{alias:string})=>r.alias==='Luna');
   for(const g of listed.items) if(String(g.name).startsWith('Equipo')){
     if(row&&row.groupId===g.id) await page.request.delete(`/api/v1/teacher/courses/${course.id}/groups/${g.id}/members/${row.studentId}`,{headers:{Origin:origin}});
     await page.request.delete(`/api/v1/teacher/courses/${course.id}/groups/${g.id}`,{headers:{Origin:origin}});
   }
   if(listed.items.length<20) break;
 }
 // GS2/GS6: un alumno sin grupo ve un aviso claro y la API se niega sin crear entrega.
 const context=await browser.newContext();const student=await context.newPage();
 await login(student,'luna');
 await student.goto(`/lessons/${lessonId}`);
 await expect(student.getByText(/todavía no tienes grupo en este curso/)).toBeVisible();
 const denied=await student.request.put(`/api/v1/lessons/${lessonId}/submission`,{headers:{Origin:origin},data:{version:0,lessonVersion,body:'Solo'}});
 expect(denied.status()).toBe(409);
 expect((await denied.json()).error.message).toContain('no tienes grupo');
 // El docente le asigna un grupo.
 await page.goto(`/teacher/courses/${course.id}/groups`);
 await page.getByLabel('Nombre del grupo',{exact:true}).fill(team);
 await page.getByRole('button',{name:'Crear grupo'}).click();
 await expect(page.getByRole('heading',{name:team,exact:true})).toBeVisible();
 const lunaForm=page.locator('form[data-member]').first();
 const lunaId=Number(await lunaForm.getAttribute('data-student'));
 await page.getByLabel('Grupo para Luna',{exact:true}).selectOption({label:team});
 await page.getByRole('button',{name:'Añadir al grupo'}).click();
 await expect(page.getByText(`Pertenece a ${team}.`,{exact:true})).toBeVisible();
 // GS2/GS6: ahora la entrega es del grupo y el alumno lo ve en la tarea.
 const draft=await(await student.request.put(`/api/v1/lessons/${lessonId}/submission`,{headers:{Origin:origin},data:{version:0,lessonVersion,body:'Trabajo compartido del equipo'}})).json();
 expect(draft.groupName).toBe(team);
 const sent=await(await student.request.post(`/api/v1/lessons/${lessonId}/submission/submit`,{headers:{Origin:origin},data:{version:draft.version}})).json();
 expect(sent.status).toBe('submitted');
 await student.goto(`/lessons/${lessonId}`);
 await expect(student.getByText(new RegExp(`esta entrega es de ${team}`))).toBeVisible();
 await expect(student.getByText(new RegExp(`Entrega del grupo ${team}`))).toBeVisible();
 // GS5/GS6: el docente ve el grupo en la entrega, por API y en la bandeja.
 const inbox=await(await page.request.get(`/api/v1/teacher/activities/${created.id}/submissions?page=1&submissionId=${sent.id}`)).json();
 expect(inbox.items[0].groupName).toBe(team);
 await page.goto(`/teacher/activities/${created.id}?submissionId=${sent.id}`);
 await expect(page.getByText(new RegExp(`Entrega del grupo ${team}`))).toBeVisible();
 // GI2: el docente califica miembro a miembro; la vía individual rechaza el equipo.
 await page.goto(`/teacher/activities/${created.id}?submissionId=${sent.id}`);
 await expect(page.getByRole('heading',{name:/Calificación de Luna/})).toBeVisible();
 await page.locator(`#score-${sent.id}-${lunaId}`).fill('70');
 await page.locator(`#feedback-${sent.id}-${lunaId}`).fill('Buen trabajo en equipo, Luna.');
 await page.getByRole('button',{name:'Guardar calificación'}).click();
 // Guardar un borrador no recarga: el formulario lo confirma con su versión.
 await expect(page.getByText(/Borrador guardado · versión 1/).first()).toBeVisible();
 const refused=await page.request.put(`/api/v1/teacher/submissions/${sent.id}/grade`,{headers:{Origin:origin},data:{version:0,score:90,feedback:'toda la clase'}});
 expect(refused.status()).toBe(409);
 await page.getByRole('button',{name:'Publicar devolución'}).click();
 await expect(page.getByText('Devolución publicada').first()).toBeVisible();
 // GI3: la alumna ve su propia nota publicada.
 await student.goto(`/lessons/${lessonId}`);
 await expect(student.getByText('Calificación: 70 / 100')).toBeVisible();
 await expect(student.getByText('Buen trabajo en equipo, Luna.')).toBeVisible();
 // FD1/FD3: el docente adjunta un archivo a la devolución y solo su dueña lo descarga.
 await page.goto(`/teacher/activities/${created.id}?submissionId=${sent.id}`);
 await page.locator(`#feedback-file-${sent.id}-${lunaId}`).setInputFiles({name:"devolucion.txt",mimeType:"text/plain",buffer:Buffer.from("Muy bien, Luna.")});
 await page.getByRole('button',{name:'Subir devolución'}).click();
 await expect(page.getByText('devolucion.txt')).toBeVisible();
 await student.goto(`/lessons/${lessonId}`);
 const feedbackLink=student.getByRole('link',{name:'Descargar devolucion.txt',exact:true});
 await expect(feedbackLink).toBeVisible();
 const feedbackDownload=await student.request.get((await feedbackLink.getAttribute('href'))!);
 expect(feedbackDownload.status()).toBe(200);
 expect(await feedbackDownload.text()).toBe('Muy bien, Luna.');
 const otherStudent=await page.request.get(`/api/v1/lessons/${lessonId}/feedback/${(await feedbackLink.getAttribute('href'))!.split('/').pop()}`);
 expect([403,404]).toContain(otherStudent.status());
 // GI5: el historial de notas registra al miembro calificado.
 const revisions=await(await page.request.get(`/api/v1/teacher/activities/${created.id}/submissions?page=1&submissionId=${sent.id}`)).json();
 expect(revisions.items[0].members[0].grade.status).toBe('published');
 // Deja el curso como estaba: el grupo con entregas no se borra, se vacía.
 const groupsNow=await(await page.request.get(`/api/v1/teacher/courses/${course.id}/groups?page=1`)).json();
 const mine=groupsNow.items.find((g:{name:string})=>g.name===team);
 if(mine){
   const removed=await page.request.delete(`/api/v1/teacher/courses/${course.id}/groups/${mine.id}/members/${lunaId}`,{headers:{Origin:origin}});
   expect(removed.status()).toBe(200);
   const kept=await page.request.delete(`/api/v1/teacher/courses/${course.id}/groups/${mine.id}`,{headers:{Origin:origin}});
   expect(kept.status()).toBe(409);
 }
 expect(lunaId).toBeGreaterThan(0);
 await context.close();
});
