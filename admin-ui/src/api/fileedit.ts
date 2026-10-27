import { http, read } from './http';
import type { components } from './schema';
export type FileRoot = 'lua' | 'tedit';
export type FileEntry = components['schemas']['Entry'];
export type FileAction = components['schemas']['SchemaAction'];
export type FileListing = components['schemas']['FileListingResponse'];
export type EditableFile = components['schemas']['File'];
export type ScriptUsage = components['schemas']['ScriptUsage'];
export type ScriptResolution = components['schemas']['ScriptResolveResponse'];

export const fileEditApi = {
  listing: (root: FileRoot, directory = '') => read(http.GET('/admin/files/{root}/listing', { params: { path: { root }, query: { directory } } })),
  read: async (root: FileRoot, path: string): Promise<EditableFile> => {
    const result = await http.GET('/admin/files/{root}/content', { params: { path: { root }, query: { path } } });
    const file = await read(Promise.resolve(result));
    return { ...file, etag: result.response.headers.get('ETag') || file.etag };
  },
  // Keep conditional writes; never overwrite a file read at another revision.
  save: async (root: FileRoot, path: string, content: string, etag: string | null): Promise<EditableFile> => {
    const result = await http.PUT('/admin/files/{root}/content', {
      params: { path: { root }, query: { path }, header: etag === null ? { 'If-None-Match': '*' } : { 'If-Match': etag } }, body: { content },
    });
    const file = await read(Promise.resolve(result));
    return { ...file, etag: result.response.headers.get('ETag') || file.etag };
  },
  remove: (root: FileRoot, path: string, etag: string) => read(http.DELETE('/admin/files/{root}/content', { params: { path: { root }, query: { path }, header: { 'If-Match': etag } } })),
  usage: (path: string) => read(http.GET('/admin/files/lua/usage', { params: { query: { path } } })),
  resolve: (name: string, kind?: 'room' | 'mob' | 'obj') => read(http.GET('/admin/files/lua/resolve', { params: { query: { name, kind } } })),
};
