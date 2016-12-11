#!/bin/bash
# sudo -i -u postgres
psql -c "CREATE USER testuser WITH PASSWORD 'dev'"
psql -c "CREATE DATABASE testuserdb"
psql -c "GRANT ALL PRIVILEGES ON DATABASE testuserdb TO testuser"
