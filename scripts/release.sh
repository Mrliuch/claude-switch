#!/usr/bin/env bash
set -euo pipefail

GITLAB="http://home.mrlch.com:18080"
PROJECT_PATH="liuchen/claude-switch"
BINARY="claude-switch"

# 目标平台：GOOS/GOARCH/后缀
PLATFORMS=(
    "darwin   amd64  "
    "darwin   arm64  "
    "linux    amd64  "
    "linux    arm64  "
    "windows  amd64  .exe"
)

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "用法: $0 <版本号>  例如: $0 v1.0.0"
    exit 1
fi

if [ -z "${GITLAB_TOKEN:-}" ]; then
    echo "错误: 请先设置 GITLAB_TOKEN 环境变量"
    echo "  export GITLAB_TOKEN=glpat-M1MbKSAn2fPVQDj_yTz3"
    exit 1
fi

PROJECT_ID="$(python3 -c "import urllib.parse; print(urllib.parse.quote('${PROJECT_PATH}', safe=''))")"
API="${GITLAB}/api/v4/projects/${PROJECT_ID}"

cd "$(dirname "$0")/.."
mkdir -p dist

# ── 1. 交叉编译所有平台 ─────────────────────────────────────
echo "==> 编译 ${VERSION}（5 个平台）"

for entry in "${PLATFORMS[@]}"; do
    read -r GOOS GOARCH EXT <<< "$entry"
    ASSET_NAME="${BINARY}-${GOOS}-${GOARCH}${EXT}"
    OUT="dist/${ASSET_NAME}"

    printf "    %-35s" "${ASSET_NAME}"
    GOOS=$GOOS GOARCH=$GOARCH CGO_ENABLED=0 \
        go build -ldflags="-s -w" -o "${OUT}" . 2>/dev/null
    echo "✓"

done

# ── 2. 上传到 Package Registry ──────────────────────────────
echo ""
echo "==> 上传到 Package Registry"

for entry in "${PLATFORMS[@]}"; do
    read -r GOOS GOARCH EXT <<< "$entry"
    ASSET_NAME="${BINARY}-${GOOS}-${GOARCH}${EXT}"
    OUT="dist/${ASSET_NAME}"

    printf "    %-35s" "${ASSET_NAME}"
    curl -sf -X PUT \
        -H "Authorization: Bearer ${GITLAB_TOKEN}" \
        -H "Content-Type: application/octet-stream" \
        --data-binary "@${OUT}" \
        "${API}/packages/generic/${BINARY}/${VERSION}/${ASSET_NAME}" > /dev/null
    echo "✓"
done

# ── 3. 打 tag ───────────────────────────────────────────────
echo ""
echo "==> 创建 git tag ${VERSION}"
git tag "${VERSION}" 2>/dev/null && git push origin "${VERSION}" || echo "    tag 已存在，跳过"

# ── 4. 创建 Release，附上所有平台下载链接 ────────────────────
echo "==> 创建 GitLab Release ${VERSION}"

python3 - <<PYEOF
import json, urllib.request, urllib.error

gitlab   = "${GITLAB}"
base_url = "${GITLAB}/${PROJECT_PATH}/-/packages/generic/${BINARY}/${VERSION}"
binary   = "${BINARY}"
version  = "${VERSION}"
token    = "${GITLAB_TOKEN}"
api      = "${API}"

platforms = [
    ("darwin",  "amd64", ""),
    ("darwin",  "arm64", ""),
    ("linux",   "amd64", ""),
    ("linux",   "arm64", ""),
    ("windows", "amd64", ".exe"),
]

links = []
install_lines = []
for goos, goarch, ext in platforms:
    name = f"{binary}-{goos}-{goarch}{ext}"
    url  = f"{base_url}/{name}"
    links.append({"name": name, "url": url, "link_type": "package"})
    install_lines.append(f"# {goos}/{goarch}\ncurl -L \"{url}\" -o /usr/local/bin/{binary} && chmod +x /usr/local/bin/{binary}")

desc = "## 各平台安装\n\n" + "\n\n".join(
    f"### {goos}/{goarch}\n\`\`\`bash\n# 下载\ncurl -L \"{base_url}/{binary}-{goos}-{goarch}{ext}\" -o /usr/local/bin/{binary}{ext}\nchmod +x /usr/local/bin/{binary}{ext}\n\`\`\`"
    for goos, goarch, ext in platforms
)

payload = {
    "name": version,
    "tag_name": version,
    "description": desc,
    "assets": {"links": links}
}

req = urllib.request.Request(
    f"{api}/releases",
    data=json.dumps(payload).encode(),
    headers={"Authorization": f"Bearer {token}", "Content-Type": "application/json"},
    method="POST"
)
try:
    with urllib.request.urlopen(req) as resp:
        d = json.load(resp)
        print(f"    Release 创建成功: {d.get('name','')}")
except urllib.error.HTTPError as e:
    body = json.load(e)
    print(f"    注意: {body.get('message','')}")
PYEOF

# ── 5. 清理 dist ────────────────────────────────────────────
rm -rf dist/

echo ""
echo "完成！"
echo "Release 页面: ${GITLAB}/${PROJECT_PATH}/-/releases/${VERSION}"
echo ""
echo "各平台下载地址:"
for entry in "${PLATFORMS[@]}"; do
    read -r GOOS GOARCH EXT <<< "$entry"
    ASSET_NAME="${BINARY}-${GOOS}-${GOARCH}${EXT}"
    printf "  %-10s %-6s  %s\n" "${GOOS}" "${GOARCH}" \
        "${GITLAB}/${PROJECT_PATH}/-/packages/generic/${BINARY}/${VERSION}/${ASSET_NAME}"
done
