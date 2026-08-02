#!/usr/bin/env python3
"""
Azimutt User Management CLI Script.
Allows administrators to create, list, delete, repair, and reset passwords for Azimutt users.
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

def check_user_exists(email: str) -> bool:
    sql = f"SELECT id FROM users WHERE email = '{email}';"
    output = execute_sql(sql)
    return any(len(line.strip()) == 36 and '-' in line.strip() for line in output.splitlines())

def repair_user_memberships():
    """Ensures all existing users belong to at least one organization with owner/writer role and valid enterprise plan fields."""
    sql = """
    UPDATE organizations 
    SET plan = 'enterprise', 
        plan_status = 'active', 
        plan_seats = 100, 
        plan_validated = NOW()
    WHERE plan_validated IS NULL OR plan_status IS DISTINCT FROM 'active';

    INSERT INTO organizations (id, slug, name, logo, is_personal, created_by, updated_by, created_at, updated_at, plan, plan_status, plan_seats, plan_validated)
    SELECT gen_random_uuid(), 'sibsutis', 'Sibsutis', 'https://robohash.org/sibsutis', false, u.id, u.id, NOW(), NOW(), 'enterprise', 'active', 100, NOW()
    FROM users u
    WHERE NOT EXISTS (SELECT 1 FROM organizations)
    LIMIT 1;

    INSERT INTO organization_members (user_id, organization_id, created_by, updated_by, created_at, updated_at, role)
    SELECT u.id, o.id, u.id, u.id, NOW(), NOW(), 'owner'
    FROM users u
    CROSS JOIN (SELECT id FROM organizations ORDER BY created_at ASC LIMIT 1) o
    ON CONFLICT (user_id, organization_id) DO NOTHING;
    """
    execute_sql(sql)

def create_user(email: str, password: str, name: str):
    email = email.lower().strip()
    name = name.strip()
    
    if check_user_exists(email):
        print(f"⚠️ User with email '{email}' already exists!")
        print(f"🔄 Updating password and repairing access for '{email}'...")
        reset_password(email, password)
        return

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
    repair_user_memberships()
    
    print(f"\n✅ User successfully created!")
    print(f"----------------------------------------")
    print(f"  Name:     {name}")
    print(f"  Email:    {email}")
    print(f"  Password: {password}")
    print(f"  ID:       {user_id}")
    print(f"----------------------------------------")
    print(f"User can now log in at http://localhost:4000 (or your server URL).\n")

def reset_password(email: str, password: str):
    email = email.lower().strip()
    if not check_user_exists(email):
        print(f"❌ User with email '{email}' does not exist.", file=sys.stderr)
        sys.exit(1)

    print(f"🔐 Hashing new password for {email}...")
    hashed_password = generate_bcrypt_hash(password)
    sql = f"UPDATE users SET hashed_password = '{hashed_password}', updated_at = NOW() WHERE email = '{email}';"
    execute_sql(sql)
    repair_user_memberships()
    print(f"🔑 Password and organization membership repaired successfully for {email}")

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
    if not check_user_exists(email):
        print(f"❌ User with email '{email}' does not exist.", file=sys.stderr)
        sys.exit(1)

    sql_count = f"SELECT count(*) FROM users WHERE email != '{email}';"
    count_output = execute_sql(sql_count)
    other_count = 0
    for line in count_output.splitlines():
        line_str = line.strip()
        if line_str.isdigit():
            other_count = int(line_str)
            break

    if other_count == 0:
        sql = f"""
        DELETE FROM events WHERE created_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_tokens WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_profiles WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_auth_tokens WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM project_tokens WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR revoked_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM projects WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}') OR local_owner IN (SELECT id FROM users WHERE email = '{email}') OR archived_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM organization_invitations WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR answered_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM organization_members WHERE user_id IN (SELECT id FROM users WHERE email = '{email}') OR created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM organizations WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}') OR deleted_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM users WHERE email = '{email}';
        """
    else:
        sql = f"""
        UPDATE organizations SET created_by = (SELECT id FROM users WHERE email != '{email}' LIMIT 1), updated_by = (SELECT id FROM users WHERE email != '{email}' LIMIT 1) WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}');
        UPDATE projects SET created_by = (SELECT id FROM users WHERE email != '{email}' LIMIT 1), updated_by = (SELECT id FROM users WHERE email != '{email}' LIMIT 1), local_owner = (SELECT id FROM users WHERE email != '{email}' LIMIT 1) WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}') OR local_owner IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM events WHERE created_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_tokens WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_profiles WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM user_auth_tokens WHERE user_id IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM project_tokens WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR revoked_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM organization_invitations WHERE created_by IN (SELECT id FROM users WHERE email = '{email}') OR answered_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM organization_members WHERE user_id IN (SELECT id FROM users WHERE email = '{email}') OR created_by IN (SELECT id FROM users WHERE email = '{email}') OR updated_by IN (SELECT id FROM users WHERE email = '{email}');
        DELETE FROM users WHERE email = '{email}';
        """
    res = execute_sql(sql)
    print(f"🗑️ Deleted user {email}: {res}")

def repair():
    repair_user_memberships()
    print("🛠️ All user organization memberships repaired successfully!")

def main():
    parser = argparse.ArgumentParser(description="Azimutt User Management CLI")
    subparsers = parser.add_subparsers(dest="command", required=True)
    
    # Create subcommand
    create_parser = subparsers.add_parser("create", help="Create a new Azimutt user")
    create_parser.add_argument("--email", required=True, help="User email address")
    create_parser.add_argument("--password", required=True, help="User password")
    create_parser.add_argument("--name", required=True, help="User full name")
    
    # Reset password subcommand
    reset_parser = subparsers.add_parser("reset-password", help="Reset password for existing user")
    reset_parser.add_argument("--email", required=True, help="User email address")
    reset_parser.add_argument("--password", required=True, help="New user password")

    # List subcommand
    subparsers.add_parser("list", help="List all Azimutt users")
    
    # Delete subcommand
    delete_parser = subparsers.add_parser("delete", help="Delete a user by email")
    delete_parser.add_argument("--email", required=True, help="User email address")

    # Repair subcommand
    subparsers.add_parser("repair", help="Repair user organization memberships")
    
    args = parser.parse_args()
    
    try:
        if args.command == "create":
            create_user(args.email, args.password, args.name)
        elif args.command == "reset-password":
            reset_password(args.email, args.password)
        elif args.command == "list":
            list_users()
        elif args.command == "delete":
            delete_user(args.email)
        elif args.command == "repair":
            repair()
    except Exception as e:
        print(f"❌ Error: {e}", file=sys.stderr)
        sys.exit(1)

if __name__ == "__main__":
    main()
