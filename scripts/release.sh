#!/usr/bin/env bash
set -euo pipefail

GITLAB="http://home.mrlch.com:18080"
PROJECT_PATH="liuchen/claude-switch"
BINARY="claude-switch"
OS_ARCH="darwin-amd64"

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
ASSET_NAME="${BINARY}-${OS_ARCH}"
PACKAGE_URL="${GITLAB}/${PROJECT_PATH}/-/packages/generic/${BINARY}/${VERSION}/${ASSET_NAME}"

echo "==> 编译 ${BINARY} ${VERSION}"
cd "$(dirname "$0")/.."
CGO_ENABLED=0 go build -ldflags="-s -w" -o "${BINARY}" .

echo "==> 上传二进制到 Package Registry"
curl -sf -X PUT \
    -H "Authorization: Bearer ${GITLAB_TOKEN}" \
    -H "Content-Type: application/octet-stream" \
    --data-binary "@${BINARY}" \
    "${API}/packages/generic/${BINARY}/${VERSION}/${ASSET_NAME}" > /dev/null
echo "    上传完成: ${PACKAGE_URL}"

echo "==> 创建 git tag ${VERSION}"
git tag "${VERSION}" 2>/dev/null && git push origin "${VERSION}" || echo "    tag 已存在，跳过"

echo "==> 创建 GitLab Release ${VERSION}"
INSTALL_CMD="curl -L \\\"${PACKAGE_URL}\\\" -o /usr/local/bin/${BINARY} && chmod +x /usr/local/bin/${BINARY}"
python3 - <<PYEOF
import json, urllib.request, urllib.error

payload = {
    "name": "${VERSION}",
    "tag_name": "${VERSION}",
    "description": "## 安装\n\`\`\`bash\n${INSTALL_CMD}\n\`\`\`",
    "assets": {
        "links": [{
            "name": "${ASSET_NAME}",
            "url": "${PACKAGE_URL}",
            "link_type": "package"
        }]
    }
}

req = urllib.request.Request(
    "${API}/releases",
    data=json.dumps(payload).encode(),
    headers={"Authorization": "Bearer ${GITLAB_TOKEN}", "Content-Type": "application/json"},
    method="POST"
)
try:
    with urllib.request.urlopen(req) as resp:
        d = json.load(resp)
        print(f"    Release 创建成功: {d.get('name','')}")
except urllib.error.HTTPError as e:
    d = json.load(e)
    print(f"    注意: {d.get('message','')}")
PYEOF

echo ""
echo "完成！"
echo "Release 页面: ${GITLAB}/${PROJECT_PATH}/-/releases/${VERSION}"
echo "下载命令:"
echo "  curl -L \"${PACKAGE_URL}\" -o /usr/local/bin/${BINARY} && chmod +x /usr/local/bin/${BINARY}"
