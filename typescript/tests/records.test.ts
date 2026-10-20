import { describe, expect, test } from 'bun:test';
import fixtures from '../../Fixtures/standard-site.json';
import {
  buildDocumentRecord,
  buildNativeDocumentWrapper,
  buildPublicationRecord,
  type DocumentRecordInput,
  type PublicationRecordInput,
} from '../src/records';

const document: DocumentRecordInput = {
  site: 'at://did:plc:example/site.standard.publication/example',
  title: 'Example',
  publishedAt: '2026-10-01T12:30:00.000Z',
};

describe('fixture goldens', () => {
  for (const fixture of fixtures) {
    test(fixture.name, () => {
      const result: Record<string, unknown> = fixture.kind === 'publication'
        ? buildPublicationRecord(fixture.input as PublicationRecordInput)
        : buildDocumentRecord(fixture.input as DocumentRecordInput);
      expect(result).toEqual(fixture.expected);
    });
  }
});

test('preserves explicit timestamps without generating update times', () => {
  expect(buildDocumentRecord(document).publishedAt).toBe(document.publishedAt);
  expect(buildDocumentRecord(document)).not.toHaveProperty('updatedAt');
  for (const value of ['2024-02-29T23:59:59Z', '2026-10-01T12:30:00.123456789Z', '2026-10-01T12:30:00-05:00', '2026-10-01T12:30:00+00:00']) {
    expect(buildDocumentRecord({ ...document, publishedAt: value, updatedAt: value }).updatedAt).toBe(value);
  }
});

test('rejects invalid calendar dates and noncanonical timestamp syntax', () => {
  for (const date of [
    '2026-02-29T12:00:00.000Z', '2024-02-30T12:00:00.000Z',
    '2026-04-31T12:00:00Z', '2026-13-01T00:00:00Z',
    '2026-01-01T24:00:00Z', '2026-01-01T00:60:00Z', '2026-01-01T00:00:60Z',
    '2026-01-01', '2026-1-1T00:00:00Z', '2026-01-01T00:00:00+24:00', '2026-01-01T00:00:00+00:60', '2026-02-30T12:00:00-05:00',
    '2026-01-01T00:00:00z', 'garbage', '',
  ]) {
    expect(() => buildDocumentRecord({ ...document, publishedAt: date })).toThrow('publishedAt');
    expect(() => buildDocumentRecord({ ...document, updatedAt: date })).toThrow('updatedAt');
  }
});

test('grapheme limits count combining sequences and emoji as characters', () => {
  for (const character of ['e\u0301', '👨‍👩‍👧‍👦']) {
    // Family emoji reach the byte limit before the grapheme limit.
    const count = character === 'e\u0301' ? 500 : 200;
    expect(buildDocumentRecord({ ...document, title: character.repeat(count) }).title).toBe(character.repeat(count));
  }
  expect(() => buildDocumentRecord({ ...document, title: 'e\u0301'.repeat(501) })).toThrow('title');
  expect(() => buildDocumentRecord({ ...document, title: '👨‍👩‍👧‍👦'.repeat(201) })).toThrow('UTF-8');
  expect(() => buildPublicationRecord({ url: 'https://example.com', name: 'a'.repeat(501) })).toThrow('name');
  expect(buildPublicationRecord({ url: 'https://example.com', name: 'a'.repeat(500) }).name.length).toBe(500);
});

test('description and tags enforce both byte and grapheme boundaries', () => {
  expect(buildDocumentRecord({ ...document, description: 'é'.repeat(3000), tags: ['é'.repeat(128)] }).tags).toEqual(['é'.repeat(128)]);
  expect(() => buildDocumentRecord({ ...document, description: 'a'.repeat(3001) })).toThrow('description');
  expect(() => buildPublicationRecord({ url: 'https://example.com', name: 'n', description: 'a'.repeat(3001) })).toThrow('description');
  expect(() => buildDocumentRecord({ ...document, tags: ['a'.repeat(129)] })).toThrow('tags');
  const combining = 'a' + '\u0301'.repeat(640);
  expect(() => buildDocumentRecord({ ...document, tags: [combining] })).toThrow('UTF-8');
  expect(() => buildDocumentRecord({ ...document, description: '👨‍👩‍👧‍👦'.repeat(1201) })).toThrow('UTF-8');
  expect(() => buildPublicationRecord({ url: 'https://example.com', name: '👨‍👩‍👧‍👦'.repeat(201) })).toThrow('UTF-8');
});

test('document optionals retain, replace, and explicitly clear without mutating previous records', () => {
  const optional = {
    updatedAt: '2026-10-02T00:00:00.000Z', path: '/old', tags: ['old'], langs: ['en'],
    content: { $type: 'at.markpub.markdown', text: 'old' }, textContent: 'old',
    description: 'old', coverImage: { $type: 'blob', ref: { $link: 'cid' } },
  };
  const previous = Object.freeze({ ...document, ...optional, $type: 'wrong', custom: { nested: true } });
  const kept = buildDocumentRecord({ ...document, title: 'Edited' }, previous);
  expect(kept).toMatchObject({ ...optional, custom: { nested: true }, $type: 'site.standard.document' });
  const clear = Object.fromEntries(Object.keys(optional).map(key => [key, undefined]));
  const result = buildDocumentRecord({ ...document, ...clear }, previous);
  for (const field of Object.keys(optional)) expect(Object.hasOwn(result, field)).toBe(false);
  expect(previous.path).toBe('/old');
  expect(buildDocumentRecord({ ...document, tags: [], custom: 'new' }, previous)).toMatchObject({ tags: [], custom: 'new' });
  expect(buildDocumentRecord({ ...document, publishedAt: '2025-01-01T00:00:00Z' }, previous).publishedAt).toBe('2025-01-01T00:00:00Z');
});

test('publication optionals and extensions retain or clear, with authoritative record type', () => {
  const input = { url: 'https://example.com///', name: 'Publication' };
  const optionals = { icon: { $type: 'blob' }, description: 'desc', basicTheme: { color: 'red' }, labels: { values: [] }, preferences: { showInDiscover: true } };
  const previous = Object.freeze({ ...optionals, custom: 42, $type: 'wrong' });
  expect(buildPublicationRecord(input, previous)).toEqual({ ...previous, ...input, url: 'https://example.com', $type: 'site.standard.publication' });
  const clear = Object.fromEntries(Object.keys(optionals).map(key => [key, undefined]));
  const cleared = buildPublicationRecord({ ...input, ...clear }, previous);
  for (const field of Object.keys(optionals)) expect(Object.hasOwn(cleared, field)).toBe(false);
  expect(cleared.custom).toBe(42);
  expect(previous.description).toBe('desc');
});

test('validates HTTPS and publication AT URI sites', () => {
  for (const site of ['https://example.com', 'https://example.com/publication', 'at://writer.example/site.standard.publication/key', 'at://did:web:localhost%3A8443/site.standard.publication/key']) {
    expect(buildDocumentRecord({ ...document, site }).site).toBe(site);
  }
  for (const site of [
    'http://example.com', 'https://', 'https://u:p@example.com', 'https://@example.com',
    'https://example.com?', 'https://example.com#', 'https://example.com/a b',
    'at://did:plc:example/site.standard.document/key', 'at://did:plc:example/site.standard.publication/..',
    'at://did:plc:example/site.standard.publication/%61', 'at://did:web:localhost%zz/site.standard.publication/key', 'at://did:plc:example/site.standard.publication/' + 'a'.repeat(513),
  ]) expect(() => buildDocumentRecord({ ...document, site })).toThrow('site');
  for (const url of ['http://example.com', document.site, 'https://example.com?x=1']) {
    expect(() => buildPublicationRecord({ url, name: 'Name' })).toThrow('url');
  }
});

test('safe paths permit Unicode and encoded characters while rejecting traversal', () => {
  for (const path of ['/posts/café', 'posts/hello', '/posts/caf%C3%A9', '/']) {
    expect(buildDocumentRecord({ ...document, path }).path).toBe(path);
  }
  for (const path of ['', '//evil', '/..', '/./post', '/%2e%2E/post', '/%252e%252e', '/%2f%2fevil', '/%5Cevil', '/x?y', '/x%3Fy', '/x#y', '/x%23y', 'https://evil', '/%3a', '/%', '/%FF', '/a%20b', '/%00']) {
    expect(() => buildDocumentRecord({ ...document, path })).toThrow('path');
  }
});

test('content requires a valid Lexicon type', () => {
  for (const type of ['at.markpub.markdown', 'at.markpub.markdown#main_view']) {
    expect(buildDocumentRecord({ ...document, content: { $type: type, custom: true } }).content).toEqual({ $type: type, custom: true });
  }
  for (const content of [null, [], 'text', {}, { $type: 'invalid' }, { $type: 'at.markpub.bad-name' }, { $type: 'at.markpub.text#' }, { $type: 'at.markpub.markdown#main' }, { $type: 'at.markpub.text#x#y' }, { $type: `${'a'.repeat(64)}.example.text` }]) {
    expect(() => buildDocumentRecord({ ...document, content } as DocumentRecordInput)).toThrow('content');
  }
});

test('validates retained fields rather than trusting previous records', () => {
  expect(() => buildDocumentRecord(document, { path: '/..' })).toThrow('path');
  expect(() => buildDocumentRecord(document, { updatedAt: 'invalid' })).toThrow('updatedAt');
  expect(() => buildPublicationRecord({ url: 'https://example.com', name: 'Name' }, { description: 10 })).toThrow('description');
});

describe('native wrappers', () => {
  const reference = { uri: 'at://did:plc:example/site.standard.document/key', cid: 'bafyexample' };
  test('Offprint adds the strongRef type and retains wrapper and reference extensions', () => {
    const previous = Object.freeze({ $type: 'old', custom: true, document: { ...reference, cid: 'old', nested: 1 } });
    expect(buildNativeDocumentWrapper('offprint', reference, undefined, previous)).toEqual({
      $type: 'app.offprint.document.article', custom: true,
      document: { ...reference, nested: 1, $type: 'com.atproto.repo.strongRef' },
    });
    expect(previous.document.cid).toBe('old');
    expect(reference).not.toHaveProperty('$type');
  });
  test('Pckt requires a site and preserves or replaces previous site', () => {
    expect(() => buildNativeDocumentWrapper('pckt', reference)).toThrow('site');
    expect(() => buildNativeDocumentWrapper('pckt', reference, '')).toThrow('site');
    expect(buildNativeDocumentWrapper('pckt', reference, 'publication')).toEqual({ $type: 'blog.pckt.document', document: reference, site: 'publication' });
    expect(buildNativeDocumentWrapper('pckt', reference, undefined, { site: 'previous', extension: true })).toMatchObject({ site: 'previous', extension: true });
    expect(buildNativeDocumentWrapper('pckt', reference, 'new', { site: 'previous' }).site).toBe('new');
  });
  test('requires a nonempty reference', () => {
    expect(() => buildNativeDocumentWrapper('offprint', { ...reference, cid: '' })).toThrow('document');
  });
});
