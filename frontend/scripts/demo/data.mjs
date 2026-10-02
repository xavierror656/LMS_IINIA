export const demoPassword = "aulaquest-demo";
export const users = [
  { id: 1, username: "luna", alias: "Luna", role: "student" },
  { id: 2, username: "sol", alias: "Sol", role: "student" },
  { id: 3, username: "profe", alias: "Profe Alex", role: "teacher" },
  { id: 4, username: "admin", alias: "Admin AulaQuest", role: "admin" },
];
export const courses = [
  {
    id: 1,
    slug: "creative-coding",
    title: "Exploradores del código",
    description:
      "Tu primera aventura entre ideas, instrucciones y pequeños descubrimientos.",
    icon: "rocket",
  },
  {
    id: 2,
    slug: "reading-worlds",
    title: "Historias que despiertan",
    description: "Lee, imagina y descubre mundos con tus propias palabras.",
    icon: "sprout",
  },
];
export const modules = [
  { id: 1, courseId: 1, title: "Primeros descubrimientos", position: 1 },
  {
    id: 2,
    courseId: 1,
    title: "Ideas que se convierten en pasos",
    position: 2,
  },
  { id: 3, courseId: 2, title: "La biblioteca del bosque", position: 1 },
];
export const lessons = [
  {
    id: 1,
    moduleId: 1,
    position: 1,
    type: "reading",
    title: "Cada aventura empieza con una idea",
    description: "Descubre qué es un algoritmo con una pequeña historia.",
    config: {
      body: "Luna quería llegar a la biblioteca del bosque. Primero miró el mapa, después eligió un camino y finalmente dio el primer paso.\n\nUn plan es una lista de pasos: también lo llamamos algoritmo. Piensa en algo que haces cada mañana. ¿Qué paso va primero? ¿Qué pasaría si cambias el orden?\n\nTu reto: explica con tres pasos cómo preparar tu mochila. ¡Ya estás pensando como una persona que programa!",
    },
  },
  {
    id: 2,
    moduleId: 1,
    position: 2,
    type: "code",
    title: "Tu laboratorio de ideas",
    description: "Escribe instrucciones y conoce una consola simulada.",
    config: {
      language: "javascript",
      starter: "// Escribe tu primera idea\nconsole.log('¡Hola, aventura!');",
    },
  },
  {
    id: 3,
    moduleId: 1,
    position: 3,
    type: "h5p",
    title: "Un reto para explorar",
    description: "Una actividad que tu docente puede configurar.",
    config: { activity: "demo" },
  },
  {
    id: 4,
    moduleId: 2,
    position: 1,
    type: "reading",
    title: "Una receta para un robot",
    description: "Aprende a dar instrucciones claras, paso a paso.",
    config: {
      body: "Nuestro robot quiere plantar una semilla, pero necesita ayuda.\n\n1. Llena una maceta con tierra.\n2. Haz un pequeño hueco.\n3. Coloca la semilla y cúbrela.\n4. Añade un poco de agua.\n\nEl orden importa. Si ponemos agua antes de tener una maceta, ¡acabará en el suelo!\n\nTu reto: ¿qué instrucción añadirías para que la planta reciba luz?",
    },
  },
  {
    id: 5,
    moduleId: 2,
    position: 2,
    type: "code",
    title: "Saluda con Python",
    description: "Explora otro lenguaje en el laboratorio de demostración.",
    config: {
      language: "python",
      starter:
        "# Este laboratorio no ejecuta código real\nnombre = 'Luna'\nprint('Hola, ' + nombre)",
    },
  },
  {
    id: 6,
    moduleId: 3,
    position: 1,
    type: "reading",
    title: "La semilla de las preguntas",
    description: "Una historia sobre la curiosidad y la paciencia.",
    config: {
      body: "Sol encontró una semilla azul junto al río. Nadie sabía qué planta crecería, así que decidió cuidarla.\n\nCada día observaba la tierra y escribía una pregunta: ¿necesitará más agua?, ¿le gustará el sol?, ¿cuándo aparecerá la primera hoja?\n\nUna mañana nació un brote. Sol descubrió que aprender se parece a cultivar: hacemos preguntas, probamos y esperamos con paciencia.\n\n¿Qué pregunta te gustaría plantar hoy?",
    },
  },
  {
    id: 7,
    moduleId: 3,
    position: 2,
    type: "reading",
    title: "El mapa que dibujamos juntos",
    description: "Descubre por qué compartir ideas nos ayuda a aprender.",
    config: {
      body: "Luna dibujó el río. Sol añadió los árboles. Alex recordó dónde estaba el puente.\n\nNinguno conocía todo el bosque, pero entre los tres hicieron un mapa lleno de caminos. Cuando apareció un sendero desconocido, lo marcaron con una pregunta.\n\nNo saber algo todavía es una invitación a explorar.\n\nTu reto: dibuja un lugar que conozcas y pide a alguien que añada un detalle nuevo.",
    },
  },
];
export const publicUser = ({ id, alias, role }) => ({ id, alias, role });
export const initialState = () => ({
  version: 1,
  completed: { 1: [], 2: [1, 6] },
  started: { 1: [], 2: [2] },
  attempts: [],
});
