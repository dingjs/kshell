// 文件树可见行收集：按渲染顺序输出 relPath 平铺数组（展开目录的子层内联在目录行之后）。
// shift+点击范围多选依赖「当前可见平铺行序列」在此序列上取锚点到目标行的区间；
// 虚拟根行（relPath 为空串）只是容器，不参与选中与删除，收集时跳过但仍下钻其子层。
// FileTree 的 TreeItem 结构兼容本文件的最小结构描述（结构化类型，避免循环依赖）。
export interface TreeRowItem {
  relPath: string;
  node: { IsDir: boolean };
  expanded: boolean;
  children: TreeRowItem[] | null;
}

export function collectVisibleRows(items: TreeRowItem[]): string[] {
  const out: string[] = [];
  const walk = (list: TreeRowItem[]) => {
    for (const it of list) {
      if (it.relPath !== '') out.push(it.relPath);
      if (it.node.IsDir && it.expanded && it.children) walk(it.children);
    }
  };
  walk(items);
  return out;
}
