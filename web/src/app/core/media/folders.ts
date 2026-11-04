import { NzTreeNodeOptions } from 'ng-zorro-antd/core/tree';

import { Folder } from './media.service';

/** Tree-select key standing for the library root, which has no folder id. */
export const ROOT_KEY = '__root__';
export const ROOT_NAME = 'Biblioteca';

/** Folders directly inside parentId (null: the root), sorted by name. */
export function childrenOf(folders: readonly Folder[], parentId: string | null): Folder[] {
  return folders
    .filter((f) => f.parentId === parentId)
    .sort((a, b) => a.name.localeCompare(b.name, 'es', { sensitivity: 'base' }));
}

/**
 * The folders from the root down to id, for breadcrumbs. Empty for the root or an unknown id; a
 * corrupted, cyclic chain stops instead of looping.
 */
export function pathTo(folders: readonly Folder[], id: string | null): Folder[] {
  const byId = new Map(folders.map((f) => [f.id, f]));
  const path: Folder[] = [];

  for (let current = id ? byId.get(id) : undefined; current && path.length <= folders.length; ) {
    path.unshift(current);
    current = current.parentId ? byId.get(current.parentId) : undefined;
  }

  return path;
}

/** Ids of a folder and everything below it. */
export function subtreeIds(folders: readonly Folder[], id: string): Set<string> {
  const ids = new Set([id]);
  let grew = true;

  while (grew) {
    grew = false;

    for (const f of folders) {
      if (f.parentId && ids.has(f.parentId) && !ids.has(f.id)) {
        ids.add(f.id);
        grew = true;
      }
    }
  }

  return ids;
}

/**
 * Destinations for "Mover a…": the root plus every folder, as an nz-tree-select tree. When moving
 * a folder, pass its id as `moving`: it and its subfolders are disabled, since a folder cannot go
 * inside itself.
 */
export function destinationTree(folders: readonly Folder[], moving?: string): NzTreeNodeOptions[] {
  const blocked = moving ? subtreeIds(folders, moving) : new Set<string>();

  const build = (parentId: string | null): NzTreeNodeOptions[] =>
    childrenOf(folders, parentId).map((f) => {
      const children = build(f.id);

      return {
        title: f.name,
        key: f.id,
        disabled: blocked.has(f.id),
        isLeaf: children.length === 0,
        children,
        expanded: true,
      };
    });

  return [{ title: ROOT_NAME, key: ROOT_KEY, isLeaf: false, expanded: true, children: build(null) }];
}

/** A tree-select key back to a folder id (null: the root). */
export function keyToFolderId(key: string): string | null {
  return key === ROOT_KEY ? null : key;
}
