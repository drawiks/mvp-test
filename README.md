# MVP Calculator

GUI-программа для Dota 2: разбирает реплей (`.dem` / `.dem.bz2` / `.dem.zst`) через
[`manta_cli`](https://github.com/drawiks/manta-cli) и высчитывает MVP по формуле.

Одновременно показываются **три** игрока:

- 🥇 лучший игрок победившей команды;
- 🥈 второй лучший игрок победившей команды;
- 🥉 лучший игрок проигравшей команды.

## Формула

```
score = Kills*0.3
      + (3.0 - Deaths*0.3)
      + Assists*0.15
      + LastHits*0.003
      + GPM*0.002
      + XPM*0.002
      + StunDuration*0.05
      + Healing*0.004
      + TowerDamage*0.001
      + CampsStacked*0.5
      + RunePickups*0.2
      + FirstBlood*1.0
```

Все коэффициенты редактируются в панели справа **на лету** (для тестов на реальных
матчах): результат пересчитывается мгновенно на уже загруженном реплее, повторно
запускать разбор не нужно. Значения сохраняются между запусками (`QSettings`).

## Локальный запуск (Linux/dev)

```sh
python3 -m venv .venv
.venv/bin/pip install -r requirements-dev.txt
.venv/bin/python -m mvp
```

Можно сразу открыть реплей: `.venv/bin/python -m mvp /path/to/match.dem`.

Приложение ищет бинарник `manta_cli` в таком порядке:

1. путь, указанный вручную в *Настройки → Путь к manta_cli…*;
2. рядом с запускаемым приложением (для собранного `.exe` — рядом с ним);
3. в `PATH`.

## Тесты

```sh
.venv/bin/python -m pytest -q
```

Проверяется формула на эталонном реплее (`tests/fixtures/match.json`), выбор трёх
MVP, разбор JSON и работа кастомных коэффициентов.

## Сборка `.exe` через Nuitka

`.exe` собирается автоматически в **GitHub Actions** (вариант без установки
Windows): GitHub запускает Windows-раннер, ставит Python 3.12 + PyQt6 + Nuitka,
собирает `manta_cli.exe` из Go-исходников и упаковывает всё в один файл.

Workflow: [`.github/workflows/build-exe.yml`](.github/workflows/build-exe.yml).
Триггеры: push в `main` или запуск вручную (Actions → Run workflow).

Шаги:

1. Создай репозиторий на GitHub и подключи его:

   ```sh
   git remote add origin https://github.com/<you>/mvp-test.git
   git branch -M main
   git push -u origin main
   ```

2. Открой **Actions → Build Windows EXE → Run workflow** (или сделай push).
3. В конце джобы скачай артефакт `MVP_Calculator-windows` — это и есть готовый
   `MVP_Calculator.exe` (внутри уже лежит `manta_cli.exe`).

### Сборка вручную (Linux, для проверки команды)

```sh
pip install nuitka zstandard
# нужен patchelf (0.17.x), например: pacman -S patchelf / apt install patchelf
cp /path/to/manta_cli ./manta_cli
python -m nuitka --mode=onefile --enable-plugin=pyqt6 \
  --include-data-files=manta_cli=manta_cli \
  --output-filename=MVP_Calculator --output-dir=dist \
  mvp/__main__.py
```

### Заметки

- `--onefile` даёт один исполняемый файл; антивирусы иногда ложно-положительно
  реагируют на self-extracting исполняемые файлы. Если это мешает — соберите с
  `--mode=standalone` (папка `dist/MVP_Calculator.dist/`) и раздавайте папкой.
- MinGW64-компилятор Nuitka не работает с Python 3.13+; в CI используется
  Python 3.12 + MSVC (предустановлен на `windows-latest`).
