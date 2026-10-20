import type { ArticleBlob, ArticleHost, ArticleRecord } from "./types";

export type ContentOffload = { kind: "markpub-markdown" | "leaflet-pages" | "pckt-items"; mimeType: "text/markdown" | "application/json"; body: string };
export type PreparedContent = { content: ArticleRecord; offload?: ContentOffload };
const bytes = (value: string) => new TextEncoder().encode(value).length;

/** Pure upload plan. The caller uploads the exact body and supplies the resulting blob. */
export function prepareContentOffload(host: ArticleHost, content: ArticleRecord): PreparedContent {
  if (host === "markpub" || host === "unknown") {
    const text = content.text as ArticleRecord | undefined;
    if (!text || typeof text.markdown !== "string") throw new Error("Invalid Markpub Markdown content.");
    if (bytes(text.markdown) > 100 * 1024) return { content, offload: { kind: "markpub-markdown", mimeType: "text/markdown", body: text.markdown } };
  } else if (host === "leaflet") {
    if (!Array.isArray(content.pages)) throw new Error("Invalid Leaflet pages.");
    const body = JSON.stringify(content.pages);
    if (bytes(body) > 100 * 1024) return { content, offload: { kind: "leaflet-pages", mimeType: "application/json", body } };
  } else if (host === "pckt") {
    if (!Array.isArray(content.items)) throw new Error("Invalid pckt items.");
    const body = JSON.stringify(content.items);
    if (bytes(body) > 20_000) return { content, offload: { kind: "pckt-items", mimeType: "application/json", body } };
  } else if (bytes(JSON.stringify(content)) >= 900_000) throw new Error("Offprint article content exceeds its record size limit.");
  return { content };
}

export function replaceContentOffload(prepared: PreparedContent, blob: ArticleBlob): ArticleRecord {
  const { content, offload } = prepared;
  if (!offload) return content;
  if (blob.$type !== "blob" || !blob.ref?.$link || blob.mimeType !== offload.mimeType || blob.size !== bytes(offload.body)) throw new Error("The content blob does not match the offload plan.");
  if (offload.kind === "markpub-markdown") return { ...content, text: { ...content.text as ArticleRecord, textBlob: blob, markdown: offload.body.slice(0, 1000) } };
  if (offload.kind === "leaflet-pages") return { ...content, blobPages: blob, pages: [], blobs: [] };
  const result = { ...content, blob, references: [] };
  delete (result as ArticleRecord).items;
  return result;
}
