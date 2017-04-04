# -*- mode: ruby -*-
# vi: set ft=ruby :

# VM Customized Settings
$CPUS              = 1
$MEMORY            = 512

# Setup systemd service file
# Creates and enable systemd service
$setup_systemd = <<SCRIPT
cat > /etc/systemd/system/tileserver.service <<-EOF
[Unit]
Description=The TileServer Server

[Service]
TimeoutStartSec=10
RestartSec=10
ExecStart=/vagrant/TileServer -c configs/psql8080.json
WorkingDirectory=/vagrant
Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

systemctl enable tileserver.service
systemctl daemon-reload
systemctl start tileserver.service
SCRIPT

# Setup database tables and models
$setup_db = <<SCRIPT
cd /vagrant
sudo -u postgres psql -c "CREATE USER mapnik WITH PASSWORD 'dev'"
sudo -u postgres psql -c "CREATE DATABASE mbtiles"
sudo -u postgres psql -c "GRANT ALL PRIVILEGES ON DATABASE mbtiles TO mapnik"
sudo -u postgres psql -c "ALTER USER mapnik WITH SUPERUSER;"
SCRIPT
#sudo -u postgres psql -c "CREATE EXTENSION postgis; CREATE EXTENSION postgis_topology; CREATE EXTENSION fuzzystrmatch; CREATE EXTENSION postgis_tiger_geocoder;" mbtiles

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
	config.vm.provision "shell", inline: 'aptitude -yy install curl'
	config.vm.provision "shell", inline: 'curl https://getcaddy.com | bash'
	config.vm.provision "shell", inline: 'aptitude -yy install htop'
	config.vm.provision "shell", inline: 'aptitude -yy install libmapnik-dev'
	config.vm.provision "shell", inline: 'aptitude -yy install postgresql'
	config.vm.provision "shell", inline: 'aptitude -yy install postgresql-9.4-postgis-2.1'
	# create user
	config.vm.provision "shell", inline: 'useradd -m mapnik'
	config.vm.provision "shell", inline: 'echo -e "dev\ndev" | passwd mapnik'
	config.vm.provision "shell", inline: $setup_db
	config.vm.provision "shell", inline: $setup_systemd
	config.vm.provision "shell", run: "always", inline: "systemctl restart tileserver.service"

	config.vm.provider "virtualbox" do |v|
		v.memory = $MEMORY
		v.cpus = $CPUS
	end
end
