import{test,expect,type Page}from'@playwright/test';
test('CG7 docente crea grupos, asigna estudiantes y respeta un grupo por curso',async({page})=>{
 test.skip(process.env.E2E_ACADEMIC!=='1','Requiere Go/PostgreSQL sintético');test.setTimeout(120_000);const origin=process.env.E2E_BASE_URL??'http://localhost:4321';
 async function login(p:Page,user:string){await p.goto('/login');await p.getByLabel('Tu usuario').fill(user);await p.getByLabel('Tu contraseña').fill(user==='profe'?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);await p.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(p).toHaveURL(user==='profe'?/\/teacher$/:/\/courses$/);}
 // El borrado pide confirmación en el AlertDialog de shadcn.
await login(page,'profe');
 const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];
 const stamp=Date.now();const first=`Equipo Azul ${stamp}`;const second=`Equipo Verde ${stamp}`;
 const groupsPath=`/api/v1/teacher/courses/${course.id}/groups`;
 await page.goto(`/teacher/courses/${course.id}/groups`);
 await expect(page.getByRole('heading',{name:'Grupos del curso',exact:true})).toBeVisible();
 // CG1: crear y rechazar el nombre repetido con un mensaje propio.
 await page.getByLabel('Nombre del grupo',{exact:true}).fill(first);await page.getByRole('button',{name:'Crear grupo'}).click();
 await expect(page.getByRole('heading',{name:first,exact:true})).toBeVisible();
 await page.getByLabel('Nombre del grupo',{exact:true}).fill(first);await page.getByRole('button',{name:'Crear grupo'}).click();
 await expect(page.locator('group-admin [data-status]')).toContainText('Ya existe un grupo con ese nombre');
 await page.getByLabel('Nombre del grupo',{exact:true}).fill(second);await page.getByRole('button',{name:'Crear grupo'}).click();
 await expect(page.getByRole('heading',{name:second,exact:true})).toBeVisible();
 let listed=await(await page.request.get(groupsPath)).json();
 const firstGroup=listed.items.find((g:{name:string})=>g.name===first);const secondGroup=listed.items.find((g:{name:string})=>g.name===second);
 expect(firstGroup).toBeTruthy();expect(secondGroup).toBeTruthy();
 // CG5: solo aparece el roster vinculado e inscrito.
 await expect(page.getByRole('heading',{name:'Luna',exact:true})).toBeVisible();await expect(page.getByRole('heading',{name:'Sol',exact:true})).toHaveCount(0);
 // CG3: añadir a Luna al primer grupo.
 const lunaForm=page.locator('form[data-member]').first();
 const lunaId=Number(await lunaForm.getAttribute('data-student'));
 await page.getByLabel('Grupo para Luna',{exact:true}).selectOption(String(firstGroup.id));await page.getByRole('button',{name:'Añadir al grupo'}).click();
 await expect(page.getByText(`Pertenece a ${first}.`,{exact:true})).toBeVisible();
 // CG3: un segundo grupo del mismo curso no admite al mismo estudiante.
 const conflict=await page.request.put(`${groupsPath}/${secondGroup.id}/members/${lunaId}`,{headers:{Origin:origin}});
 expect(conflict.status()).toBe(409);
 await expect(page.getByText('Pertenece a ' + first + '.',{exact:true})).toBeVisible();
 // CG2: renombrar con revisión.
 await page.locator(`form[data-rename][data-id="${firstGroup.id}"] input[name="name"]`).fill(`${first} renombrado`);
 await page.locator(`form[data-rename][data-id="${firstGroup.id}"] button[type="submit"]`).click();
 await expect(page.getByRole('heading',{name:`${first} renombrado`,exact:true})).toBeVisible();
 await page.screenshot({path:'../docs/screenshots/groups-tablet.png',fullPage:true,animations:'disabled'});
 // CG4: quitar del grupo devuelve el selector.
 await page.getByRole('button',{name:'Quitar del grupo'}).click();
 await expect(page.getByLabel('Grupo para Luna',{exact:true})).toBeVisible();
 listed=await(await page.request.get(groupsPath)).json();
 expect(listed.roster.find((row:{studentId:number})=>row.studentId===lunaId).groupId).toBe(0);
 // CG4: eliminar grupos deja el curso sin grupos y sin tocar el trabajo del alumno.
 const cardFor=(name:string)=>page.locator('article.panel').filter({has:page.getByRole('heading',{name,exact:true})});
 await cardFor(second).getByRole('button',{name:'Eliminar grupo'}).click();await page.getByRole('alertdialog').getByRole('button',{name:'Eliminar',exact:true}).click();
 await expect(page.getByRole('heading',{name:second,exact:true})).toHaveCount(0);
 await cardFor(`${first} renombrado`).getByRole('button',{name:'Eliminar grupo'}).click();await page.getByRole('alertdialog').getByRole('button',{name:'Eliminar',exact:true}).click();
 await expect(page.getByRole('heading',{name:`${first} renombrado`,exact:true})).toHaveCount(0);
 // Los grupos creados por esta prueba desaparecen; el curso puede tener otros.
 listed=await(await page.request.get(groupsPath)).json();
 expect(listed.items.some((g:{name:string})=>[first,second,`${first} renombrado`].includes(g.name))).toBe(false);
 expect(listed.roster.length).toBeGreaterThan(0);
});
