import type { ArticleBlob, ArticleHost, ArticleImageAsset, ArticleRecord, MarkdownBlock } from "./types";

export const PREPARE_NSID = "at.anypub.prepare";
export interface PreparationOptions { signal?: AbortSignal; fetcher?: typeof fetch; endpoint?: string }
export interface DocumentRecordInput {
  site: string; title: string; publishedAt: string; updatedAt?: string;
  path?: string; tags?: string[]; langs?: string[]; content?: ArticleRecord;
  textContent?: string; description?: string; coverImage?: ArticleRecord;
}
export interface PublicationRecordInput {
  url: string; name: string; icon?: ArticleRecord; description?: string;
  basicTheme?: unknown; labels?: unknown; preferences?: unknown;
}
export type ContentOffload = { kind: "markpub-markdown" | "leaflet-pages" | "pckt-items"; mimeType: "text/markdown" | "application/json"; body: string };
export type PreparedContent = { content: ArticleRecord; offload?: ContentOffload };
export type NativeContent = { content: ArticleRecord; textContent: string };
export type NativeContentInput = { host: ArticleHost; markdown: string; description?: string; assets: ArticleImageAsset[]; blocks: readonly MarkdownBlock[] };
export type DocumentStrongReference = { uri: string; cid: string };

async function prepare<T>(body: ArticleRecord, key: string, options: PreparationOptions = {}): Promise<T> {
  const response = await (options.fetcher ?? fetch)(options.endpoint ?? `/xrpc/${PREPARE_NSID}`, {
    method: "POST", headers: { "Content-Type": "application/json" },
    credentials: "omit", redirect: "error", cache: "no-store",
    signal: options.signal ?? AbortSignal.timeout(30_000), body: JSON.stringify(body),
  });
  const result: unknown = await response.json();
  if (!response.ok) {
    const message = result && typeof result === "object" && "message" in result && typeof result.message === "string"
      ? result.message : `Publishing preparation failed (${response.status}).`;
    throw new Error(message);
  }
  if (!result || typeof result !== "object" || Array.isArray(result) || !(key in result) || !result[key as keyof typeof result] || typeof result[key as keyof typeof result] !== "object" || Array.isArray(result[key as keyof typeof result])) {
    throw new Error("The publishing service returned an invalid preparation result.");
  }
  return result[key as keyof typeof result] as T;
}

function recordInput(input: object, previous?: ArticleRecord): ArticleRecord {
  // JSON omits undefined. Preserve the editor's explicit clearing semantics on the wire.
  const clearFields = Object.keys(input).filter(key => (input as ArticleRecord)[key] === undefined);
  return { input, previous, clearFields };
}
export function buildDocumentRecord<T extends DocumentRecordInput>(input: T, previous?: ArticleRecord, options?: PreparationOptions): Promise<ArticleRecord> {
  return prepare({ operation: "document", ...recordInput(input, previous) }, "record", options);
}
export function buildPublicationRecord<T extends PublicationRecordInput>(input: T, previous?: ArticleRecord, options?: PreparationOptions): Promise<ArticleRecord> {
  return prepare({ operation: "publication", ...recordInput(input, previous) }, "record", options);
}
export function buildArticleNativeContent(input: NativeContentInput, options?: PreparationOptions): Promise<NativeContent> {
  return prepare({ operation: "article", native: input }, "native", options);
}
export function prepareContentOffload(host: ArticleHost, content: ArticleRecord, options?: PreparationOptions): Promise<PreparedContent> {
  return prepare({ operation: "offload", host, content }, "prepared", options);
}
export function replaceContentOffload(prepared: PreparedContent, blob: ArticleBlob, options?: PreparationOptions): Promise<ArticleRecord> {
  return prepare({ operation: "replaceOffload", prepared, blob }, "content", options);
}
export function appendLiveArticleContent(host: ArticleHost, content: ArticleRecord | undefined, timestamp: string, text: string, options?: PreparationOptions): Promise<ArticleRecord> {
  return prepare({ operation: "live", host, content, timestamp, text }, "content", options);
}
export function buildNativeDocumentWrapper(host: "offprint" | "pckt", reference: DocumentStrongReference, site?: string, previous?: ArticleRecord, options?: PreparationOptions): Promise<ArticleRecord> {
  return prepare({ operation: "wrapper", host, reference, site, previous }, "record", options);
}
