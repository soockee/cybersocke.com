---
name: "Version Control with Unreal Engine 4"
slug: "version-control-ue4"
tags: ["unreal engine", "game dev", "ansible", "nginx"]
date: 2020-09-07
description: "A guide to setting up Perforce for version control with Unreal Engine 4."
---

Version control in software development can feel as doing the tax return statement. 
The first few attempts might be overwhelming, but with practice, it becomes second nature. Unreal Engine 4 supports four version control systems:

- Git + LFS
- Perforce
- SVN
- Plastic SCM

In this post, we’ll focus on **Perforce**. Perforce offers Helix Core, a version control system designed for [large-scale development environments](https://en.wikipedia.org/wiki/Perforce#Helix_Core).

## Initial Setup and Challenges

Without prior experience, I started by following Perforce’s "Getting Started" guide, which recommends setting up a local server and connecting to it with the client.

This concept felt unusual coming from a Git background, where you’re accustomed to separate remote and local repositories. However, Perforce employs similar concepts, namely **workspaces** and **depots**.

### Local Server Issues

Setting up the local server didn’t go smoothly. While researching, I discovered a [guide](http://www.how-to-guide.online/GCPInstanceForPerforce/) for deploying a Perforce server on Google Cloud Platform (GCP). The guide provided a script to simplify the process:

```bash
#!/bin/bash
{
if [ -f /successfully_installed ]; then
    cat <<EOF > /log.txt
File found!
EOF
    exit 0
fi
}
wget http://www.how-to-guide.online/GCPInstanceForPerforce/UserInput.txt
apt-get update
apt-get upgrade -y
wget https://package.perforce.com/perforce.pubkey
wget -qO - https://package.perforce.com/perforce.pubkey | sudo apt-key add -
cat <<EOF > /etc/apt/sources.list.d/perforce.list
deb http://package.perforce.com/apt/ubuntu bionic release
EOF
apt-get update
apt-get install helix-p4d -y

/opt/perforce/sbin/configure-helix-p4d.sh < UserInput.txt

touch /successfully_installed
```

### Automating with Ansible

As an automation enthusiast, I turned to Ansible for provisioning instead of relying solely on scripts. But first, I needed a VM to work on. Leveraging the GitHub Education Pack, I accessed €100 in AWS credits and created an education account. Familiar with AWS, I chose to deploy infrastructure using **Terraform** from HashiCorp, avoiding manual setup through the AWS console.

Terraform enables defining infrastructure states in code, making deployment processes repeatable and well-documented.

## Infrastructure Setup

The repository structure includes two main components: Terraform scripts and Ansible playbooks. The workflow is managed via a Makefile, which simplifies commands and loads sensitive data as environment variables from a `.env` file, ensuring sensitive data is excluded from the repository.

### Repository Structure

```plaintext
.
├── ansible
│   ├── site.yml
│   ├── group_vars
│   └── roles
│       ├── docker
│       │   └── tasks
│       ├── gcc
│       │   └── tasks
│       ├── git
│       │   └── tasks
│       ├── perforce
│       │   ├── files
│       │   └── tasks
│       ├── python-docker
│       │   └── tasks
│       ├── python-pip3
│       │   └── tasks
└── terraform
    ├── ansible-provisioning
    │── aws-hosts
    ├── inventory.tf
    ├── main.tf
    ├── network.tf
    ├── security-groups.tf
    ├── ssh_config
    ├── sshconfig.tf
    ├── templates
    │   ├── hosts.tpl
    │   ├── ssh_config.tpl
    ├── terraform.tfstate
    ├── terraform.tfstate.backup
    ├── terraform.tfvars
    └── variables.tf
```

### Makefile Overview

The Makefile simplifies operations like initializing Terraform, deploying infrastructure, and provisioning with Ansible. It also generates an SSH configuration and AWS hosts file for dynamic environments.

```makefile
include .env
export $(shell sed 's/=.*//' .env)

ssh:= ssh -F terraform/ssh_config

terraform-init:
	cd terraform && terraform init && cd ..

terraform-plan:
	cd terraform && \
	terraform plan && \
	cd ..

terraform-apply:
	cd terraform && \
	terraform apply && \
	cd ..

terraform-destroy:
	cd terraform && \
	terraform destroy && \
	cd ..

ansible-provisioning:
	cd ansible && \
	ansible-playbook -i ../terraform/ansible-provisioning/aws-hosts site.yml && \
	cd ..

connect:
	$(ssh) perforce

move-ssh-config-to-ssh-directory:
	cd terraform && \
	mv ssh_config ~/.ssh/config && \
	cd ..
```

## Setting Up Perforce

Once the infrastructure was deployed (a t2.micro EC2 instance to minimize costs), Perforce was set up using an Ansible role based on the script mentioned earlier. An interesting aspect was configuring the server and setting up an admin account through prompts in the configuration script. The following Ansible tasks illustrate this:

```yaml
- name: Check if Perforce is already installed
  stat:
    path: ~/successfully_installed
  register: stat_result

- name: Add Perforce signing key
  become: true
  apt_key:
    url: https://package.perforce.com/perforce.pubkey
    state: present
  when: stat_result.stat.exists == false

- name: Add Perforce repository
  become: true
  apt_repository:
    repo: deb http://package.perforce.com/apt/ubuntu bionic release
    state: present
  when: stat_result.stat.exists == false

- name: Install Helix Core
  become: true
  apt:
    name: helix-p4d
    state: present
  when: stat_result.stat.exists == false

- name: Configure Helix Core
  become: true
  expect:
    command: |
      /opt/perforce/sbin/configure-helix-p4d.sh master -n -p ssl:1666 -r /opt/perforce/servers/master -u {{admin_name}} -P {{admin_pass}}
    responses:
      admin_name: "{{ admin_name }}"
      admin_pass: "{{ admin_pass }}"
    timeout: null
  when: stat_result.stat.exists == false

- name: Mark installation as successful
  file:
    path: ~/successfully_installed
    state: touch
```

## Conclusion

This post covers deploying a Perforce server with Terraform and Ansible. While this is just the deployment phase, mastering the use of Perforce and integrating it into Unreal Engine workflows is the next step. A word of caution: transferring large Unreal Engine files (about 400 MB in my case) incurred €0.07 in data processing costs. Keep an eye on those expenses!

The complete repository can be found on GitHub:

[Soockee/perforce-server-aws](https://github.com/Soockee/perforce-server-aws)

