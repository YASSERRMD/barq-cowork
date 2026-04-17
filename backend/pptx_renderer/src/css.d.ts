declare module "*.css" {
  const content: string;
  export default content;
}

declare module "*.woff2" {
  const content: string; // base64 data URL injected by esbuild dataurl loader
  export default content;
}

declare module "*.woff" {
  const content: string;
  export default content;
}
