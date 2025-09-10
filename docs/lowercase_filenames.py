#!/usr/bin/env python3
"""
Convierte a minúsculas TODOS los nombres de archivos del repositorio (recursivo).
Opcionalmente, también puede convertir directorios.

¡ADVERTENCIA! Renombrar archivos puede romper imports, rutas y referencias.
Use --dry-run para previsualizar y verifique con git antes de confirmar.

Uso (Windows CMD):
  python lowercase_filenames.py --yes                 # renombra todos los archivos
  python lowercase_filenames.py --include-dirs --yes  # también directorios
  python lowercase_filenames.py --dry-run             # sólo vista previa

Opciones:
  --root <ruta>            Directorio raíz (default: carpeta del script)
  --include-dirs           Renombrar también directorios (desactivado por defecto)
  --exclude <patrones>     Directorios a excluir separados por ';' (default: .git;.qodo;__pycache__)
  --conflict <modo>        Qué hacer ante colisiones (skip|suffix|abort). Default: skip
  --yes                    No pedir confirmación (equivalente a AUTO_CONFIRM=1)
  --dry-run                Simulación, no cambia nada (equivalente a DRY_RUN=1)
  --verbose                Mostrar cada renombrado

Notas:
- En Windows (NTFS) si sólo cambia el case, se realiza un renombrado temporal intermedio
  para forzar el cambio de mayúsculas/minúsculas.
- Archivos/directorios con nombres reservados en Windows (CON, PRN, AUX, NUL, COM1..9, LPT1..9)
  se omiten por seguridad.
"""

import os
import sys
import argparse
import re
import uuid
from pathlib import Path
from datetime import datetime
from typing import List, Tuple

RESERVED_WIN = re.compile(r"^(con|prn|aux|nul|com[1-9]|lpt[1-9])(\..*)?$", re.IGNORECASE)


def is_reserved_windows(name: str) -> bool:
    return RESERVED_WIN.match(name) is not None


def scan_paths(root: Path, exclude_dirs: List[str]) -> Tuple[List[Path], List[Path]]:
    files: List[Path] = []
    dirs: List[Path] = []
    exclude_set = {e.strip().lower() for e in exclude_dirs if e.strip()}

    for current_root, dirnames, filenames in os.walk(root):
        # filtrar directorios a excluir (in-place)
        dirnames[:] = [d for d in dirnames if d.lower() not in exclude_set]

        # recolectar archivos
        for fname in filenames:
            files.append(Path(current_root) / fname)
        # recolectar directorios (renombrar después, de más profundo a más superficial)
        for d in dirnames:
            dirs.append(Path(current_root) / d)

    # ordenar dirs de mayor profundidad a menor para evitar conflictos
    dirs.sort(key=lambda p: len(p.parts), reverse=True)
    return files, dirs


def is_case_only_rename(src: Path, dest: Path) -> bool:
    try:
        # En Windows (case-insensitive), normcase igual y cadenas distintas => solo cambia el case
        return os.path.normcase(str(src)) == os.path.normcase(str(dest)) and str(src) != str(dest)
    except Exception:
        return False


def resolve_conflict(src: Path, dest: Path, mode: str) -> Path | None:
    # Caso especial: renombrado solo de mayúsculas/minúsculas
    if is_case_only_rename(src, dest):
        return dest

    if not dest.exists():
        return dest
    if mode == 'skip':
        return None
    if mode == 'abort':
        raise RuntimeError(f"Colisión: ya existe destino {dest}")
    # mode == 'suffix'
    parent = dest.parent
    stem = dest.stem
    suffix = dest.suffix
    i = 1
    while True:
        candidate = parent / f"{stem}_dup{i}{suffix}"
        if not candidate.exists():
            return candidate
        i += 1


def safe_rename(src: Path, dest: Path, dry_run: bool, verbose: bool) -> bool:
    # Si las cadenas completas (incluyendo case) son idénticas, no hay nada que hacer
    if str(src) == str(dest):
        return True

    # Si sólo cambia el case (Windows case-insensitive), hacemos rename intermedio
    if str(src).lower() == str(dest).lower() and str(src) != str(dest):
        tmp = dest.with_name(dest.name + f".__tmp_lower__{uuid.uuid4().hex[:8]}")
        if dry_run:
            if verbose:
                print(f"   (dry-run) {src} -> {dest} (via {tmp.name})")
            return True
        try:
            os.rename(src, tmp)
            os.rename(tmp, dest)
            if verbose:
                print(f"   {src} -> {dest}")
            return True
        except Exception as e:
            print(f"   ⚠️  Falló renombrado (case-only) {src} -> {dest}: {e}")
            # intentar limpiar temporal si quedó
            try:
                if tmp.exists():
                    os.rename(tmp, src)
            except Exception:
                pass
            return False

    # Renombre normal
    if dry_run:
        if verbose:
            print(f"   (dry-run) {src} -> {dest}")
        return True
    try:
        os.makedirs(dest.parent, exist_ok=True)
        os.rename(src, dest)
        if verbose:
            print(f"   {src} -> {dest}")
        return True
    except Exception as e:
        print(f"   ⚠️  Falló renombrado {src} -> {dest}: {e}")
        return False


def main():
    parser = argparse.ArgumentParser(description="Renombrar a minúsculas todos los archivos (y opcionalmente dirs)")
    parser.add_argument('--root', help='Raíz a procesar (default: carpeta del script)')
    parser.add_argument('--include-dirs', action='store_true', help='Renombrar también directorios')
    parser.add_argument('--exclude', default='.git;.qodo;__pycache__', help='Directorios a excluir separados por ;')
    parser.add_argument('--conflict', choices=['skip', 'suffix', 'abort'], default='skip', help='Acción ante colisiones')
    parser.add_argument('--yes', action='store_true', help='No pedir confirmación')
    parser.add_argument('--dry-run', action='store_true', help='Simulación, no cambia nada')
    parser.add_argument('--verbose', action='store_true', help='Mostrar cada renombrado')
    args = parser.parse_args()

    auto_confirm = args.yes or os.environ.get('AUTO_CONFIRM', '0').lower() in ['1', 'true', 'yes', 'y']
    dry_run = args.dry_run or os.environ.get('DRY_RUN', '0').lower() in ['1', 'true', 'yes', 'y']

    root = Path(args.root).resolve() if args.root else Path(__file__).resolve().parent
    if not root.exists() or not root.is_dir():
        print(f"❌ Raíz inválida: {root}")
        return 1

    print("🔤 RENOMBRADO A MINÚSCULAS")
    print("=" * 60)
    print(f"📁 Raíz: {root}")
    print(f"📅 Fecha: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"🔧 Modo: {'DRY-RUN' if dry_run else 'REAL'} | Con dirs: {'sí' if args.include_dirs else 'no'} | Conflictos: {args.conflict}")

    files, dirs = scan_paths(root, args.exclude.split(';'))

    # Construir plan de renombrado para archivos
    file_plan: List[Tuple[Path, Path]] = []
    conflicts = 0
    for f in files:
        new_name = f.name.lower()
        if new_name == f.name:
            continue  # ya está en minúsculas
        if is_reserved_windows(new_name):
            print(f"   ⏭️  Omitido (reservado Windows): {f}")
            continue
        dest = f.with_name(new_name)
        resolved = resolve_conflict(f, dest, args.conflict)
        if resolved is None:
            conflicts += 1
            print(f"   ⏭️  Colisión (skip): {f} -> {dest}")
            continue
        file_plan.append((f, resolved))

    # Plan para directorios (opcional)
    dir_plan: List[Tuple[Path, Path]] = []
    if args.include_dirs:
        for d in dirs:
            new_name = d.name.lower()
            if new_name == d.name:
                continue
            if is_reserved_windows(new_name):
                print(f"   ⏭️  Omitido dir (reservado Windows): {d}")
                continue
            dest = d.with_name(new_name)
            resolved = resolve_conflict(d, dest, args.conflict)
            if resolved is None:
                conflicts += 1
                print(f"   ⏭️  Colisión dir (skip): {d} -> {dest}")
                continue
            dir_plan.append((d, resolved))

    print(f"\n📊 Resumen plan:")
    print(f"   📄 Archivos por renombrar: {len(file_plan)} de {len(files)}")
    if args.include_dirs:
        print(f"   📁 Directorios por renombrar: {len(dir_plan)} de {len(dirs)}")
    if conflicts > 0:
        print(f"   ⚠️  Colisiones detectadas y saltadas: {conflicts}")

    if not auto_confirm:
        resp = input("\n❓ ¿Aplicar renombrados? (y/N): ").strip().lower()
        if resp not in ['y', 'yes', 's', 'si', 'sí']:
            print("❌ Operación cancelada por el usuario")
            return 1
    else:
        print("\n✅ Confirmación automática habilitada")

    # Ejecutar renombrado de archivos
    files_ok = 0
    for src, dest in file_plan:
        if safe_rename(src, dest, dry_run, args.verbose):
            files_ok += 1

    # Renombrar directorios (de más profundo a menos)
    dirs_ok = 0
    for src, dest in dir_plan:
        if safe_rename(src, dest, dry_run, args.verbose):
            dirs_ok += 1

    # Verificación post-rename
    unresolved = []
    if not dry_run:
        for src, dest in file_plan:
            try:
                # si el destino no existe o la ruta original sigue existiendo con el mismo nombre exacto
                if not dest.exists() or (src.exists() and src.name == src.name.upper() and dest.name != dest.name.lower()):
                    unresolved.append((src, dest))
            except Exception:
                pass

    print("\n✅ Finalizado")
    print("=" * 60)
    print(f"   📄 Archivos renombrados (operaciones OK): {files_ok}")
    if args.include_dirs:
        print(f"   📁 Directorios renombrados: {dirs_ok}")
    if unresolved:
        print(f"   ⚠️  Advertencia: {len(unresolved)} renombrado(s) no se reflejaron en el FS:")
        for s, d in unresolved[:10]:
            print(f"      - {s.name} -> {d.name} (verifique permisos/bloqueo)" )
        if len(unresolved) > 10:
            print(f"      ... y {len(unresolved)-10} más")
        print("   Sugerencias:")
        print("     • Cierre editores/procesos que tengan los archivos abiertos")
        print("     • Ejecute con --verbose para ver cada paso y reintente")
        print("     • Pruebe con --conflict suffix por si hay colisiones reales")
    return 0


if __name__ == '__main__':
    sys.exit(main())
