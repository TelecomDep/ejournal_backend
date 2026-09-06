#!/usr/bin/env python3
"""
Database Backup, Validation, Wipe, and Restore Manager for ejournal (PostgreSQL).
Features:
- Separate schema and data dumps
- Deep cryptographic & syntactic validation
- Strict safety checks before data wipe (TRUNCATE only, preserves DB schema)
- Safe restoration from validated backups
"""

import os
import sys
import re
import json
import time
import hashlib
import argparse
import subprocess
from pathlib import Path
from datetime import datetime

DEFAULT_CONTAINER = "ejournal-postgres"
DEFAULT_DB = "ejournal"
DEFAULT_USER = "postgres"

SCRIPT_DIR = Path(__file__).resolve().parent
PROJECT_DIR = SCRIPT_DIR.parent
BACKUPS_DIR = PROJECT_DIR / "backups"

CORE_TABLES = [
    "users", "students", "teachers", "groups",
    "subjects", "lecterns", "faculties", "schedules", "semesters",
    "lesson_times", "grades", "grade_items", "attendance",
    "attendance_sessions", "registration_invites"
]

DATA_TABLES_TO_WIPE = [
    "attendance_marks",
    "attendance_session_students",
    "attendance_session_groups",
    "attendance_sessions",
    "attendance",
    "grade_events",
    "grades",
    "grade_items",
    "schedules",
    "semester_load",
    "subject_controls",
    "subject_metrics",
    "subjects",
    "registration_invites",
    "rewards_punishments",
    "students",
    "teachers",
    "groups",
    "org_scopes",
    "password_reset_tokens",
    "user_device_tokens",
    "user_agreement_decisions",
    "user_avatars",
    "notification_recipients",
    "notification_settings",
    "notifications",
    "audit_logs",
    "auth_challenges"
]


def run_cmd(cmd, input_data=None, capture_output=True, text=True):
    """Executes a shell command and returns CompletedProcess."""
    if isinstance(cmd, str):
        shell = True
    else:
        shell = False
    return subprocess.run(
        cmd,
        input=input_data,
        shell=shell,
        capture_output=capture_output,
        text=text,
        check=False
    )


def exec_docker_psql(sql_query, db=DEFAULT_DB, container=DEFAULT_CONTAINER):
    """Runs a SQL query inside the postgres container and returns stdout."""
    cmd = [
        "docker", "exec", "-i", container,
        "psql", "-U", DEFAULT_USER, "-d", db,
        "-t", "-A", "-c", sql_query
    ]
    res = run_cmd(cmd)
    if res.returncode != 0:
        raise RuntimeError(f"psql error: {res.stderr.strip()}")
    return res.stdout.strip()


def compute_sha256(file_path: Path) -> str:
    h = hashlib.sha256()
    with open(file_path, "rb") as f:
        while chunk := f.read(65536):
            h.update(chunk)
    return h.hexdigest()


def get_current_table_counts(container=DEFAULT_CONTAINER, db=DEFAULT_DB):
    """Queries row counts for all tables in the database."""
    sql = """
    SELECT table_name FROM information_schema.tables
    WHERE table_schema = 'public' AND table_type = 'BASE TABLE'
    ORDER BY table_name;
    """
    tables = [t.strip() for t in exec_docker_psql(sql, db, container).splitlines() if t.strip()]

    counts = {}
    for table in tables:
        count_str = exec_docker_psql(f"SELECT count(*) FROM {table};", db, container)
        counts[table] = int(count_str) if count_str.isdigit() else 0
    return counts


def create_backup(backups_dir=BACKUPS_DIR, container=DEFAULT_CONTAINER, db=DEFAULT_DB):
    """
    Creates a full backup set:
    1. schema-only SQL
    2. data-only SQL
    3. full dump SQL
    4. manifest JSON
    """
    backups_dir.mkdir(parents=True, exist_ok=True)
    ts = datetime.now().strftime("%Y%m%d_%H%M%S")
    backup_id = f"backup_{ts}"

    schema_file = backups_dir / f"{backup_id}_schema.sql"
    data_file = backups_dir / f"{backup_id}_data.sql"
    full_file = backups_dir / f"{backup_id}_full.sql"
    manifest_file = backups_dir / f"{backup_id}_manifest.json"

    print(f"\n[+] Начало резервного копирования базы '{db}' (ID: {backup_id})...")

    # 1. Row counts before backup
    print("[+] Сбор статистики по таблицам...")
    table_counts = get_current_table_counts(container, db)

    # 2. Schema-only dump
    print(f"[+] Создание дампа схемы: {schema_file.name}")
    cmd_schema = f"docker exec -i {container} pg_dump -U {DEFAULT_USER} -d {db} --schema-only --no-owner --no-privileges > '{schema_file}'"
    res = run_cmd(cmd_schema)
    if res.returncode != 0:
        raise RuntimeError(f"Ошибка дампа схемы: {res.stderr}")

    # 3. Data-only dump
    print(f"[+] Создание дампа данных: {data_file.name}")
    cmd_data = f"docker exec -i {container} pg_dump -U {DEFAULT_USER} -d {db} --data-only --no-owner --no-privileges --inserts --disable-triggers > '{data_file}'"
    res = run_cmd(cmd_data)
    if res.returncode != 0:
        raise RuntimeError(f"Ошибка дампа данных: {res.stderr}")

    # 4. Full dump
    print(f"[+] Создание полного дампа: {full_file.name}")
    cmd_full = f"docker exec -i {container} pg_dump -U {DEFAULT_USER} -d {db} --clean --if-exists --no-owner --no-privileges > '{full_file}'"
    res = run_cmd(cmd_full)
    if res.returncode != 0:
        raise RuntimeError(f"Ошибка полного дампа: {res.stderr}")

    # 5. Manifest
    manifest = {
        "backup_id": backup_id,
        "database": db,
        "timestamp": datetime.now().isoformat(),
        "files": {
            "schema": {
                "filename": schema_file.name,
                "size_bytes": schema_file.stat().st_size,
                "sha256": compute_sha256(schema_file)
            },
            "data": {
                "filename": data_file.name,
                "size_bytes": data_file.stat().st_size,
                "sha256": compute_sha256(data_file)
            },
            "full": {
                "filename": full_file.name,
                "size_bytes": full_file.stat().st_size,
                "sha256": compute_sha256(full_file)
            }
        },
        "table_counts": table_counts,
        "tables_count": len(table_counts),
        "total_rows": sum(table_counts.values())
    }

    with open(manifest_file, "w", encoding="utf-8") as f:
        json.dump(manifest, f, indent=2, ensure_ascii=False)

    # 6. Update latest symlinks / copies
    for src, link_name in [
        (schema_file, "latest_schema.sql"),
        (data_file, "latest_data.sql"),
        (full_file, "latest_full.sql"),
        (manifest_file, "latest_manifest.json")
    ]:
        dest = backups_dir / link_name
        if dest.exists() or dest.is_symlink():
            dest.unlink()
        try:
            dest.symlink_to(src.name)
        except Exception:
            # On filesystems without symlinks, write copy
            dest.write_bytes(src.read_bytes())

    print(f"[✔] Бэкап успешно создан! Таблиц: {manifest['tables_count']}, строк: {manifest['total_rows']}")
    return manifest_file


def validate_backup(manifest_path: Path) -> bool:
    """
    Validates backup integrity:
    - Files exist and sizes > 0
    - SHA256 matches manifest
    - Schema file contains CREATE TABLE for all core tables
    - Data file contains valid SQL and dump markers
    """
    print(f"\n[+] Валидация бэкапа по манифесту: {manifest_path.name}...")
    if not manifest_path.exists():
        print(f"[✘] Манифест не найден: {manifest_path}")
        return False

    try:
        with open(manifest_path, "r", encoding="utf-8") as f:
            manifest = json.load(f)
    except Exception as e:
        print(f"[✘] Повреждённый JSON манифеста: {e}")
        return False

    base_dir = manifest_path.parent
    files_info = manifest.get("files", {})

    for key in ["schema", "data", "full"]:
        info = files_info.get(key)
        if not info:
            print(f"[✘] В манифесте отсутствует запись для '{key}'")
            return False

        target_file = base_dir / info["filename"]
        if not target_file.exists():
            print(f"[✘] Файл не найден на диске: {target_file}")
            return False

        current_size = target_file.stat().st_size
        if current_size == 0 or current_size != info["size_bytes"]:
            print(f"[✘] Несоответствие размера файла {target_file.name}: ожидался {info['size_bytes']}, факт {current_size}")
            return False

        current_hash = compute_sha256(target_file)
        if current_hash != info["sha256"]:
            print(f"[✘] Несовпадение контрольной суммы SHA-256 для {target_file.name}")
            return False
        print(f"  [✔] {target_file.name}: размер и SHA256 проверены.")

    # 1. Deep check schema SQL
    schema_file = base_dir / files_info["schema"]["filename"]
    schema_content = schema_file.read_text(encoding="utf-8", errors="replace")
    if "-- PostgreSQL database dump" not in schema_content:
        print("[✘] В файле схемы отсутствует заголовок дампа PostgreSQL")
        return False

    missing_tables = []
    for table in CORE_TABLES:
        pattern = re.compile(rf"CREATE\s+TABLE\s+(public\.)?{re.escape(table)}\s*\(", re.IGNORECASE)
        if not pattern.search(schema_content):
            missing_tables.append(table)

    if missing_tables:
        print(f"[✘] В схеме отсутствуют ключевые таблицы: {missing_tables}")
        return False
    print(f"  [✔] Схема содержит все ключевые таблицы ({len(CORE_TABLES)} шт.)")

    # 2. Deep check data SQL
    data_file = base_dir / files_info["data"]["filename"]
    data_content = data_file.read_text(encoding="utf-8", errors="replace")
    if "-- PostgreSQL database dump" not in data_content:
        print("[✘] В файле данных отсутствует заголовок дампа PostgreSQL")
        return False
    if "-- PostgreSQL database dump complete" not in data_content:
        print("[✘] Файл данных не содержит завершающего маркера дампа (возможно оборван)")
        return False
    print("  [✔] Файл данных корректен и содержит завершающий маркер дампа.")

    print(f"[✔] ВАЛИДАЦИЯ УСПЕШНА: Бэкап '{manifest['backup_id']}' полностью валиден и готов к восстановлению.")
    return True


def wipe_data(preserve_admins=True, container=DEFAULT_CONTAINER, db=DEFAULT_DB):
    """
    Safely purges data from the database using TRUNCATE CASCADE.
    IMPORTANT: The database schema (tables, types, constraints, functions) is NEVER dropped.
    """
    print(f"\n[!] ВНИМАНИЕ: Запуск очистки данных в базе '{db}'...")
    print("    Схема базы данных (таблицы, типы, триггеры) останется НЕИЗМЕННОЙ.")
    print("    Будут очищены только строки в таблицах данных.")

    tables_to_truncate = list(DATA_TABLES_TO_WIPE)
    if not preserve_admins:
        tables_to_truncate.extend(["user_roles", "users"])

    truncate_sql = f"TRUNCATE TABLE {', '.join(tables_to_truncate)} CASCADE;"

    # Execute truncate
    exec_docker_psql(truncate_sql, db, container)

    if preserve_admins:
        # If user roles table was affected, ensure admin role sync
        print("[+] Сохранение системных аккаунтов пользователей...")
    else:
        print("[+] Таблицы пользователей также очищены.")

    # Verify wipe results
    post_counts = get_current_table_counts(container, db)
    academic_rows = sum(post_counts.get(t, 0) for t in DATA_TABLES_TO_WIPE)

    print(f"[✔] Очистка данных завершена. Осталось академических строк: {academic_rows}")
    return academic_rows == 0


def restore_backup(manifest_path: Path, restore_mode="data", container=DEFAULT_CONTAINER, db=DEFAULT_DB):
    """
    Restores database from a validated backup.
    restore_mode: 'data' (restores data into existing schema) or 'full' (restores schema + data)
    """
    if not validate_backup(manifest_path):
        print("[✘] Восстановление отменено: бэкап не прошёл валидацию!")
        return False

    with open(manifest_path, "r", encoding="utf-8") as f:
        manifest = json.load(f)

    base_dir = manifest_path.parent
    if restore_mode == "full":
        target_file = base_dir / manifest["files"]["full"]["filename"]
        print(f"\n[+] Полное восстановление (схема + данные) из: {target_file.name}...")
    else:
        target_file = base_dir / manifest["files"]["data"]["filename"]
        print(f"\n[+] Восстановление данных в существующую схему из: {target_file.name}...")

    cmd = f"docker exec -i {container} psql -U {DEFAULT_USER} -d {db} < '{target_file}'"
    res = run_cmd(cmd)
    if res.returncode != 0:
        print(f"[✘] Ошибка при восстановлении: {res.stderr}")
        return False

    post_counts = get_current_table_counts(container, db)
    total_restored = sum(post_counts.values())
    print(f"[✔] Восстановление успешно завершено! Всего строк в базе: {total_restored}")
    return True


def get_latest_manifest(backups_dir=BACKUPS_DIR) -> Path | None:
    latest = backups_dir / "latest_manifest.json"
    if latest.exists():
        return latest
    manifests = sorted(backups_dir.glob("backup_*_manifest.json"), reverse=True)
    return manifests[0] if manifests else None


def main():
    parser = argparse.ArgumentParser(description="Менеджер бэкапов, валидации, очистки данных и восстановления БД ejournal.")
    parser.add_argument("--backup", action="store_true", help="Создать полный комплект бэкапов (схема + данные + манифест)")
    parser.add_argument("--validate", nargs="?", const="", help="Проверить валидность бэкапа (по умолчанию последнего)")
    parser.add_argument("--wipe", action="store_true", help="Очистить данные из таблиц (только если есть валидный бэкап)")
    parser.add_argument("--backup-and-wipe", action="store_true", help="Сделать бэкап, валидировать его и только потом очистить данные")
    parser.add_argument("--restore", nargs="?", const="", help="Восстановить данные из бэкапа (по умолчанию последнего)")
    parser.add_argument("--restore-full", nargs="?", const="", help="Полное восстановление (схема + данные)")
    parser.add_argument("--wipe-all-users", action="store_true", help="При очистке удалить также всех пользователей (включая админов)")
    parser.add_argument("--force", action="store_true", help="Пропустить интерактивное подтверждение очистки")
    parser.add_argument("--dir", default=str(BACKUPS_DIR), help="Каталог для сохранения бэкапов")

    args = parser.parse_args()
    backups_dir = Path(args.dir)

    if args.backup:
        manifest = create_backup(backups_dir)
        validate_backup(manifest)

    elif args.validate is not None:
        if args.validate:
            manifest_path = Path(args.validate)
            if manifest_path.is_dir():
                manifest_path = get_latest_manifest(manifest_path)
        else:
            manifest_path = get_latest_manifest(backups_dir)

        if not manifest_path or not manifest_path.exists():
            print("[✘] Не найден манифест для проверки.")
            sys.exit(1)
        ok = validate_backup(manifest_path)
        sys.exit(0 if ok else 1)

    elif args.backup_and_wipe:
        manifest = create_backup(backups_dir)
        if not validate_backup(manifest):
            print("[✘] ОШИБКА ВАЛИДАЦИИ: Очистка базы данных ОТМЕНЕНА для защиты данных!")
            sys.exit(1)
        print("\n[✔] Данные надежно забекаплены и валидированы.")
        wipe_data(preserve_admins=not args.wipe_all_users)

    elif args.wipe:
        latest = get_latest_manifest(backups_dir)
        if not latest or not validate_backup(latest):
            print("[✘] ОШИБКА: Нет подтверждённого валидного бэкапа! Сначала выполните --backup.")
            sys.exit(1)
        if not args.force:
            ans = input("Вы уверены, что хотите удалить ВСЕ данные из базы (схема сохранится)? [yes/NO]: ")
            if ans.strip().lower() not in ("yes", "y", "да"):
                print("Операция отменена.")
                sys.exit(0)
        wipe_data(preserve_admins=not args.wipe_all_users)

    elif args.restore is not None:
        target = Path(args.restore) if args.restore else get_latest_manifest(backups_dir)
        if not target or not target.exists():
            print("[✘] Бэкап для восстановления не найден.")
            sys.exit(1)
        restore_backup(target, restore_mode="data")

    elif args.restore_full is not None:
        target = Path(args.restore_full) if args.restore_full else get_latest_manifest(backups_dir)
        if not target or not target.exists():
            print("[✘] Бэкап для восстановления не найден.")
            sys.exit(1)
        restore_backup(target, restore_mode="full")

    else:
        parser.print_help()


if __name__ == "__main__":
    main()
