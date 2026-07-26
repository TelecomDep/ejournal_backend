#!/usr/bin/env python3
"""
Azimutt User Management CLI Script.
Allows administrators to create, list, and delete Azimutt users securely.
Self-registration on the web interface is blocked by a database trigger.
Only users created by this script can log in.
"""

import sys
import os
import uuid
import argparse
import subprocess
import json
import re

def run_cmd(cmd_args):
    """Executes a command and returns the stdout string."""
    res = subprocess.run(cmd_args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)
    if res.returncode != 0:
        raise RuntimeError(f"Command failed: {' '.join(cmd_args)}\nError: {res.stderr}")
    return res.stdout.strip()

def generate_bcrypt_hash(password: str) -> str:
    """Generates a Bcrypt hash using Azimutt's Elixir runtime container."""
    elixir_code = f'IO.puts(Bcrypt.hash_pwd_salt("{password}"))'
    cmd = [
        "docker", "compose", "exec", "-T", "azimutt",
        "/app/bin/azimutt", "eval", elixir_code
    ]
    output = run_cmd(cmd)
    # Filter log output lines to extract the $2b$... bcrypt string
    for line in output.splitlines():
        line = line.strip()
        if line.startswith("$2b$") or line.startswith("$2a$"):
            return line
    raise RuntimeError(f"Failed to extract Bcrypt hash from output: {output}")

def execute_sql(sql: str) -> str:
    """Executes SQL against azimutt_db inside the postgres container."""
    cmd = [
        "docker", "compose", "exec", "-T", "postgres",
        "psql", "-U", "postgres", "-d", "azimutt_db", "-c", sql
    ]
    return run_cmd(cmd)

def create_user(email: str, password: str, name: str):
    email = email.lower().strip()
    name = name.strip()
    slug = re.sub(r'[^a-z0-9]+', '-', email.split('@')[0]) + '-' + str(uuid.uuid4())[:4]
    user_id = str(uuid.uuid4())
    avatar_url = f"https://robohash.org/{slug}"
    
    print(f"🔐 Hashing password for {email}...")
    hashed_password = generate_bcrypt_hash(password)
    
    data_json = json.dumps({"created_by_admin": "true"})
    
    sql_insert_user = f"""
    INSERT INTO users (
        id, slug, name, email, avatar, is_admin, hashed_password,
        last_signin, created_at, updated_at, data
    ) VALUES (
        '{user_id}', '{slug}', '{name}', '{email}', '{avatar_url}', false, '{hashed_password}',
        NOW(), NOW(), NOW(), '{data_json}'::jsonb
    );
    """
    
    print(f"👤 Creating user entry in azimutt_db...")
    execute_sql(sql_insert_user)
    
    # Get organization ID for Sibsutis or primary org
    get_org_sql = "SELECT id FROM organizations WHERE name = 'Sibsutis' LIMIT 1;"
    org_output = execute_sql(get_org_sql)
    org_ids = [line.strip() for line in org_output.splitlines() if len(line.strip()) == 36 and '-' in line.strip()]
    if not org_ids:
        get_org_sql = "SELECT id FROM organizations ORDER BY created_at ASC LIMIT 1;"
        org_output = execute_sql(get_org_sql)
        org_ids = [line.strip() for line in org_output.splitlines() if len(line.strip()) == 36 and '-' in line.strip()]
    
    if org_ids:
        org_id = org_ids[0]
        sql_insert_member = f"""
        INSERT INTO organization_members (user_id, organization_id, created_by, updated_by, created_at, updated_at, role)
        VALUES ('{user_id}', '{org_id}', '{user_id}', '{user_id}', NOW(), NOW(), 'writer')
        ON CONFLICT (user_id, organization_id) DO NOTHING;
        """
        print(f"🏢 Adding user to organization ({org_id})...")
        execute_sql(sql_insert_member)
    
    print(f"\n✅ User successfully created!")
    print(f"----------------------------------------")
    print(f"  Name:     {name}")
    print(f"  Email:    {email}")
    print(f"  Password: {password}")
    print(f"  ID:       {user_id}")
    print(f"----------------------------------------")
    print(f"User can now log in at http://localhost:4000 (or your server URL).\n")

def list_users():
    sql = """
    SELECT u.name, u.email, u.is_admin, o.name as org_name, om.role, u.created_at
    FROM users u
    LEFT JOIN organization_members om ON u.id = om.user_id
    LEFT JOIN organizations o ON om.organization_id = o.id
    ORDER BY u.created_at DESC;
    """
    print(execute_sql(sql))

def delete_user(email: str):
    email = email.lower().strip()
    sql = f"""
    DELETE FROM organization_members WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
    DELETE FROM users WHERE email = '{email}';
    """
    res = execute_sql(sql)
    print(f"🗑️ Deleted user {email}: {res}")

def main():
    parser = argparse.ArgumentParser(description="Azimutt User Management CLI")
    subparsers = parser.add_subparsers(dest="command", required=True)
    
    # Create subcommand
    create_parser = subparsers.add_parser("create", help="Create a new Azimutt user")
    create_parser.add_argument("--email", required=True, help="User email address")
    create_parser.add_argument("--password", required=True, help="User password")
    create_parser.add_argument("--name", required=True, help="User full name")
    
    # List subcommand
    subparsers.add_parser("list", help="List all Azimutt users")
    
    # Delete subcommand
    delete_parser = subparsers.add_parser("delete", help="Delete a user by email")
    delete_parser.add_argument("--email", required=True, help="User email address")
    
    args = parser.parse_args()
    
    try:
        if args.command == "create":
            create_user(args.email, args.password, args.name)
        elif args.command == "list":
            list_users()
        elif args.command == "delete":
            delete_user(args.email)
    except Exception as e:
        print(f"❌ Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
