# Just-In-Time SSH Access Management System

### Technical Design Document

**Author:** Manish Jangra
**Purpose:**
Build a centralized system to manage **temporary SSH access to private servers** without manually creating or removing users.

The system will provide:

* Temporary SSH access
* Approval workflow
* Automatic user creation & deletion
* Centralized audit logs
* Server discovery via tags
* Cluster / server group management

---

# 1. Problem Statement

In large infrastructures (AWS, EC2, Kubernetes nodes, databases, etc.) managing SSH access is difficult.

Typical workflow today:

1. Dev requests SSH access
2. Admin manually creates user
3. Shares SSH key
4. Later admin removes access

Problems:

* Time consuming
* Access sometimes not revoked
* No audit logs
* Difficult to manage at scale
* Security risk

---

# 2. Proposed Solution

A **Just-In-Time (JIT) SSH Access System**.

Instead of giving permanent access:

1. User requests access from **Admin Dashboard**
2. Approver reviews request
3. Once approved:

   * Agent running on server creates temporary user
4. After TTL expires:

   * Agent automatically deletes user

---

# 3. System Architecture

```text
                    +-----------------------+
                    |      Admin UI         |
                    | (React / Next.js)    |
                    +-----------+-----------+
                                |
                                |
                        HTTPS REST API
                                |
                                v
                    +-----------------------+
                    |   Control Plane API   |
                    |        (Go)           |
                    |                       |
                    |  Auth Service        |
                    |  Access Requests     |
                    |  Approval Workflow   |
                    |  Server Registry     |
                    |  Audit Logging       |
                    +-----------+-----------+
                                |
                                |
                        PostgreSQL Database
                                |
                                |
                --------------------------------
                |              |               |
                v              v               v
           +--------+     +--------+      +--------+
           | Agent  |     | Agent  |      | Agent  |
           |Server1 |     |Server2 |      |Server3 |
           +--------+     +--------+      +--------+
```

---

# 4. System Components

### 1. Admin Dashboard

Used by:

* Developers
* SRE
* Security teams
* Approvers

Features:

* Request access
* Approve access
* Manage servers
* Manage clusters
* Manage approvers
* View logs

---

### 2. Control Plane API

Main backend service.

Responsibilities:

* Manage users
* Manage access requests
* Store SSH keys
* Server registry
* Approval workflow
* Agent communication
* Audit logs

---

### 3. Agent Service

Small Go binary installed on every server.

Responsibilities:

* Register server
* Send heartbeat
* Poll tasks
* Create temporary users
* Install SSH keys
* Add sudo permissions
* Delete users after expiration

---

### 4. Database

PostgreSQL used for storing:

* Users
* Servers
* Tags
* Clusters
* Access requests
* Audit logs

---

# 5. Agent Service Design

The **agent runs on every server**.

Example:

```text
EC2
VM
Bare Metal
Kubernetes Node
```

Agent runs as:

```text
systemd service
```

Example:

```text
jit-agent.service
```

---

# 6. Agent Startup Workflow

When agent starts:

1. Collect server metadata
2. Register with control plane
3. Store agent token
4. Start polling tasks
5. Send heartbeat

---

# 7. Server Metadata Collection

Agent collects:

```text
hostname
private_ip
instance_id
region
os
tags
```

Example AWS metadata:

```bash
curl http://169.254.169.254/latest/meta-data/instance-id
```

Example data sent to API:

```http
POST /agent/register
```

```json
{
 "hostname": "prod-mongo-1",
 "private_ip": "10.0.1.12",
 "instance_id": "i-0abc123",
 "region": "ap-south-1",
 "tags": {
   "environment": "prod",
   "role": "mongodb",
   "cluster": "mongo-cluster-1"
 }
}
```

---

# 8. Agent Heartbeat

Agent sends heartbeat every **10 seconds**.

Endpoint:

```http
POST /agent/heartbeat
```

Example payload:

```json
{
 "agent_id":"agent_123",
 "hostname":"prod-db-1",
 "uptime":102342
}
```

Purpose:

* Detect online servers
* Detect offline servers

Dashboard logic:

```python
if last_seen > 30 seconds
server = offline
```

---

# 9. Access Request Workflow

User submits request via dashboard.

Example:

```text
Request SSH Access
```

Parameters:

```text
server
ssh key
duration
sudo access
reason
```

Example API request:

```http
POST /access-request
```

```json
{
 "server_id":"srv_001",
 "username":"manish",
 "pub_key":"ssh-rsa AAA",
 "duration":"1h",
 "sudo":true
}
```

---

# 10. Approval Workflow

Process:

```text
User → Request Access
      ↓
Approver receives notification
      ↓
Approver approves request
      ↓
Control Plane marks request approved
      ↓
Agent polls task
      ↓
Agent creates user
      ↓
User SSH login allowed
      ↓
TTL expires
      ↓
Agent deletes user
```

---

# 11. Agent Task Polling

Agent polls API every **10 seconds**.

```http
GET /agent/tasks?agent_id=agent123
```

Example response:

```json
[
 {
  "task_id":"task123",
  "username":"manish",
  "pubkey":"ssh-rsa AAA",
  "sudo":true,
  "expires_at":"2026-03-12T12:30:00"
 }
]
```

Agent performs:

```bash
useradd manish
```

Create ssh directory:

```bash
/home/manish/.ssh
```

Add key:

```bash
authorized_keys
```

---

# 12. User Creation

Steps:

1. Create Linux user

```bash
useradd -m manish
```

2. Create ssh directory

```bash
mkdir -p /home/manish/.ssh
```

3. Add SSH key

```bash
echo "YOUR_PUBLIC_KEY" >> /home/manish/.ssh/authorized_keys
```

4. Set permissions

```bash
chmod 700 /home/manish/.ssh
chmod 600 /home/manish/.ssh/authorized_keys
chown -R manish:manish /home/manish/.ssh
```

---

# 13. Sudo Access

If sudo required:

```bash
/etc/sudoers.d/manish
```

Example:

```bash
echo "manish ALL=(ALL) NOPASSWD:ALL" > /etc/sudoers.d/manish
chmod 440 /etc/sudoers.d/manish
```

---

# 14. User Deletion

When TTL expires:

Agent runs:

```bash
userdel -r manish
rm -f /etc/sudoers.d/manish
```

---

# 15. Allow List Users

Certain users must **never be deleted**.

Example config:

```yaml
allow_users:
 - ubuntu
 - ec2-user
 - root
 - admin
```

Agent ignores these users.

---

# 16. Server Tags

Servers can have tags like:

```text
environment=prod
role=database
cluster=mongo-cluster
```

Benefits:

* Group access control
* Easier server discovery

---

# 17. Server Groups / Clusters

Clusters represent groups of servers.

Example:

```text
mongo-cluster
redis-cluster
kafka-cluster
```

MongoDB cluster example:

```text
mongo-1
mongo-2
mongo-3
```

---

# 18. Database Schema

### Users

```text
users
-----
id
email
role
created_at
```

---

### Servers

```text
servers
-------
id
hostname
ip
instance_id
agent_id
status
last_seen
```

---

### Server Tags

```text
server_tags
------------
server_id
tag_key
tag_value
```

---

### Clusters

```text
clusters
--------
id
name
type
```

---

### Access Requests

```text
access_requests
---------------
id
user_id
server_id
pub_key
sudo
duration
status
approved_by
expires_at
```

---

### Audit Logs

```text
audit_logs
----------
id
user
server
action
timestamp
```

---

# 19. Admin Dashboard Features

### Access Requests

Users can:

* Request SSH access
* Choose server
* Select duration
* Add SSH key
* Request sudo

---

### Approval Panel

Approvers can:

* Approve request
* Reject request
* View request details

---

### Server Management

Admin can:

* View servers
* View clusters
* View tags
* Check server status

---

### User Access Logs

Admin can see:

```text
who accessed server
when access started
when access expired
```

---

# 20. Security Features

### Agent Authentication

Agent must authenticate using:

```text
agent_id
agent_token
```

Stored in:

```text
/etc/jit-agent/config.yaml
```

---

### API Security

Communication uses:

```text
HTTPS
JWT authentication
```

---

### SSH Restrictions

Disable:

```text
password authentication
```

Allow only:

```text
ssh keys
```

---

### Audit Logging

System logs:

```text
who requested access
who approved
when user logged in
when user removed
```

---

# 21. Technology Stack

### Backend

```text
Go (Gin / Fiber)
PostgreSQL
Redis (optional)
```

---

### Frontend

```text
Next.js
React
TailwindCSS
```

---

### Agent

```text
Go static binary
~10MB
```

---

### Infrastructure

```text
AWS
EC2
RDS
ALB
```

---

# 22. Deployment

Agent installed on server using script:

```bash
curl install.sh | bash
```

Script will:

1. Download binary
2. Create config
3. Install systemd service
4. Start agent

---

# 23. Future Improvements

Possible upgrades:

### WebSocket Communication

Instead of polling.

### Session Recording

Record SSH sessions.

### Slack Integration

Approvals via Slack.

### SSO Authentication

Google / Okta login.

### Command Logging

Track commands executed.

---

# 24. Similar Systems

This system is conceptually similar to:

* Teleport
* Hashicorp Boundary
* StrongDM
* AWS SSM Session Manager

---

# 25. Expected Benefits

* Remove permanent SSH access
* Improve infrastructure security
* Centralized access control
* Automated user lifecycle
* Better auditability
