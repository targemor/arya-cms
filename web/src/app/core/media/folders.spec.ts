import { ROOT_KEY, childrenOf, destinationTree, keyToFolderId, pathTo, subtreeIds } from './folders';
import { Folder } from './media.service';

function folder(id: string, name: string, parentId: string | null = null): Folder {
  return { id, name, parentId, createdAt: '', updatedAt: '' };
}

// photos/ ─ covers/ ─ 2026/     docs/
const tree = [
  folder('photos', 'Fotos'),
  folder('covers', 'Portadas', 'photos'),
  folder('y2026', '2026', 'covers'),
  folder('docs', 'documentos'),
];

describe('folder tree helpers', () => {
  it('lists children sorted by name, case-insensitively', () => {
    expect(childrenOf(tree, null).map((f) => f.name)).toEqual(['documentos', 'Fotos']);
    expect(childrenOf(tree, 'photos').map((f) => f.id)).toEqual(['covers']);
  });

  it('builds the breadcrumb path from the root', () => {
    expect(pathTo(tree, 'y2026').map((f) => f.name)).toEqual(['Fotos', 'Portadas', '2026']);
    expect(pathTo(tree, null)).toEqual([]);
    expect(pathTo(tree, 'missing')).toEqual([]);
  });

  it('stops on a cyclic chain instead of looping', () => {
    const cyclic = [folder('a', 'A', 'b'), folder('b', 'B', 'a')];

    expect(pathTo(cyclic, 'a').length).toBeLessThanOrEqual(3);
  });

  it('collects a folder and everything below it', () => {
    expect([...subtreeIds(tree, 'photos')].sort()).toEqual(['covers', 'photos', 'y2026']);
  });

  it('offers the root and every folder as move destinations, blocking the moved subtree', () => {
    const [root] = destinationTree(tree, 'covers');

    expect(root.key).toBe(ROOT_KEY);
    const photos = root.children!.find((n) => n.key === 'photos')!;
    const covers = photos.children![0];

    expect(photos.disabled).toBe(false);
    expect(covers.disabled).toBe(true);
    expect(covers.children![0].disabled).toBe(true);
  });

  it('maps the root key back to null', () => {
    expect(keyToFolderId(ROOT_KEY)).toBeNull();
    expect(keyToFolderId('photos')).toBe('photos');
  });
});
