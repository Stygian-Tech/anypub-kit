import { expect, test } from "bun:test";
import { appendLiveArticleContent, buildArticleNativeContent, buildDocumentRecord, buildNativeDocumentWrapper, buildPublicationRecord, prepareContentOffload, replaceContentOffload } from "../src/client";

const record = { $type: "site.standard.document", title: "Prepared" };
const input = { site: "https://example.com", title: "Title", publishedAt: "2026-10-01T00:00:00Z" };
function fetcher(result: object, inspect?: (url: string | URL | Request, options?: RequestInit) => void) {
  return (async (url: string | URL | Request, options?: RequestInit) => { inspect?.(url, options); return Response.json(result); }) as unknown as typeof fetch;
}
test("serializes explicit optional clearing and preserves extensions without transporting credentials", async () => {
  const signal = new AbortController().signal;
  const previous = { description: "old", custom: { future: true } };
  const result = await buildDocumentRecord({ ...input, description: undefined }, previous, { signal, fetcher: fetcher({ record }, (url, options) => {
    expect(url).toBe("/xrpc/at.anypub.prepare");
    expect(options?.method).toBe("POST");
    expect(options?.credentials).toBe("omit");
    expect(options?.redirect).toBe("error");
    expect(options?.cache).toBe("no-store");
    expect(options?.headers).toEqual({ "Content-Type": "application/json" });
    expect(options?.signal).toBe(signal);
    expect(JSON.parse(options!.body as string)).toEqual({ operation: "document", input, previous, clearFields: ["description"] });
  }) });
  expect(result).toEqual(record);
});
test("every publishing operation calls the Go service with its structured inputs", async () => {
  const calls: Record<string, unknown>[] = [];
  const options = { endpoint: "https://api.example/xrpc/at.anypub.prepare", fetcher: fetcher({ record, native: { content: {}, textContent: "" }, prepared: { content: {} }, content: { prepared: true } }, (_, init) => calls.push(JSON.parse(init!.body as string))) };
  await buildPublicationRecord({ url: "https://example.com", name: "Publication" }, undefined, options);
  await buildArticleNativeContent({ host: "markpub", markdown: "Text", assets: [], blocks: [] }, options);
  const prepared = await prepareContentOffload("markpub", {}, options);
  await replaceContentOffload(prepared, { $type: "blob", ref: { $link: "cid" }, mimeType: "text/markdown", size: 1 }, options);
  await appendLiveArticleContent("markpub", undefined, "2026-10-01T00:00:00Z", "New", options);
  await buildNativeDocumentWrapper("offprint", { uri: "at://document", cid: "cid" }, undefined, { custom: true }, options);
  expect(calls.map(call => call.operation)).toEqual(["publication", "article", "offload", "replaceOffload", "live", "wrapper"]);
  expect(calls[1].native).toEqual({ host: "markpub", markdown: "Text", assets: [], blocks: [] });
  expect(calls[5].previous).toEqual({ custom: true });
});
test("propagates validation and transport errors without a local publishing fallback", async () => {
  const invalid = (async () => Response.json({ error: "InvalidRequest", message: "title: exceeds limit" }, { status: 400 })) as unknown as typeof fetch;
  await expect(buildDocumentRecord(input, undefined, { fetcher: invalid })).rejects.toThrow("title: exceeds limit");
  const unavailable = (async () => { throw new Error("offline"); }) as unknown as typeof fetch;
  await expect(buildDocumentRecord(input, undefined, { fetcher: unavailable })).rejects.toThrow("offline");
  const malformed = (async () => Response.json({ wrong: true })) as unknown as typeof fetch;
  await expect(buildDocumentRecord(input, undefined, { fetcher: malformed })).rejects.toThrow("invalid preparation result");
});
