#!/usr/bin/env bash
set -euo pipefail

# Сборка Linux-бинарника для локальной проверки команды Nuitka.
# На Windows .exe собирается в GitHub Actions (.github/workflows/build-exe.yml).

cd "$(dirname "$0")/.."

if [ ! -x manta_cli ]; then
  echo "Нужен бинарник manta_cli рядом с проектом (cp ~/PycharmProjects/manta-cli/manta_cli ./manta_cli)" >&2
  exit 1
fi

python -m nuitka --mode=onefile --enable-plugin=pyqt6 \
  --assume-yes-for-downloads \
  --include-data-files=manta_cli=manta_cli \
  --include-data-dir=assets=assets \
  --output-filename=MVP_Calculator --output-dir=dist \
  mvp/__main__.py

echo "Готово: dist/MVP_Calculator"
