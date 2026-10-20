export type ArticleHost = "leaflet" | "markpub" | "offprint" | "pckt" | "unknown";
export type ArticleRecord = Record<string, unknown>;
export type ArticleBlob = { $type: "blob"; ref: { $link: string }; mimeType: string; size: number };
export type ArticleImageAsset = { id: string; blob: ArticleBlob; alt: string; width: number; height: number; url: string };

/** Structural boundary: the consumer supplies its Markdown parser; no editor runtime is imported. */
export type MarkdownBlock =
  | { kind: "heading"; source: string; headingLevel: number }
  | { kind: "code"; source: string; language?: string }
  | { kind: "image"; source: string; url: string; alt: string }
  | { kind: "embed"; source: string; url: string }
  | { kind: "unordered-list" | "ordered-list"; source: string; listLevel: number; listStart?: number }
  | { kind: "paragraph" | "quote" | "thematic-break" | "empty"; source: string };
export interface MarkdownModel {
  parseMarkdownBlocks(markdown: string): readonly MarkdownBlock[];
  markdownHeadingText(source: string): string;
  markdownCodeText(source: string): string;
}
