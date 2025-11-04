# 项目进度记录（node-module-man）

## 2025-11-03

- 初始化实现计划：`docs/plan.md`（里程碑、架构、测试与验收标准）。
- M1 开始：
  - 初始化 Go 模块：`go.mod`。
  - 目录结构搭建：`cmd/`, `internal/scanner`, `pkg/utils`。
  - 实现核心扫描器雏形：
    - 遍历查找 `node_modules` 目录（可限深）。
    - 并发计算目录大小（可配置并发度）。
    - 忽略符号链接目录，收集遍历错误但不中断。
- 实现基础 CLI（M2 部分）：
  - 入口：`cmd/node-module-man/main.go`
  - 参数：`--path`, `--concurrency`, `--max-depth`, `--json`。
  - 输出：表格/JSON，汇总总大小与耗时。

- M3 进行中：
  - 加入最小 TUI 骨架（Bubble Tea）：`internal/tui/model.go`
  - 在 CLI 中加入 `--tui` 开关；TUI 展示扫描中加载动画与结果汇总，按 `q` 退出。
  - 当前扫描为一次性完成后展示（后续可做进度流式化）。

- 兼容性修复：
  - 移除 `errors.Join` 依赖，适配 Go 1.19（自定义多错误聚合）。

- 测试（初版）：
  - `internal/scanner/scanner_test.go`：基础发现与大小统计、`MaxDepth` 行为。
  - `pkg/utils/size_test.go`：`HumanizeBytes` 单元测试。

- TUI 列表交互（进行中）：
  - 引入 `bubbles/list`，将扫描结果渲染为可滚动列表。
  - 多选：`space` 切换选中，累计选中体积。
  - 排序：`s` 切换字段（size/path），`r` 升降序。
  - 筛选：`/` 切换过滤输入。

- 删除流程（初版）：
  - 新增 `internal/deleter`，并发删除选中目录，逐项上报进度，汇总成功/失败与释放空间。
  - TUI：`d/enter` 打开确认，`y` 确认后进入删除进度视图，完成后显示总结并从列表中移除成功项，更新总占用。

- 取消与 Dry-Run：
  - CLI 加入 `--dry-run`，TUI 删除阶段支持模拟删除，不实际执行（仍统计释放空间）。
  - 删除进行时按 `q/esc` 可取消，已完成的删除会计入汇总，未完成的停止。

- 排除规则（Exclude）：
  - `scanner.Options.Excludes` 支持 glob；在遍历阶段跳过匹配的目录。
  - CLI 新增 `--exclude` 可重复传入，匹配全路径或基名。
  - 测试覆盖：`TestScanNodeModules_Exclude`。

- 流式扫描（Streaming）：
  - 新增 `scanner.ScanNodeModulesStream`，在大小计算完成后逐条产出结果。
  - TUI 改为边扫边渲染列表与总量，头部显示 `Scanning...` 以及计数与耗时。

- 构建脚本：
  - 新增 `build.sh`，支持 macOS (arm64/amd64) 以及常见平台（`all`）。
  - 受沙箱限制未在当前环境实际构建；可在本地终端执行 `bash build.sh` 生成二进制到 `dist/`。

下一步计划：
- 完成 M1/M2 的边界条件与健壮性检查（如排除模式、上下文取消）。
- 添加单元测试（scanner、utils），并准备集成测试脚手架。
- TUI：引入列表组件、排序/筛选与多选选择器；删除流程设计草图。
