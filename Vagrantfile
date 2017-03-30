# -*- mode: ruby -*-
# vi: set ft=ruby :

# DeoDjango Vagrant Box
# https://github.com/david-wilson/vagrant-geodjango-base

# VM Customized Settings
$CPUS              = 1
$MEMORY            = 512


# Setup systemd service file
# Creates and enable systemd service
$setup_systemd = <<SCRIPT
cat > /etc/systemd/system/pomegranate.service <<-EOF
[Unit]
Description=The Pomegranate Server

[Service]
TimeoutStartSec=10
RestartSec=10
ExecStart=/usr/bin/python3 /vagrant/manage.py runserver 0.0.0.0:8080
WorkingDirectory=/vagrant
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl enable pomegranate.service
systemctl daemon-reload
systemctl start pomegranate.service
SCRIPT


# Setup database tables and models
# creates superuser
$setup_db_old = <<SCRIPT
if [ ! -f /vagrant/geo.sqlite3 ]; then
	cd /vagrant
	python3 manage.py migrate
	python3 manage.py makemigrations
	echo "from django.contrib.auth.models import User; User.objects.create_superuser('admin', 'admin@pomegranate.com', 'dev')" | python3 manage.py shell
fiSafranek
SCRIPT


$setup_db = <<SCRIPT
cd /vagrant

sudo -u postgres psql -c "CREATE USER pomegranateuser WITH PASSWORD 'dev'"
sudo -u postgres psql -c "CREATE DATABASE pomegranatedb"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE pomegranatedb TO pomegranateuser"
sudo -u postgres psql -c "ALTER USER pomegranateuser WITH SUPERUSER;"
sudo -u postgres psql -c "CREATE EXTENSION postgis; CREATE EXTENSION postgis_topology; CREATE EXTENSION fuzzystrmatch; CREATE EXTENSION postgis_tiger_geocoder;" pomegranatedb

if [ -f config.json ]; then
	cp config.json config.json_backup
fi

echo '{"default": {"ENGINE": "django.contrib.gis.db.backends.postgis","NAME": "pomegranatedb","USER": "pomegranateuser","PASSWORD": "dev","HOST": "127.0.0.1","PORT": "5432"}}' > config.json
./bootstrapper.sh

SCRIPT


# Vagrantfile API/syntax version. Don't touch unless you know what you're doing!
VAGRANTFILE_API_VERSION = "2"

#Vagrant::Config.run do |config|
Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|
	# Base box to build off, and download URL for when it doesn't exist on the user's system already
	#config.vm.box = "ubuntu/trusty64"
	#config.vm.box = "debian/jessie64"
	# "debian/jessie64" has a bug with `synced_folder` impacting guest and host sharing of `/vagrant`
	config.vm.box = "debian/contrib-jessie64"
	
	# Boot with a GUI so you can see the screen. (Default is headless)
	# config.vm.boot_mode = :gui
	
	# Assign this VM to a host only network IP, allowing you to access it
	# via the IP.
	#config.vm.network "private_network", ip: "172.20.0.10", netmask: "255.240.0.0", :mac => "08002719318B"
	
	# Forward a port from the guest to the host, which allows for outside
	# computers to access the VM, whereas host only networking does not.
	config.vm.network :forwarded_port, guest: 8080, host: 8080
	
	# Share an additional folder to the guest VM. The first argument is
	# an identifier, the second is the path on the guest to mount the
	# folder, and the third is the path on the host to the actual folder.	
	#config.vm.synced_folder ".", "/vagrant", type: "virtualbox"
	#config.vm.synced_folder ".", "/vagrant", type: "rsync"

	# Enable provisioning with a shell script.
	# sudo apt-add-repository ppa:ubuntugis/ubuntugis-unstable
	config.vm.provision "shell", inline: 'aptitude update'
	config.vm.provision "shell", inline: 'aptitude -yy install install libgeos++ binutils libproj-dev gdal-bin curl htop python3-pip python3-gdal python3-jinja2 python3-psycopg2'
	config.vm.provision "shell", inline: 'pip3 install django'
	config.vm.provision "shell", inline: 'aptitude -yy install postgresql'
	#config.vm.provision "shell", inline: 'aptitude -yy install postgis'
	config.vm.provision "shell", inline: 'aptitude -yy install postgresql-9.4-postgis-2.1'
	config.vm.provision "shell", inline: $setup_db
	config.vm.provision "shell", inline: $setup_systemd
	config.vm.provision "shell", run: "always", inline: "systemctl restart pomegranate.service"

	config.vm.provider "virtualbox" do |v|
		v.memory = $MEMORY
		v.cpus = $CPUS
	end
end
