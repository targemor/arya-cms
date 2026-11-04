import { FileSizePipe } from './file-size.pipe';

describe('FileSizePipe', () => {
  const pipe = new FileSizePipe();

  it.each([
    [0, '0 B'],
    [512, '512 B'],
    [1024, '1 KB'],
    [52341, '51,1 KB'],
    [10 * 1024 * 1024, '10 MB'],
    [150 * 1024 * 1024, '150 MB'],
    [null, ''],
    [-1, ''],
  ])('%s → %s', (bytes, expected) => {
    expect(pipe.transform(bytes)).toBe(expected);
  });
});
