import { request, requestResponse } from './client';

export type FileRoot = 'lua' | 'tedit';

export interface FileEntry {
  name: string;
  path: string;
  directory: boolean;
  max_bytes?: number;
  required_level?: number;
  required_label?: string;
  allowed: boolean;
}

export interface FileAction {
  key: string;
  label: string;
  required_level: number;
  required_label: string;
  allowed: boolean;
}

export interface FileListing {
  entries: FileEntry[];
  actions: FileAction[];
}

export interface EditableFile {
  path: string;
  content: string;
  etag: string;
}

export interface ScriptUsage {
  kind: 'room' | 'mob' | 'obj';
  vnum: number;
  name: string;
  script_name: string;
}

export interface ScriptResolution {
  path: string;
  exists: boolean;
}

const query = (path: string) => `?path=${encodeURIComponent(path)}`;

export const fileEditApi = {
  listing: (root: FileRoot, directory = '') =>
    request<FileListing>(`/files/${root}/listing${directory ? `?directory=${encodeURIComponent(directory)}` : ''}`),
  read: async (root: FileRoot, path: string): Promise<EditableFile> => {
    const response = await requestResponse(`/files/${root}/content${query(path)}`);
    const file = await response.json() as EditableFile;
    return { ...file, etag: response.headers.get('ETag') || file.etag };
  },
  // etag null creates the file (If-None-Match: *); a string replaces only
  // the exact content that was read (If-Match). There is no unconditional save.
  save: async (root: FileRoot, path: string, content: string, etag: string | null): Promise<EditableFile> => {
    const response = await requestResponse(`/files/${root}/content${query(path)}`, {
      method: 'PUT',
      headers: etag === null ? { 'If-None-Match': '*' } : { 'If-Match': etag },
      body: JSON.stringify({ content }),
    });
    const file = await response.json() as EditableFile;
    return { ...file, etag: response.headers.get('ETag') || file.etag };
  },
  remove: (root: FileRoot, path: string, etag: string) =>
    request<void>(`/files/${root}/content${query(path)}`, { method: 'DELETE', headers: { 'If-Match': etag } }),
  usage: (path: string) => request<ScriptUsage[]>(`/files/lua/usage${query(path)}`),
  resolve: (name: string, kind?: string) =>
    request<ScriptResolution>(`/files/lua/resolve?name=${encodeURIComponent(name)}${kind ? `&kind=${encodeURIComponent(kind)}` : ''}`),
};
