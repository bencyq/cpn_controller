#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PYTHON_BIN="${PYTHON_BIN:-/opt/miniconda3/bin/python}"
PYINSTALLER_VERSION="${PYINSTALLER_VERSION:-6.19.0}"
PIP_INDEX_URL="${PIP_INDEX_URL:-https://pypi.tuna.tsinghua.edu.cn/simple}"

if ! "$PYTHON_BIN" -m PyInstaller --version >/dev/null 2>&1; then
  "$PYTHON_BIN" -m pip install --user wheel -i "$PIP_INDEX_URL"
  "$PYTHON_BIN" -m pip install --user "pyinstaller==${PYINSTALLER_VERSION}" -i "$PIP_INDEX_URL"
fi

"$PYTHON_BIN" -m PyInstaller \
  --noconfirm \
  --clean \
  --onefile \
  --name dcgm_latency_predictor \
  --distpath "${ROOT_DIR}/bin" \
  --workpath "${ROOT_DIR}/build/pyinstaller" \
  --specpath "${ROOT_DIR}/build/pyinstaller" \
  --add-data "${ROOT_DIR}/pkg/python/results.csv:." \
  "${ROOT_DIR}/pkg/python/dcgm_latency_predictor.py"
