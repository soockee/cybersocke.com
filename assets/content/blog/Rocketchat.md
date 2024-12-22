---
name: "Rocket.Chat Digital Ocean Deployment"
slug: "rocketchat-dc-deployment"
tags: ["cloud", "rocketchat", "ansible", "nginx"]
date: 2020-09-06
description: "Guide to deploying Rocket.Chat on Digital Ocean with step-by-step instructions."
---

Recently, I set up a Rocket.Chat server and documented the deployment process. Leveraging the GitHub Education Pack, which provided €50 in Digital Ocean credits, I opted to deploy on a Digital Ocean droplet. Each droplet costs €5 per month, making it an affordable option for testing and learning.

Here’s a high-level overview of the deployment process:

1. Provisioning the VM (Droplet).
2. Configuring the domain and hostname.
3. Installing and setting up NGINX.
4. Deploying Rocket.Chat.
5. Enabling HTTPS with Let’s Encrypt.

## Step 1: Provisioning the VM

Digital Ocean requires an initial deposit of €5 and linking a payment method (credit card or PayPal), even for the student offer. After signing up, I provisioned a droplet by selecting an image. The choice came down to plain Ubuntu or the Ubuntu Docker image. I opted for the latter to save some provisioning effort.

The newly created droplet appears in the Digital Ocean control panel, ready for configuration.

## Step 2: Domain and Host Configuration

This part has become routine over time. I updated the domain’s nameservers through my registrar (Namecheap, in this case). Digital Ocean provides three nameservers:

- `ns1.digitalocean.com`
- `ns2.digitalocean.com`
- `ns3.digitalocean.com`

It usually takes some time for the changes to propagate. After an hour, the domain was resolvable via Digital Ocean’s nameservers.

Next, I created two **A Records**:

1. Pointing the main domain to the droplet’s IP address.
2. Pointing the `www` subdomain to the same IP address.

Once configured, I tested the setup by pinging the domain.


## Step 3: Installing and Configuring NGINX

Configuring NGINX can be challenging initially, particularly when dealing with server blocks, HTTP-to-HTTPS redirection, and certificate management.

I started with a basic HTTP configuration, followed by a self-signed certificate setup. While functional, self-signed certificates aren’t ideal for production. Later, I transitioned to Let’s Encrypt for proper certificates.

Below is the initial NGINX configuration:

```nginx
server {
    listen 80;
    server_name _;
    if ($http_x_forwarded_proto != 'https') {
        return 301 https://$host$request_uri;
    }
}

server {
    listen 443 ssl;
    server_name example.com;

    error_log /var/log/nginx/rocketchat_error.log;
    ssl_dhparam /etc/nginx/dhparams.pem;
    ssl_protocols TLSv1.2;
    ssl_ciphers 'ECDHE-RSA-AES128-GCM-SHA256:ECDHE-ECDSA-AES128-GCM-SHA256:...';
    ssl_prefer_server_ciphers on;
    ssl_session_cache shared:SSL:20m;
    ssl_session_timeout 180m;

    location / {
        proxy_pass http://example.com:3000/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $http_host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto https;
        proxy_set_header X-Nginx-Proxy true;
        proxy_redirect off;
    }
}
```

## Step 4: Deploying Rocket.Chat

Rocket.Chat provides a convenient [Docker Compose template](https://rocket.chat/docs/installation/docker-containers/). While I tested other alternatives like [Zulip](https://zulipchat.com/) and [Mattermost](https://mattermost.com/), Rocket.Chat proved the simplest to configure for my requirements.

Using the `docker-compose.yml` file, the deployment process is straightforward:

```yaml
version: '2'

services:
  rocketchat:
    image: rocketchat/rocket.chat:latest
    command: >
      bash -c "for i in `seq 1 30`; do node main.js && s=$$? && break || s=$$?; echo \"Tried $$i times. Waiting 5 secs...\"; sleep 5; done; (exit $$s)"
    restart: unless-stopped
    volumes:
      - ./uploads:/app/uploads
    environment:
      - PORT=3000
      - ROOT_URL=http://example.com:3000
      - MONGO_URL=mongodb://mongo:27017/rocketchat
      - MONGO_OPLOG_URL=mongodb://mongo:27017/local
      - MAIL_URL=smtp://smtp.email
    depends_on:
      - mongo
    ports:
      - 3000:3000

  mongo:
    image: mongo:4.0
    restart: unless-stopped
    volumes:
      - ./data/db:/data/db
    command: mongod --smallfiles --oplogSize 128 --replSet rs0 --storageEngine=mmapv1

  mongo-init-replica:
    image: mongo:4.0
    command: >
      bash -c "for i in `seq 1 30`; do mongo mongo/rocketchat --eval \"rs.initiate({ _id: 'rs0', members: [ { _id: 0, host: 'localhost:27017' } ]})\" && s=$$? && break || s=$$?; echo \"Tried $$i times. Waiting 5 secs...\"; sleep 5; done; (exit $$s)"
    depends_on:
      - mongo
```

Running `docker-compose up` starts the Rocket.Chat server.

## Step 5: Enabling HTTPS with Let’s Encrypt

The final step is enabling HTTPS using Let’s Encrypt. This process turned out to be much simpler than expected, especially when automated with Ansible.

Here’s the Ansible task for Let’s Encrypt:

```yaml
---
- name: Update and upgrade apt packages
  apt:
    upgrade: "yes"
    update_cache: yes
    cache_valid_time: 86400

- name: Install Let’s Encrypt Repository
  command: |
    curl -o- https://raw.githubusercontent.com/vinyll/certbot-install/master/install.sh | bash

- name: Obtain SSL Certificate
  command: |
    certbot --non-interactive --agree-tos --nginx -m <insert email> -d example.com -d www.example.com
```

Once complete, visiting the domain shows everything working perfectly, secured with HTTPS.

## Conclusion

This process demonstrates how to deploy Rocket.Chat on Digital Ocean efficiently. The full set of scripts and configurations can be found on the [GitHub repository](https://github.com/Soockee/rocketchat.resource.me).

