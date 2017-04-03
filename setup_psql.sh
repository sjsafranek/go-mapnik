#!/bin/bash

sudo -u postgres psql -c "CREATE USER mapnik WITH PASSWORD 'dev'"
sudo -u postgres psql -c "CREATE DATABASE mbtiles"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE mbtiles TO mapnik"
sudo -u postgres psql -c "ALTER USER mapnik WITH SUPERUSER;"
sudo -u postgres psql -c "CREATE EXTENSION postgis; CREATE EXTENSION postgis_topology; CREATE EXTENSION fuzzystrmatch; CREATE EXTENSION postgis_tiger_geocoder;" mbtiles
