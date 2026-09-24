"""Initialize/start this project's loopback-only PostgreSQL cluster."""
import os
from pathlib import Path
import socket
import subprocess
import tempfile

import psycopg2
from psycopg2 import sql
from dotenv import dotenv_values

ROOT = Path(__file__).resolve().parents[1]
CFG = dotenv_values(ROOT / 'backend' / '.env')
DATA = ROOT / 'postgres-data'
LOGS = ROOT / 'runtime-logs'
LOGS.mkdir(exist_ok=True)


def pg_bin():
    override = os.environ.get('POSTGRES_BIN')
    if override:
        return Path(override)
    versions = Path(os.environ.get('ProgramFiles', r'C:\Program Files')) / 'PostgreSQL'
    candidates = sorted(versions.glob('*/bin'), key=lambda p: int(p.parent.name.split('.')[0]), reverse=True)
    for candidate in candidates:
        if (candidate / 'pg_ctl.exe').exists():
            return candidate
    raise RuntimeError('Install PostgreSQL or set POSTGRES_BIN to its bin directory')


def command(*args, check=True):
    return subprocess.run([str(a) for a in args], check=check,
                          creationflags=subprocess.CREATE_NO_WINDOW if os.name == 'nt' else 0)


def ensure_database():
    binaries = pg_bin()
    port = int(CFG['DB_PORT'])
    running = DATA.exists() and command(binaries / 'pg_ctl.exe', '-D', DATA, 'status', check=False).returncode == 0
    if not running:
        with socket.socket() as sock:
            if sock.connect_ex(('127.0.0.1', port)) == 0:
                raise RuntimeError(f'Port {port} is occupied by another server; refusing to alter it')
        if not (DATA / 'PG_VERSION').exists():
            with tempfile.NamedTemporaryFile(mode='w', encoding='utf-8', dir=LOGS, delete=False) as pw:
                pw.write(CFG['DB_PASSWORD'] + '\n')
                pwpath = Path(pw.name)
            try:
                command(binaries / 'initdb.exe', '-D', DATA, '-U', CFG['DB_USERNAME'],
                        '--auth=scram-sha-256', '--encoding=UTF8', '--locale=C', f'--pwfile={pwpath}')
                with (DATA / 'postgresql.conf').open('a', encoding='utf-8') as f:
                    f.write(f"\nlisten_addresses = '127.0.0.1'\nport = {port}\n")
            finally:
                pwpath.unlink(missing_ok=True)
        command(binaries / 'pg_ctl.exe', '-D', DATA, '-l', LOGS / 'postgres.log', '-w', 'start')
    conn = psycopg2.connect(host=CFG['DB_HOST'], port=port, user=CFG['DB_USERNAME'],
                            password=CFG['DB_PASSWORD'], dbname='postgres', connect_timeout=5)
    conn.autocommit = True
    try:
        with conn.cursor() as cursor:
            cursor.execute('SELECT 1 FROM pg_database WHERE datname = %s', (CFG['DB_DATABASE'],))
            if cursor.fetchone() is None:
                cursor.execute(sql.SQL('CREATE DATABASE {}').format(sql.Identifier(CFG['DB_DATABASE'])))
    finally:
        conn.close()
    print(f"Database ready: {CFG['DB_HOST']}:{port}/{CFG['DB_DATABASE']}")


if __name__ == '__main__':
    import sys
    if '--stop' in sys.argv:
        if (DATA / 'PG_VERSION').exists():
            command(pg_bin() / 'pg_ctl.exe', '-D', DATA, '-m', 'fast', '-w', 'stop')
    else:
        ensure_database()
