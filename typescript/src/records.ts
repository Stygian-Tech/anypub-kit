/** Callers own timestamps, blob uploads, and persistence. No clock is read here. */
export interface DocumentRecordInput {
  site: string;
  title: string;
  publishedAt: string;
  updatedAt?: string | undefined;
  path?: string | undefined;
  tags?: string[] | undefined;
  langs?: string[] | undefined;
  content?: Record<string, unknown> | undefined;
  textContent?: string | undefined;
  description?: string | undefined;
  coverImage?: Record<string, unknown> | undefined;
}

export interface PublicationRecordInput {
  url: string;
  name: string;
  icon?: Record<string, unknown> | undefined;
  description?: string | undefined;
  basicTheme?: unknown;
  labels?: unknown;
  preferences?: unknown;
}

export interface BuiltDocumentRecord extends DocumentRecordInput {
  $type: 'site.standard.document';
  [key: string]: unknown;
}

export interface BuiltPublicationRecord extends PublicationRecordInput {
  $type: 'site.standard.publication';
  [key: string]: unknown;
}

const encoder = new TextEncoder();
const segmenter = new Intl.Segmenter('en', { granularity: 'grapheme' });
const clean = (value: string): boolean => !/[\s\p{Cc}\p{Cf}]/u.test(value);

function invalid(field: string, reason: string): never {
  throw new TypeError(`${field}: ${reason}`);
}

function text(value: unknown, field: string, bytes?: number, graphemes?: number): asserts value is string {
  if (typeof value !== 'string') invalid(field, 'Expected a string');
  if (bytes !== undefined && encoder.encode(value).length > bytes) invalid(field, `Exceeds ${bytes} UTF-8 bytes`);
  if (graphemes !== undefined) {
    let count = 0;
    for (const _ of segmenter.segment(value)) {
      if (++count > graphemes) invalid(field, `Exceeds ${graphemes} graphemes`);
    }
  }
}

function validHTTPS(value: string): boolean {
  if (!clean(value) || !/^https:\/\//i.test(value) || /[?#\\]/.test(value)) return false;
  try {
    const url = new URL(value);
    return url.protocol === 'https:' && !!url.hostname && !url.username && !url.password && !value.slice(8).split('/')[0]?.includes('@');
  } catch {
    return false;
  }
}

function validPublicationURI(value: string): boolean {
  if (!clean(value) || /[%?#@]/.test(value)) return false;
  const match = /^at:\/\/([^/]+)\/site\.standard\.publication\/([A-Za-z0-9._~:-]+)$/.exec(value);
  if (!match) return false;
  const authority = match[1]!;
  const key = match[2]!;
  const validAuthority = authority.startsWith('did:')
    ? /^did:[a-z]+:[A-Za-z0-9._:%-]+$/.test(authority)
    : /^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?\.[A-Za-z]{2,}$/.test(authority);
  return validAuthority && key !== '.' && key !== '..' && key.length <= 512;
}

function validPath(value: string): boolean {
  if (!value || !clean(value) || value.startsWith('//')) return false;
  try {
    const decoded = decodeURIComponent(value);
    return clean(decoded) && !decoded.startsWith('//') && !/[\\?#:%]/.test(decoded)
      && !decoded.split('/').some(part => part === '.' || part === '..');
  } catch {
    return false;
  }
}

function validType(value: unknown): boolean {
  if (typeof value !== 'string') return false;
  const pieces = value.split('#');
  const nsid = pieces[0]!;
  const parts = nsid.split('.');
  if (pieces.length > 2 || parts.length < 3 || encoder.encode(nsid).length > 317) return false;
  return parts.every((part, index) => encoder.encode(part).length <= 63 && (
    index === parts.length - 1 ? /^[A-Za-z][A-Za-z0-9]*$/ : /^[A-Za-z]([A-Za-z0-9-]*[A-Za-z0-9])?$/
  ).test(part)) && (pieces.length === 1 || /^[A-Za-z][A-Za-z0-9_]*$/.test(pieces[1]!));
}

function timestamp(value: unknown, field: string): void {
  text(value, field);
  // Validate local calendar components before parsing: Date.parse otherwise
  // silently rolls dates such as February 30 into March. Preserve RFC3339 offsets.
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.\d{1,9})?(Z|[+-]\d{2}:\d{2})$/.exec(value);
  if (!match) invalid(field, 'Expected an ISO timestamp with a timezone');
  const [year, month, day, hour, minute, second] = match.slice(1, 7).map(Number);
  const leapYear = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const days = [31, leapYear ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  const zone = match[7]!;
  if (month < 1 || month > 12 || day < 1 || day > days[month - 1]! || hour > 23 || minute > 59 || second > 59
    || (zone !== 'Z' && (Number(zone.slice(1, 3)) > 23 || Number(zone.slice(4)) > 59)) || !Number.isFinite(Date.parse(value))) {
    invalid(field, 'Invalid calendar date or time');
  }
}

/** Omission retains a previous field; an own property with undefined removes it. */
function merge(input: object, previous: Record<string, unknown> | undefined, type: string): Record<string, unknown> {
  const result: Record<string, unknown> = { ...previous, ...input, $type: type };
  for (const key of Object.keys(result)) {
    if (result[key] === undefined) delete result[key];
  }
  return result;
}

export function buildDocumentRecord<T extends DocumentRecordInput>(input: T, previous?: Record<string, unknown>): BuiltDocumentRecord {
  const record = merge(input, previous, 'site.standard.document');
  text(record.site, 'site');
  if (!validHTTPS(record.site) && !validPublicationURI(record.site)) invalid('site', 'Expected an AT publication URI or HTTPS URL');
  text(record.title, 'title', 5_000, 500);
  timestamp(record.publishedAt, 'publishedAt');
  if ('updatedAt' in record) timestamp(record.updatedAt, 'updatedAt');
  if ('description' in record) text(record.description, 'description', 30_000, 3_000);
  if ('textContent' in record) text(record.textContent, 'textContent');
  if ('path' in record) {
    text(record.path, 'path');
    if (!validPath(record.path)) invalid('path', 'Expected a relative path without traversal, query, fragment, or authority');
  }
  for (const field of ['tags', 'langs'] as const) {
    if (!(field in record)) continue;
    const values = record[field];
    if (!Array.isArray(values)) invalid(field, 'Expected an array of strings');
    for (const value of values) text(value, field, field === 'tags' ? 1_280 : undefined, field === 'tags' ? 128 : undefined);
  }
  if ('content' in record) {
    const content = record.content;
    if (!content || typeof content !== 'object' || Array.isArray(content) || !validType((content as Record<string, unknown>).$type)) {
      invalid('content', 'Expected an object with a valid Lexicon $type');
    }
  }
  return record as BuiltDocumentRecord;
}

export function buildPublicationRecord<T extends PublicationRecordInput>(input: T, previous?: Record<string, unknown>): BuiltPublicationRecord {
  const record = merge(input, previous, 'site.standard.publication');
  text(record.url, 'url');
  if (!validHTTPS(record.url)) invalid('url', 'Expected an HTTPS publication URL');
  record.url = record.url.replace(/\/+$/, '');
  text(record.name, 'name', 5_000, 500);
  if ('description' in record) text(record.description, 'description', 30_000, 3_000);
  return record as BuiltPublicationRecord;
}

export interface DocumentStrongReference {
  uri: string;
  cid: string;
}

export interface NativeDocumentWrapper extends Record<string, unknown> {
  $type: 'app.offprint.document.article' | 'blog.pckt.document';
  document: DocumentStrongReference & { $type?: string; [key: string]: unknown };
  site?: string;
}

export function buildNativeDocumentWrapper(
  host: 'offprint' | 'pckt',
  reference: DocumentStrongReference,
  site?: string,
  previous?: Record<string, unknown>,
): NativeDocumentWrapper {
  if (host !== 'offprint' && host !== 'pckt') invalid('host', 'Expected offprint or pckt');
  text(reference.uri, 'document.uri');
  text(reference.cid, 'document.cid');
  if (!reference.uri || !reference.cid) invalid('document', 'Expected a URI and CID');
  const priorDocument = previous?.document;
  const document = {
    ...(priorDocument && typeof priorDocument === 'object' && !Array.isArray(priorDocument) ? priorDocument : {}),
    ...reference,
  };
  const record = merge({ document }, previous, host === 'offprint' ? 'app.offprint.document.article' : 'blog.pckt.document');
  if (host === 'offprint') {
    Object.assign(document, { $type: 'com.atproto.repo.strongRef' });
  } else {
    delete (document as Record<string, unknown>).$type;
    if (site !== undefined) record.site = site;
    text(record.site, 'site');
    if (!record.site || !clean(record.site)) invalid('site', 'Pckt requires a site');
  }
  return record as NativeDocumentWrapper;
}
