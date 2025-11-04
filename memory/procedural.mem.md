## Fix Bubble Tea TUI Indentation & Duplicate Lines

**目标**: 使 TUI 列表项左对齐（除固定 2 字符光标槽外），并在上下移动时不出现重复行/重影。

**前置条件**:
- 使用 Bubble Tea + Bubbles `list`（v0.24.2 + v0.16.1）
- 自定义 `list.ItemDelegate` 单行渲染
- 顶部有自定义 Header 文本

**步骤**:
1. Header 仅用纯文本输出，不要用 lipgloss 对整段多行进行 `Render()` 包裹。
2. 在 `delegate.Render` 中：
   - 采用固定 2 字符左槽：选中行前缀为 `"> "`，未选中为两个空格。
   - 仅对“行内片段”着色（选择标记、容量、路径）；不要对整行进行 `Width/Align` 样式包裹。
   - 使用 `fmt.Fprint(w, line)` 输出（不要 `Fprintln` 换行），换行由 list 控制。
3. 关闭 list 额外 UI 与边距：`SetShowTitle(false)`, `SetShowStatusBar(false)`, `SetShowHelp(false)`, `SetShowPagination(false)`；避免额外左缩进。
4. 处理窗口大小：在 `tea.WindowSizeMsg` 中直接调用 `m.l.SetSize(msg.Width, 期望高度)`；高度可用固定 Header 行数（建议常量 2）进行扣减，避免用 `Count('\n')` 动态计算带来偏移。
5. 流式扫描时的更新策略：
   - 避免同一帧对 list 调用 `InsertItem()` 后又 `SetItems()`；改为收集一批增量后仅 `SetItems()` 一次，或维护有序切片后直接 `SetItems()`。
   - 排序稳定，避免频繁重排导致可见闪烁或重复。
6. 验证：启动后不触键时首行即左对齐；上下移动不会产生重复行；颜色与选择状态正常。

**验证**:
- 运行并观察首行左边距；多次按上下键确认无重复行；调整窗口大小验证列表高度与对齐无异常。
