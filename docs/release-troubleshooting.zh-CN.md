# 发布故障排查

English: [release-troubleshooting.md](release-troubleshooting.md)

本文档是 preview release failures 的 operator triage path。它保持 local-first 和 terminal-first；不增加 hosted orchestration、dashboards、cloud sync、SaaS release tracking、automatic Git execution、provider-native resume 或默认真实 Codex/Claude gates。

## 第一响应

当 required release check 失败时：

- 停止该 release candidate。不要 tag、announce 或 publish artifact。
- 在 operator notes 中保留失败 command、exit status、相关 output、source form、commit value、`VERSION`、`BUILD_DATE`、Go version 和任何 environment overrides。
- 修改前，先从同一个 source form 重新运行最小失败 command。如果只在 `scripts/check-all.sh` 中失败，应检查 aggregate ordering 和 temporary directory setup。
- 把 failure 分类为 ADP regression、documentation drift、package assembly error、source archive error、checksum mismatch、install rehearsal error 或 operator environment issue。
- 修复后，先重新运行失败 command，再重新运行 `scripts/check-all.sh`，然后才记录 passing release evidence。

不要把可选真实 Codex 或 Claude evidence 当作默认门禁。只有 release note 声明了 deterministic fake-provider evidence 之外的 real-agent compatibility 时，real CLI failure 才会阻塞 release。

## Release Candidate 决策

继续前先对 candidate 分类：

- `blocked`：任何 required gate、checksum、package manifest、install rehearsal、source archive rehearsal 或 project-root cleanliness check 失败。
- `passing-provider-free`：所有 required deterministic checks 通过，optional real-agent tiers 均为 `not run` 或没有被声明。
- `passing-with-real-agent-evidence`：所有 required deterministic checks 通过，并且明确声明的 optional real-agent tiers 也通过。

不要把 required failure 降级为 optional evidence。不要把 optional real-agent evidence 升级成默认 release requirement。如果 release note 声明了某个 real-agent tier 但该 tier 失败，应移除该声明并把该 tier 记录为 failed 或 deferred，或者停止 candidate 直到该 tier 通过。

## Source Form 失败

对于 Git checkout，从这些命令开始：

```bash
git status --short --branch
git rev-parse HEAD
```

意外的 tracked changes 表示 release source 不干净。`.envrc` 和 `mvp.md` 等 ignored local files 应继续保持 ignored 和 uncommitted。

对于没有 `.git` 的 source archive，使用显式 commit 值构建：

```bash
COMMIT=<published-commit-or-archive-id>
```

如果 archive build 需要 operator machine 上的文件，应从干净 checkout 重新构建 archive。不要把 archive contents 和本机本地 ADP state 混合。

## 构建和版本失败

如果 `adp version` 输出 `adp dev`，说明 release ldflags 没有注入。按 [release-packaging.zh-CN.md](release-packaging.zh-CN.md) 使用明确的 `VERSION`、`COMMIT` 和 `BUILD_DATE` 重新构建。

如果报告的 commit 或 build date 与 release evidence 不一致，应丢弃该 artifact 并重新构建。不要修改 evidence 去匹配意外 binary。

## Checksum 失败

checksum evidence 必须指向将要打包的同一个 artifact：

```bash
sha256sum dist/adp > dist/adp.sha256
sha256sum -c dist/adp.sha256
```

如果 verification 失败，丢弃 checksum 和 artifact 这一组，重新构建 artifact，重新生成 checksum，并再次验证。记录 checksum 后不要再修改 artifact。

## Package Manifest 失败

发布前检查 archive contents：

```bash
tar -tf adp-0.1.0-preview.1-linux-amd64.tar.gz | sort
```

package 必须包含 binary、`README.md`、`README.zh-CN.md`、`LICENSE`、`COMMERCIAL.md`、`COMMERCIAL.zh-CN.md`、release packaging docs、release evidence docs 和简短 release note。必须排除 `.envrc`、`mvp.md`、`$ADP_HOME`、`$ADP_RUNTIME_DIR`、runtime overlays、logs、task state、credentials、shell startup files 和临时 rehearsal directories。

如果 required files 缺失，应修复干净 staging directory 并重新构建 package。如果 excluded files 出现，应修复 package assembly path；不能削弱 local-first boundary 或发布 operator state。

## 安装演练失败

从 packaged artifact path 安装并运行，不要从 source tree 运行：

```bash
ADP_INSTALL_BIN="$(mktemp -d)"
install -m 0755 dist/adp "${ADP_INSTALL_BIN}/adp"
PATH="${ADP_INSTALL_BIN}:${PATH}" adp version
```

如果 installed binary 失败但 `dist/adp` 成功，应检查 file permissions、package extraction、target platform 和 `PATH` ordering。rehearsal 必须使用临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、临时 project root 和 fake provider commands，除非已明确启用 optional real CLI evidence。

如果 project-root pollution scan 找到 ADP files，应修复 runtime 或 planning output boundaries。不能接受真实 project root 中出现 `AGENTS.md`、`CLAUDE.md`、`.codex`、`.claude`、`.adp-runtime.yaml`、`planning`、task files、phase files 或 progress reports。

## 门禁失败

对于 `scripts/check-all.sh`，检查第一个失败的 child command，并从同一个 source form 直接重新运行该 child。Aggregate gate 是 release decision point，但最小失败 command 通常是最快的 triage target。

对于 `scripts/release-artifact-smoke.sh`，优先检查 package staging、checksums、manifest assertions、install-from-artifact、source archive `COMMIT`、fake Codex command、临时 ADP directories 和 project-root pollution output。

对于 `scripts/release-operator-drill-smoke.sh`，优先检查 no-`.git` source copy、文档化 release commands、release script syntax checks、显式 commit build、checksum verification、installed `PATH` binary、fake Codex handoff sequence、本地 phase evidence records、fake Git tripwire 和 project-root pollution scan。

对于 `scripts/install-onboarding-smoke.sh`，优先检查 deterministic build metadata、临时 `GOBIN` install、`PATH` ordering、临时 `ADP_HOME`、临时 `ADP_RUNTIME_DIR`、workspace registration、fake Codex 与 fake Claude commands、task-bound evidence、fake Git tripwire output 和 project-root pollution scan。

对于 `scripts/release-rehearsal-smoke.sh`，优先检查 clean workspace copy、release ldflags、复制后的 example workspace bootstrap、隔离的 runtime directories 和 fake Git tripwire output。

对于 `scripts/check-docs-bilingual.sh`，补齐缺失的 English default 或 Simplified Chinese counterpart。对于 `scripts/check-file-lines.sh`，在新增行为前拆分报告的 code file。对于 `git diff --check`，移除 whitespace errors 或 conflict markers。

不确定时，保持 failure narrow：重新运行失败步骤，修复本地原因，再重新运行 aggregate gate。不能通过增加新的 product scope 来解决 release failures。

## 可选 Real-Agent 失败

对于 command availability 失败，确认外部 CLI 已安装在 `PATH` 上，然后只重新运行已经 opt in 的 real flag。不要把 real flag 加入 `scripts/check-all.sh` 或 CI。

对于非交互 invocation 失败，记录 failure 属于 command launch、authentication、quota、network、model access、prompt handling、timeout 还是 unexpected output。除非 deterministic fake-agent gate 也失败，否则这些 failure 描述的是 operator environment。

对于手工交互式 acceptance 失败，保持 notes 已脱敏，并与 package evidence 分开。不要在 release artifacts 中包含 credentials、账号标识、私有 prompts、provider-native session files 或敏感 model output。
