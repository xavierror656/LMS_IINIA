import { test, expect, type Page } from "@playwright/test";
import { readFile } from "node:fs/promises";

test("FL5 adjuntos privados, descarga, borrado y entrega solo con archivo", async ({ page, browser }) => {
  test.skip(process.env.E2E_ACADEMIC !== "1", "Requiere Go/PostgreSQL sintético");
  test.setTimeout(120_000);
  const origin=process.env.E2E_BASE_URL??"http://localhost:4321";
  async function login(p:Page,user:string){
    await p.goto("/login");await p.getByLabel("Tu usuario").fill(user);
    await p.getByLabel("Tu contraseña").fill(user==="profe"?process.env.E2E_TEACHER_PASSWORD!:process.env.E2E_STUDENT_PASSWORD!);
    await p.getByRole("button",{name:"Entrar a mi aventura"}).click();await expect(p).toHaveURL(user==="profe"?/\/teacher$/:/\/courses$/);
  }
  await login(page,"profe");
  const course=(await (await page.request.get("/api/v1/teacher/courses")).json()).items[0];
  const module=(await (await page.request.get(`/api/v1/teacher/courses/${course.id}/activities`)).json()).modules[0];
  const created=await page.request.post(`/api/v1/teacher/courses/${course.id}/activities`,{headers:{Origin:origin},data:{moduleId:module.id,type:"assignment",title:`Mis archivos ${Date.now()}`,description:"Comparte tu trabajo",body:"Puedes entregar tu trabajo como texto o archivo."}});
  expect(created.status()).toBe(201);let activity=await created.json();
  // FI1: la guía se sube antes de publicar; el PDF se analiza de verdad.
  const guide=Buffer.from("%PDF-1.4\n1 0 obj\n<< /Type /Catalog /Pages 2 0 R >>\nendobj\n2 0 obj\n<< /Type /Pages /Kids [3 0 R] /Count 1 >>\nendobj\n3 0 obj\n<< /Type /Page /Parent 2 0 R /MediaBox [0 0 200 200] >>\nendobj\nxref\n0 4\n0000000000 65535 f \n0000000009 00000 n \n0000000058 00000 n \n0000000115 00000 n \ntrailer\n<< /Size 4 /Root 1 0 R >>\nstartxref\n190\n%%EOF\n");
  await page.goto(`/teacher/activities/${activity.id}`);
  await page.getByLabel("Añadir un archivo (texto, imagen o PDF; hasta 2 MiB)").setInputFiles({name:"guia.pdf",mimeType:"application/pdf",buffer:guide});
  await page.getByRole("button",{name:"Subir archivo"}).click();
  await expect(page.getByText("guia.pdf")).toBeVisible();
  const published=await page.request.post(`/api/v1/teacher/activities/${activity.id}/publish`,{headers:{Origin:origin},data:{version:1}});expect(published.ok()).toBeTruthy();activity=await published.json();
  const context=await browser.newContext({viewport:{width:768,height:1024},reducedMotion:"reduce"});const student=await context.newPage();
  try{
    await login(student,"luna");await student.goto(`/lessons/${activity.lessonId}`);
    await student.getByRole("link",{name:"Gestionar adjuntos"}).click();
    const input=student.getByLabel("Archivo para adjuntar");
    await input.setInputFiles({name:"fake.png",mimeType:"image/png",buffer:Buffer.from("not an image")});
    await student.getByRole("button",{name:"Adjuntar archivo",exact:true}).click();
    await expect(student.locator("[data-file-status]")).toContainText("Revisa");
    expect(await input.evaluate((el:HTMLInputElement)=>el.files?.[0]?.name)).toBe("fake.png");
    await input.setInputFiles({name:"mi-idea.txt",mimeType:"text/plain",buffer:Buffer.from("Mi idea de prueba con acentos: árbol")});
    await student.getByRole("button",{name:"Adjuntar archivo",exact:true}).click();
    const link=student.getByRole("link",{name:"Descargar mi-idea.txt",exact:true});await expect(link).toBeVisible();
    const path=await link.getAttribute("href");
    expect((await page.request.get(path!)).status()).toBe(404);
    const waiting=student.waitForEvent("download");await link.click();const download=await waiting;
    expect(download.suggestedFilename()).toBe("mi-idea.txt");expect((await readFile((await download.path())!)).toString()).toBe("Mi idea de prueba con acentos: árbol");
    await student.reload();await expect(link).toBeVisible();
    student.once("dialog",dialog=>dialog.accept());await student.getByRole("button",{name:"Quitar mi-idea.txt",exact:true}).click();
    await expect(link).toHaveCount(0);
    expect((await student.request.get(path!)).status()).toBe(404);
    await input.setInputFiles({name:"entrega.txt",mimeType:"text/plain",buffer:Buffer.from("Esta es mi entrega sin texto adicional.")});
    await student.getByRole("button",{name:"Adjuntar archivo",exact:true}).click();
    await expect(student.getByRole("link",{name:"Descargar entrega.txt",exact:true})).toBeVisible();
    await expect(student.getByLabel("Tu progreso")).toContainText("Luna");
    await student.screenshot({path:"../docs/screenshots/attachments-student-tablet.png",fullPage:true,animations:"disabled"});
    await student.getByRole("link",{name:"Volver a mi tarea"}).click();
    await expect(student.getByLabel("Escribe tu trabajo")).toHaveValue("");
    await student.getByRole("button",{name:"Enviar al docente"}).click();
    await expect(student.getByRole("heading",{name:"Tu trabajo está enviado"})).toBeVisible();
    const finalLink=student.getByRole("link",{name:"Descargar entrega.txt",exact:true});const finalPath=await finalLink.getAttribute("href");
    await student.goto(`/lessons/${activity.lessonId}/files`);await expect(student.getByLabel("Archivo para adjuntar")).toHaveCount(0);
    await page.goto(`/teacher/activities/${activity.id}`);
    await expect(page.getByRole("link",{name:"Descargar entrega.txt",exact:true})).toBeVisible();
    const allowed=await page.request.get(finalPath!);expect(allowed.status()).toBe(200);expect(await allowed.text()).toBe("Esta es mi entrega sin texto adicional.");
    // FI2/FI3: el alumno ve la guía congelada y quitarla del borrador no la rompe.
    await student.goto(`/lessons/${activity.lessonId}`);
    const guideLink=student.getByRole("link",{name:"Descargar guia.pdf",exact:true});
    await expect(guideLink).toBeVisible();
    const guideHref=await guideLink.getAttribute("href");
    const guideDownload=await student.request.get(guideHref!);
    expect(guideDownload.status()).toBe(200);
    expect(guideDownload.headers()["content-type"]).toBe("application/octet-stream");
    await page.goto(`/teacher/activities/${activity.id}`);
    page.once("dialog",(dialog)=>dialog.accept());
    await page.getByRole("button",{name:"Quitar",exact:true}).click();
    await expect(page.getByText("guia.pdf")).toHaveCount(0);
    await student.reload();
    await expect(student.getByRole("link",{name:"Descargar guia.pdf",exact:true})).toBeVisible();
    await page.screenshot({path:"../docs/screenshots/attachments-teacher-tablet.png",fullPage:true,animations:"disabled"});
  }finally{await context.close();}
});
