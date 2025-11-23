# DATABASE PASSWORD - READ THIS!

## The Problem

Your **LOCAL PostgreSQL 17** is running on port 5432, but we don't know the password you set when you installed it.

## What We Tried

- ✅ Password `changeme123` - **DOESN'T WORK**
- ✅ Docker PostgreSQL - **Docker daemon not running**

## Solution - YOU NEED TO PROVIDE THE PASSWORD

### Step 1: Find Your PostgreSQL 17 Password

Your PostgreSQL 17 password was set when you installed PostgreSQL. Common options:
- Check your password manager
- Check installation notes
- It might be blank (try just pressing Enter)
- You may have set it to your Mac user password

### Step 2: Test the Connection

```bash
# Try connecting (it will prompt for password)
psql -h localhost -p 5432 -U postgres

# If that works, you have the right password!
# If not, you may need to reset it
```

### Step 3: Reset Password (If Needed)

```bash
# Edit PostgreSQL config to allow trust authentication temporarily
sudo nano /Library/PostgreSQL/17/data/pg_hba.conf

# Change this line:
# host    all             all             127.0.0.1/32            scram-sha-256
# TO:
# host    all             all             127.0.0.1/32            trust

# Restart PostgreSQL
sudo /Library/PostgreSQL/17/bin/pg_ctl restart -D /Library/PostgreSQL/17/data

# Connect without password
psql -h localhost -p 5432 -U postgres

# Set new password to changeme123
ALTER USER postgres PASSWORD 'changeme123';

# Exit psql
\q

# Change pg_hba.conf back to scram-sha-256
sudo nano /Library/PostgreSQL/17/data/pg_hba.conf

# Restart PostgreSQL
sudo /Library/PostgreSQL/17/bin/pg_ctl restart -D /Library/PostgreSQL/17/data
```

### Step 4: Update the Script

Once you know the password, edit:

```bash
nano /Users/Mounir/Task-Master/Codebase/golang-backend-service/scripts/setup-env.sh

# Change this line:
export DB_PASSWORD="changeme123"

# To your actual password:
export DB_PASSWORD="YOUR_ACTUAL_PASSWORD"
```

### Step 5: Create the Database

```bash
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
./setup-local-db.sh
# Enter your password when prompted
```

### Step 6: Start the Service

```bash
./START_SERVICE.sh
```

## OR - Use Docker (If You Can Start It)

If you can get Docker Desktop running:

```bash
# Start Docker Desktop app
# Then:
cd /Users/Mounir/Task-Master/Codebase/golang-backend-service
docker compose up -d

# Update port in scripts/setup-env.sh to 5433
# Then start service
```

## Current Status

- ✅ Code is ready
- ✅ IONOS token configured  
- ✅ PostgreSQL 17 is running
- ❌ **We need YOUR PostgreSQL password**
- ❌ Docker daemon not running (optional alternative)

---

**WRITE YOUR PASSWORD HERE FOR YOUR REFERENCE:**

PostgreSQL Password: ________________________________

(Keep this file safe and don't commit it to git!)

