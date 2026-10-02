import {test,expect} from '@playwright/test';

test('QB6 banco versionado, cuatro evaluadores, recuperación y archivo',async({page})=>{
 test.skip(process.env.E2E_ACADEMIC!=='1','Requiere Go/PostgreSQL sintético');test.setTimeout(120_000);
 await page.goto('/login');await page.getByLabel('Tu usuario').fill('profe');await page.getByLabel('Tu contraseña').fill(process.env.E2E_TEACHER_PASSWORD!);await page.getByRole('button',{name:'Entrar a mi aventura'}).click();await expect(page).toHaveURL(/\/teacher$/);
 const course=(await(await page.request.get('/api/v1/teacher/courses')).json()).items[0];const bank=`/teacher/courses/${course.id}/questions`;
 await page.goto(`/teacher/courses/${course.id}`);await page.getByRole('link',{name:'Banco de preguntas',exact:false}).click();await expect(page.getByRole('heading',{name:'Banco de preguntas',exact:true})).toBeVisible();
 async function begin(kind:string,name:string){await page.goto(bank);await page.getByLabel('Nombre interno').fill(name);await page.getByLabel('Tipo de pregunta').selectOption(kind);await page.getByLabel('Enunciado',{exact:true}).fill('Elige la respuesta de esta prueba.');await page.getByLabel('Explicación de la solución').fill('Explicación privada del docente.');}
 async function create(){await page.getByRole('button',{name:'Crear pregunta',exact:true}).click();await expect(page).toHaveURL(/\/teacher\/questions\/\d+$/);}
 async function score(expected:number){await page.getByRole('button',{name:'Evaluar respuesta de prueba',exact:true}).click();await expect(page.locator('question-preview [data-result]')).toContainText(`Resultado de prueba: ${expected} / 100.`);}
 const name=`Pregunta única ${Date.now()}`;await begin('single_choice',name);
 await page.getByLabel('Opción 1',{exact:true}).fill('Pájaro');await page.getByLabel('Opción 2',{exact:true}).fill('Pez');await page.getByLabel('Correcta 1',{exact:true}).check();
 // Transport failure must preserve the author's inputs and permit recovery.
 await page.route('**/api/v1/teacher/courses/*/questions',route=>route.abort());await page.getByRole('button',{name:'Crear pregunta',exact:true}).click();await expect(page.locator('question-form [data-status]')).toContainText('Tus entradas permanecen');await expect(page.getByLabel('Opción 1',{exact:true})).toHaveValue('Pájaro');await page.unroute('**/api/v1/teacher/courses/*/questions');await create();
 const questionURL=page.url();await page.getByLabel('Pájaro',{exact:true}).check();await score(100);await page.getByLabel('Pez',{exact:true}).check();await score(0);
 await page.getByLabel('Nombre interno').fill(name+' revisada');await page.getByLabel('Correcta 1',{exact:true}).uncheck();await page.getByLabel('Correcta 2',{exact:true}).check();
 // Archiving must not silently discard an edited question.
 await page.getByRole('button',{name:'Archivar pregunta',exact:true}).click();await expect(page.locator('question-archive [data-status]')).toContainText('Guarda primero');
 await page.getByRole('button',{name:'Guardar nueva versión',exact:true}).click();await expect(page.locator('question-form')).toHaveAttribute('data-version','2');
 await page.getByRole('link',{name:`Versión 1 · ${name} · Activa`,exact:true}).click();await expect(page.locator('question-form')).toHaveCount(0);await page.getByLabel('Pájaro',{exact:true}).check();await score(100);
 await page.screenshot({path:'../docs/screenshots/question-history-tablet.png',fullPage:true,animations:'disabled'});
 await page.getByRole('link',{name:'Abrir versión actual',exact:true}).click();await page.getByRole('button',{name:'Archivar pregunta',exact:true}).click();await expect(page.getByRole('button',{name:'Restaurar pregunta',exact:true})).toBeVisible();await expect(page.locator('question-form')).toHaveCount(0);
 await page.goto(bank+'?archived=true');await expect(page.getByRole('link',{name:`Abrir pregunta: ${name} revisada`,exact:true})).toBeVisible();await page.goto(questionURL);await page.getByRole('button',{name:'Restaurar pregunta',exact:true}).click();await expect(page.locator('question-form')).toHaveAttribute('data-version','4');
 await begin('multiple_choice',`Múltiple ${Date.now()}`);for(const [i,value] of ['Rojo','Piedra','Azul'].entries())await page.getByLabel(`Opción ${i+1}`,{exact:true}).fill(value);await page.getByLabel('Correcta 1',{exact:true}).check();await page.getByLabel('Correcta 3',{exact:true}).check();await create();
 await page.getByLabel('Rojo',{exact:true}).check();await score(0);await page.getByLabel('Azul',{exact:true}).check();await score(100);await page.getByLabel('Piedra',{exact:true}).check();await score(0);
 await begin('true_false',`Verdadero falso ${Date.now()}`);await page.getByLabel('Respuesta correcta',{exact:true}).selectOption('1');await create();await page.getByLabel('Falso',{exact:true}).check();await score(100);
 await begin('short_answer',`Respuesta corta ${Date.now()}`);await page.getByLabel('Una respuesta por línea (hasta 10)').fill('árbol\narboleda');await create();await page.getByLabel('Tu respuesta de prueba').fill(' ÁRBOL ');await score(100);await page.getByLabel('Tu respuesta de prueba').fill('arbol');await score(0);
 await page.screenshot({path:'../docs/screenshots/question-editor-tablet.png',fullPage:true,animations:'disabled'});
 expect(await page.evaluate(()=>document.documentElement.scrollWidth<=innerWidth)).toBe(true);
});
