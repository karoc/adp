# GitHub Release 发布指南

## 状态

- ✅ 版本号已更新为 1.0.0
- ✅ Git tag v1.0.0 已创建
- ✅ Tag 已推送到远程仓库
- ✅ 多平台二进制文件已构建
- ✅ SHA256 校验和已生成
- ✅ 发布说明已准备
- ⚠️  GitHub Release 创建需要额外权限

## GitHub Token 权限问题

自动创建 GitHub Release 失败，原因：
```
HTTP 403: Resource not accessible by personal access token
```

当前 token 缺少创建 Release 所需的权限。

## 手动创建 Release 步骤

### 方式 1: 通过 GitHub Web UI

1. 访问 https://github.com/karoc/adp/releases/new

2. 选择 tag: `v1.0.0`

3. Release title: `ADP 1.0.0 - First Production Release`

4. 复制 `dist/RELEASE_NOTES.md` 的内容到 description 区域

5. 上传以下文件到 "Attach binaries" 区域：
   - `dist/adp-1.0.0-linux-amd64`
   - `dist/adp-1.0.0-linux-arm64`
   - `dist/adp-1.0.0-darwin-amd64`
   - `dist/adp-1.0.0-darwin-arm64`
   - `dist/adp-1.0.0-windows-amd64.exe`
   - `dist/SHA256SUMS`

6. 点击 "Publish release"

### 方式 2: 刷新 gh CLI 权限后重试

```bash
# 重新认证并添加 repo 权限
gh auth refresh -h github.com -s repo

# 创建 Release
gh release create v1.0.0 \
  --title "ADP 1.0.0 - First Production Release" \
  --notes-file dist/RELEASE_NOTES.md \
  dist/adp-1.0.0-linux-amd64 \
  dist/adp-1.0.0-linux-arm64 \
  dist/adp-1.0.0-darwin-amd64 \
  dist/adp-1.0.0-darwin-arm64 \
  dist/adp-1.0.0-windows-amd64.exe \
  dist/SHA256SUMS
```

## 验证 Release

发布后，验证以下内容：

1. **Tag 可见**:
   ```bash
   gh release view v1.0.0
   ```

2. **二进制文件可下载**:
   ```bash
   curl -LO https://github.com/karoc/adp/releases/download/v1.0.0/adp-1.0.0-linux-amd64
   ```

3. **校验和文件可访问**:
   ```bash
   curl -LO https://github.com/karoc/adp/releases/download/v1.0.0/SHA256SUMS
   sha256sum -c SHA256SUMS
   ```

4. **发布说明完整**:
   - 检查 Release 页面显示完整的 markdown 格式说明
   - 所有链接可点击且正确

## 二进制文件清单

所有文件已准备在 `dist/` 目录：

```
dist/
├── SHA256SUMS                        (448 字节)
├── adp-1.0.0-darwin-amd64           (6.2 MB)
├── adp-1.0.0-darwin-arm64           (5.8 MB)
├── adp-1.0.0-linux-amd64            (6.1 MB)
├── adp-1.0.0-linux-arm64            (5.8 MB)
├── adp-1.0.0-windows-amd64.exe      (6.3 MB)
└── RELEASE_NOTES.md                  (6.5 KB)
```

## SHA256 校验和

```
0094b3a5efe3eefaa98efcd00b682886d5e626c70c74b77f93f2737c890fe21e  adp-1.0.0-darwin-amd64
1a9673ef53bafea36bcbf6cec11e28c73a137285badc2227ad054b1fd525d2a3  adp-1.0.0-darwin-arm64
ab3b06ec104cc7f4d9a8d1b4c9ed9abbf6b7b50e138152b091b5b7fb33a4c693  adp-1.0.0-linux-amd64
201ed03458ad124b427d8b8e23fd3d4672a858bbd9b2989cd82e7c30cf59206a  adp-1.0.0-linux-arm64
beb8f79c728f657d50403c931b38b878fa306ed072311b1299fc37e56b135b84  adp-1.0.0-windows-amd64.exe
```

## 完成后的公告

Release 发布后，建议在以下渠道公告：

1. **项目 README** - 添加 "Latest Release" badge
2. **CHANGELOG** - 更新 `[1.0.0]` 部分的日期
3. **社区渠道** - 根据项目需要发布公告

---

**当前状态**: 等待手动完成 GitHub Release 创建
**下一步**: 访问 https://github.com/karoc/adp/releases/new 完成发布
