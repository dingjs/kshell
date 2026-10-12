// collectVisibleRows 单测：shift+点击范围多选依赖「当前可见平铺行序列」，
// 顺序必须与渲染顺序一致（目录展开时子层内联在目录行之后）。
import { describe, expect, it } from 'vitest';
import { collectVisibleRows, type TreeRowItem } from './treeRows';

const dir = (relPath: string, children: TreeRowItem[], expanded = true): TreeRowItem => ({
  relPath,
  node: { IsDir: true },
  expanded,
  children,
});
const file = (relPath: string): TreeRowItem => ({
  relPath,
  node: { IsDir: false },
  expanded: false,
  children: null,
});

describe('collectVisibleRows', () => {
  it('按渲染顺序平铺已展开的行，跳过虚拟根但下钻其子层', () => {
    const items: TreeRowItem[] = [
      dir('', [file('a.ts'), dir('pkg', [file('pkg/b.ts')]), file('c.ts')]),
    ];
    expect(collectVisibleRows(items)).toEqual(['a.ts', 'pkg', 'pkg/b.ts', 'c.ts']);
  });

  it('收起的目录不收集子层（懒加载未展开时不可见）', () => {
    const items: TreeRowItem[] = [dir('', [dir('pkg', [file('pkg/b.ts')], false), file('c.ts')])];
    expect(collectVisibleRows(items)).toEqual(['pkg', 'c.ts']);
  });

  it('空树与空根层返回空数组', () => {
    expect(collectVisibleRows([])).toEqual([]);
    expect(collectVisibleRows([dir('', [])])).toEqual([]);
  });
});
