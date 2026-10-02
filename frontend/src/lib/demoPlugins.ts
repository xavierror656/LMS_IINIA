/** Local demonstration contract; see contracts/demo-plugins.md. */
export interface DemoPlugin {
  id: "code" | "h5p";
  title: string;
  instructions: string;
  defaultLanguage?: "lesson" | "javascript" | "python";
}
