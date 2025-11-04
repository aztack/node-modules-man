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

## Windowed List Rendering with Paging (Bubble Tea custom list)

**目标**: 在大量条目时仅渲染可见窗口，支持 PageUp/PageDown、Home/End、gg/G 导航，保持滚动流畅和选择准确。

**前置条件**:
- 使用自定义列表切片渲染（非 bubbles/list），维护 `items`, `cursor`, `scrollOffset`。

**步骤**:
1. 计算可见高度：`visibleHeight = termH - headerLines - 1`，最小 3。
2. 根据过滤结果生成 `viewIndexes()`（索引切片），窗口 `[scrollOffset : scrollOffset+visibleHeight]` 进行渲染。
3. 光标移动时使用 `adjustScroll()` 保证光标在窗口内；超界时收敛到边界。
4. 绑定按键：
   - `pgup/ctrl+b`：`cursor -= visibleHeight`
   - `pgdown/ctrl+f`：`cursor += visibleHeight`
   - `home`/`gg`：`cursor = 0`
   - `end`/`G`：`cursor = len(view)-1`
5. 选择切换使用 `idx := view[cursor]` 映射到原始 `items`，保持筛选视图与实际数据一致。

**验证**:
- 大列表滚动无卡顿；分页键工作；过滤后分页与选择仍正确；删除后列表与光标位置合理收敛。

## 夹具生成：包含可搜索标记的项目

**目标**: 通过生成名为 `text-app-<随机>` 的夹具项目，便于在 TUI 中验证搜索/过滤（输入 `text` 应能快速定位）。

**步骤**:
1. 修改 `scripts/make-test-fixtures.js`，在常规 `react-app-*` 之外额外创建 `text-app-<rand>`。
2. 在其 `src/index.tsx` 写入注释与日志，方便识别（非必须）。
3. 安装依赖可选（`--no-install`），删除测试仅需存在目录结构即可。

**验证**:
- 运行脚本后确认生成的目录名包含 `text`；在 TUI 中输入 `/text` 可筛到该项。
