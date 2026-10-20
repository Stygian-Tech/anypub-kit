import { describe, expect, test } from "bun:test";
import { articleRichText, buildArticleNativeContent } from "../src/content";
import { appendLiveArticleContent, livePostText } from "../src/live-content";
import { prepareContentOffload, replaceContentOffload } from "../src/offload";
import type { ArticleBlob, ArticleHost, ArticleImageAsset, ArticleRecord, MarkdownBlock, MarkdownModel } from "../src/types";
const blob: ArticleBlob = { $type: "blob", ref: { $link: "bafkreitest" }, mimeType: "image/png", size: 3 };
const asset: ArticleImageAsset = { id: "image", blob, alt: "Photo", width: 640, height: 480, url: "https://pds.test/blob" };
const model = (blocks: MarkdownBlock[]): MarkdownModel => ({ parseMarkdownBlocks: () => blocks, markdownHeadingText: source => source.replace(/^#+ /, ""), markdownCodeText: source => source });
const build = (host: ArticleHost, blocks: MarkdownBlock[], description = "", markdown = "source", assets = [asset]) => buildArticleNativeContent(host, markdown, assets, description, model(blocks));

test("rich text counts UTF-8 byte ranges and suppresses unsafe link facets", () => {
  expect(articleRichText("😀 **é** [bad](javascript:alert)", "pub.leaflet.richtext.facet")).toEqual({ plaintext: "😀 é bad", facets: [{ index: { byteStart: 5, byteEnd: 7 }, features: [{ $type: "pub.leaflet.richtext.facet#bold" }] }] });
});
test("native providers retain headings, nested checked lists, code, quotes, media and summary semantics", () => {
  const blocks: MarkdownBlock[] = [
    { kind: "heading", source: "###### Title", headingLevel: 6 },
    { kind: "unordered-list", source: "- [x] Done", listLevel: 0, listStart: 1 },
    { kind: "ordered-list", source: "  3. Child", listLevel: 1, listStart: 3 },
    { kind: "code", source: "literal **code**", language: "TS" },
    { kind: "quote", source: "> One\n> Two" },
    { kind: "image", source: "", url: "article-asset://image", alt: "Caption" },
    { kind: "embed", source: "", url: "https://example.com" },
    { kind: "thematic-break", source: "---" },
  ];
  const leaflet = build("leaflet", blocks);
  const pages = leaflet.content.pages as { blocks: { block: ArticleRecord }[] }[];
  expect(pages[0].blocks[0].block).toMatchObject({ $type: "pub.leaflet.blocks.header", level: 6 });
  expect(pages[0].blocks[1].block).toMatchObject({ children: [{ checked: true, orderedListChildren: { startIndex: 3 } }] });
  expect(pages[0].blocks[2].block).toMatchObject({ plaintext: "literal **code**", language: "ts" });
  expect(pages[0].blocks[4].block).toMatchObject({ image: blob, aspectRatio: { width: 640, height: 480 } });
  const offprint = build("offprint", blocks).content.items as ArticleRecord[];
  expect(offprint[0]).toMatchObject({ $type: "app.offprint.block.heading", level: 3 });
  expect(offprint[1]).toMatchObject({ $type: "app.offprint.block.taskList", children: [{ checked: true }] });
  expect(offprint[3]).toMatchObject({ content: [{ plaintext: "One" }, { plaintext: "Two" }] });
  const pckt = build("pckt", blocks, "Summary").content.items as ArticleRecord[];
  expect(pckt[0]).toMatchObject({ level: 3, plaintext: "Summary" });
  expect(pckt[5]).toMatchObject({ attrs: { src: "blob:bafkreitest", blob, align: "center", alt: "Caption" } });
  expect(pckt.at(-1)).toMatchObject({ $type: "blog.pckt.block.text", plaintext: "" });
  expect(leaflet.textContent).toContain("literal **code**\n\nOne\nTwo\n\nCaption");
});
test("Markpub resolves assets and embeds without importing the consumer parser", () => {
  const result = build("markpub", [{ kind: "image", source: "", url: "article-asset://image", alt: "x" }], "", "![x](article-asset://image)\n@[embed](https://example.com)");
  expect(result.content).toMatchObject({ $type: "at.markpub.markdown", flavor: "gfm", text: { $type: "at.markpub.text", markdown: "![x](https://pds.test/blob)\n[https://example.com](https://example.com)" } });
});
test("rejects missing assets, remote native images, unsafe URLs and malformed image dimensions", () => {
  const image = (url: string): MarkdownBlock => ({ kind: "image", source: "", url, alt: "x" });
  expect(() => build("leaflet", [image("article-asset://missing")])).toThrow("not been uploaded");
  expect(() => build("pckt", [image("https://example.com/x.png")])).toThrow("Upload remote");
  expect(() => build("markpub", [image("javascript:bad")])).toThrow("HTTP(S)");
  expect(() => build("offprint", [image("article-asset://image")], "", "", [{ ...asset, width: 0 }])).toThrow("Invalid article image");
});

describe("pure blob offload plans", () => {
  for (const [host, content, kind] of [
    ["markpub", { $type: "at.markpub.markdown", custom: true, text: { custom: true, markdown: "é".repeat(52000) } }, "markpub-markdown"],
    ["leaflet", { $type: "pub.leaflet.content", custom: true, pages: [{ blocks: ["é".repeat(52000)] }] }, "leaflet-pages"],
    ["pckt", { $type: "blog.pckt.content", custom: true, items: ["é".repeat(11000)] }, "pckt-items"],
  ] as const) test(`${host} uploads exact UTF-8 body and preserves extensions`, () => {
    const original = JSON.stringify(content);
    const plan = prepareContentOffload(host, content);
    expect(plan.offload?.kind).toBe(kind);
    const uploaded = { ...blob, mimeType: plan.offload!.mimeType, size: new TextEncoder().encode(plan.offload!.body).length };
    const replaced = replaceContentOffload(plan, uploaded);
    expect(replaced.custom).toBe(true);
    if (host === "markpub") expect(replaced.text).toMatchObject({ custom: true, textBlob: uploaded, markdown: "é".repeat(1000) });
    if (host === "leaflet") expect(replaced).toMatchObject({ pages: [], blobs: [], blobPages: uploaded });
    if (host === "pckt") { expect(replaced.items).toBeUndefined(); expect(replaced).toMatchObject({ blob: uploaded, references: [] }); }
    expect(JSON.stringify(content)).toBe(original);
    expect(() => replaceContentOffload(plan, { ...uploaded, size: 0 })).toThrow("does not match");
  });
  test("thresholds and Offprint record limit", () => {
    const content = { text: { markdown: "x".repeat(100 * 1024) } };
    expect(prepareContentOffload("markpub", content).offload).toBeUndefined();
    expect(() => prepareContentOffload("offprint", { items: ["x".repeat(900000)] })).toThrow("record size limit");
  });
});

test("live appends preserve extensions and existing native data without mutation", () => {
  const timestamp = "2026-10-20T12:00:00.000Z";
  for (const host of ["leaflet", "offprint", "pckt", "markpub", "unknown"] as const) {
    const initial = { ...appendLiveArticleContent(host, undefined, timestamp, "Old"), custom: { preserved: true } };
    const before = JSON.stringify(initial);
    const next = appendLiveArticleContent(host, initial, timestamp, "**Literal**");
    expect(next.custom).toEqual(initial.custom);
    expect(JSON.stringify(initial)).toBe(before);
    if (host === "markpub" || host === "unknown") expect((next.text as ArticleRecord).markdown).toContain("\\*\\*Literal\\*\\*");
    else expect(JSON.stringify(next)).toContain('"plaintext":"**Literal**"');
  }
  expect(livePostText(" Update ", ["#live", "two words"])).toBe("Update\n\n#live #twowords");
  expect(() => appendLiveArticleContent("markpub", { $type: "at.markpub.markdown", text: { markdown: "old", textBlob: blob } }, timestamp, "New")).toThrow("Blob-backed");
  expect(() => appendLiveArticleContent("leaflet", { $type: "pub.leaflet.content", pages: [], blobPages: blob }, timestamp, "New")).toThrow("Blob-backed");
  expect(() => appendLiveArticleContent("offprint", { $type: "other" }, timestamp, "New")).toThrow("format changed");
});
