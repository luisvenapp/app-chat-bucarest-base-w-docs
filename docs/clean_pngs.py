#!/usr/bin/env python3
"""
Limpia (elimina) TODOS los archivos PNG del repositorio de forma recursiva.

Uso básico (Windows CMD):
  python clean_pngs.py

Opciones:
  --root <ruta>        Directorio raíz a limpiar (por defecto, el directorio del script)
  --yes                No pedir confirmación (equivalente a AUTO_CONFIRM=1)
  --dry-run            Solo mostrar qué se eliminaría, sin borrar
  --verbose            Mostrar cada archivo eliminado

Variables de entorno:
  AUTO_CONFIRM=1       No pedir confirmación
  DRY_RUN=1            Solo mostrar, no borrar

Notas:
- El script elimina todo archivo con extensión .png (insensible a mayúsculas/minúsculas)
- Úsalo con precaución (no hay papelera). Haz un backup si es necesario
"""

import os
import sys
import argparse
from pathlib import Path
from datetime import datetime


def human_size(num_bytes: int) -> str:
    if num_bytes < 1024:
        return f"{num_bytes} B"
    if num_bytes < 1024 * 1024:
        return f"{num_bytes / 1024:.1f} KB"
    if num_bytes < 1024 * 1024 * 1024:
        return f"{num_bytes / (1024 * 1024):.1f} MB"
    return f"{num_bytes / (1024 * 1024 * 1024):.1f} GB"


def find_png_files(root: Path) -> list[Path]:
    files = []
    for p in root.rglob('*'):
        try:
            if p.is_file() and p.suffix.lower() == '.png':
                files.append(p)
        except PermissionError:
            # Saltar rutas sin permiso
            continue
    return files


def main():
    parser = argparse.ArgumentParser(description="Eliminar todos los PNG del repositorio (recursivo)")
    parser.add_argument('--root', help='Directorio raíz a limpiar (default: directorio del script)')
    parser.add_argument('--yes', action='store_true', help='No pedir confirmación')
    parser.add_argument('--dry-run', action='store_true', help='Solo mostrar qué se eliminaría')
    parser.add_argument('--verbose', action='store_true', help='Mostrar cada archivo eliminado')
    args = parser.parse_args()

    auto_confirm = args.yes or os.environ.get('AUTO_CONFIRM', '0').lower() in ['1', 'true', 'yes', 'y']
    dry_run = args.dry_run or os.environ.get('DRY_RUN', '0').lower() in ['1', 'true', 'yes', 'y']

    root = Path(args.root).resolve() if args.root else Path(__file__).resolve().parent

    if not root.exists() or not root.is_dir():
        print(f"❌ Raíz inválida: {root}")
        return 1

    print("🧹 LIMPIEZA DE PNGS")
    print("=" * 50)
    print(f"📁 Raíz: {root}")
    print(f"📅 Fecha: {datetime.now().strftime('%Y-%m-%d %H:%M:%S')}")
    print(f"🔧 Modo: {'DRY-RUN' if dry_run else 'ELIMINACIÓN REAL'}")

    png_files = find_png_files(root)

    if not png_files:
        print("✅ No se encontraron archivos .png")
        return 0

    total_size = sum(f.stat().st_size for f in png_files if f.exists())
    print(f"\n📊 Resumen previo:")
    print(f"   🖼️  Archivos PNG encontrados: {len(png_files)}")
    print(f"   💾 Tamaño total: {human_size(total_size)}")

    if not auto_confirm:
        resp = input("\n❓ ¿Eliminar TODOS los PNG encontrados? (y/N): ").strip().lower()
        if resp not in ['y', 'yes', 's', 'si', 'sí']:
            print("❌ Operación cancelada por el usuario")
            return 1
    else:
        print("\n✅ Confirmación automática habilitada")

    removed = 0
    errors = 0

    for f in png_files:
        try:
            if dry_run:
                if args.verbose:
                    print(f"   🗑️  (dry-run) {f}")
                continue
            if args.verbose:
                print(f"   🗑️  Eliminado: {f}")
            os.remove(f)
            removed += 1
        except Exception as e:
            errors += 1
            print(f"   ⚠️  No se pudo eliminar {f}: {e}")

    print("\n✅ Limpieza finalizada")
    print("=" * 50)
    print(f"🗑️  Eliminados: {removed}")
    print(f"⚠️  Errores: {errors}")

    return 0 if errors == 0 else 2


if __name__ == '__main__':
    sys.exit(main())
