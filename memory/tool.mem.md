## `bubbletea` + `bubbles/list` 使用要点（v0.24.2 / v0.16.1）

**功能**: TUI 列表展示与交互（滚动、选择、过滤等）。

**要点**:
- `ItemDelegate.Render` 应使用 `fmt.Fprint`（无结尾换行）；换行由 list 控制。
- `Height()=1`, `Spacing()=0` 可实现紧凑列表；如需固定左槽，手动拼接前缀（例如 `"> "` / `"  "`）。
- 避免在 delegate 内对整行套 `lipgloss.Width/Align`；可能与 list 自己的布局冲突，导致缩进/重影。
- 关闭不需要的 UI：`SetShowTitle/StatusBar/Help/Pagination(false)`，减少隐式左缩进。
- `WindowSizeMsg` 后调用 `SetSize(w,h)`；`h` 可用 header 固定行数扣减，而非按换行计数。
- 过滤 API 在旧版本无 `StartFiltering()`；仅使用 `SetFilteringEnabled()` 切换。

**用法示例**:
```go
// 渲染
prefix := "  "
if index == m.Index() { prefix = "> " }
line := fmt.Sprintf("%s%s %s %s", prefix, mark, sizeStr, path)
fmt.Fprint(w, line) // 不要 Fprintln
```

**注意事项/限制**:
- 与 `lipgloss` 混用时，尽量只对“片段”着色，不要整行 Render。
- 流式更新：避免同帧 InsertItem + SetItems；批量 SetItems 更安全。
