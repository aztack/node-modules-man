## Bubble Tea + Lipgloss 渲染策略

**摘要**: 避免对多行块（header/容器）使用 lipgloss 样式包裹，以免影响终端光标位置与 list 布局；仅对行内元素着色。让 list 组件负责对齐与换行。

**关键点**:
- 仅对“行内片段”（如大小、路径、标记）用 lipgloss，上层 header 用纯文本。
- delegate 内不要套 `Width()/Align()` 等块级样式；由 list 自行布局。
- 固定 2 字符光标槽，防止移动时抖动。
- 流式数据与排序应批量更新，避免同帧 `InsertItem` + `SetItems` 造成重影。

## 大列表策略：窗口化渲染

**摘要**: 对大型结果集采用窗口化（windowed）渲染，仅输出可见行，分页导航与筛选组合，确保性能和交互响应。

**关键点**:
- 维护 `cursor` + `scrollOffset` + `visibleHeight()`，避免全量渲染。
- 过滤视图与原始数据分离（索引映射），选择/删除针对原始索引生效。
- 提供 PageUp/PageDown、Home/End、gg/G，使用户快速跳转。

**来源/上下文**: TUI 在包含数百/上千项时出现滚动迟滞，窗口化渲染显著改善交互体验。

**来源/上下文**: 2025-11-04 缩进和重复行问题定位于样式层；[参见](../docs/issues/2025-11-04_Indentation-issue.md)。
