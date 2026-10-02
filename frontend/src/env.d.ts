/// <reference types="astro/client" />
declare namespace App {
  interface Locals {
    user?: import("./lib/api").SessionUser;
    apiError?: string;
  }
}
declare module "h5p-standalone" {
  export class H5P {
    constructor(
      element: HTMLElement,
      options: {
        h5pJsonPath: string;
        frameJs: string;
        frameCss: string;
        frame?: boolean;
      },
    );
    then(resolve: () => void, reject?: (error: unknown) => void): Promise<void>;
  }
}
